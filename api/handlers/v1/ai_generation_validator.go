package v1

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"ucode/ucode_go_api_gateway/api/handlers/helper"
	"ucode/ucode_go_api_gateway/api/models"
)

// ============================================================================
// POST-GENERATION IMPORT/EXPORT VALIDATOR
//
// Scans all generated files after merge and detects:
// 1. Imports from files that don't exist in the output
// 2. Named imports that aren't exported by the target file
// 3. Env variables referenced in code but missing from .env
//
// This catches every class of error we've encountered:
// - Missing exports (Badge.tsx lost QuoteStatusBadge)
// - API renames (TabList → TabsList)
// - Missing files (import from non-existent path)
// ============================================================================

// Compiled regexps for import/export parsing — built once at startup.
var (
	// Matches: import { X, Y, Z } from '@/path' or './path' or '../path'
	reImportNamed = regexp.MustCompile(`import\s*\{([^}]+)\}\s*from\s*['"]([^'"]+)['"]`)

	// Matches: import X from '@/path' (default import)
	reImportDefault = regexp.MustCompile(`import\s+([A-Z]\w+)\s+from\s*['"]([^'"]+)['"]`)

	// Matches: export function X, export const X, export class X, export type X, export interface X
	reExportNamed = regexp.MustCompile(`export\s+(?:function|const|let|var|class|type|interface|enum)\s+(\w+)`)

	// Matches: export { X, Y, Z }
	reExportBraces = regexp.MustCompile(`export\s*\{([^}]+)\}`)

	// Matches: export default function X or export default class X
	reExportDefault = regexp.MustCompile(`export\s+default\s+(?:function|class)\s+(\w+)`)

	// Matches: X.displayName = 'X' pattern (React.forwardRef components)
	reDisplayName = regexp.MustCompile(`(\w+)\.displayName\s*=`)

	// Matches: import.meta.env.VITE_XXX
	reEnvUsage = regexp.MustCompile(`import\.meta\.env\.(\w+)`)
)

// ImportStatement represents one parsed import.
type ImportStatement struct {
	Names    []string // named imports: {A, B, C}
	Default  string   // default import name (if any)
	Path     string   // import path: '@/components/ui/Button'
	FilePath string   // source file that has this import
}

// ValidationError is one detected issue in the generated code.
type ValidationError struct {
	Severity string // "error" or "warning"
	File     string // file where the issue was found
	Message  string
}

// validateGeneratedProject scans all merged files for import/export mismatches
// and env variable inconsistencies. Returns a list of validation errors.
//
// Call this after mergeChunks() and before publishing.
func validateGeneratedProject(files []models.ProjectFile, envVars map[string]any) []ValidationError {
	var errors []ValidationError

	// Step 1: Build export registry — path → set of exported names.
	exportRegistry := buildExportRegistry(files)

	// Step 2: Build file path set for existence checks.
	fileSet := make(map[string]bool, len(files))
	for _, f := range files {
		fileSet[f.Path] = true
	}

	// Step 3: Scan all files for imports and validate them.
	for _, f := range files {
		imports := parseImports(f.Path, f.Content)
		for _, imp := range imports {
			if isNPMImport(imp.Path) {
				continue
			}

			resolvedPath := resolveImportPath(f.Path, imp.Path)
			if resolvedPath == "" {
				continue // couldn't resolve — skip
			}

			// Check: does the target file exist?
			exportSet, exists := exportRegistry[resolvedPath]
			if !exists {
				// Try common alternatives (.tsx, .ts, /index.tsx, /index.ts)
				found := false
				for _, alt := range resolveAlternatives(resolvedPath) {
					if _, altExists := exportRegistry[alt]; altExists {
						found = true
						resolvedPath = alt
						exportSet = exportRegistry[alt]
						break
					}
				}
				if !found {
					// Skip template files (hooks/useApi, lib/apiUtils, etc.)
					if isTemplateFile(resolvedPath) {
						continue
					}
					errors = append(errors, ValidationError{
						Severity: "error",
						File:     f.Path,
						Message:  fmt.Sprintf("imports from %q but file does not exist in generated output", imp.Path),
					})
					continue
				}
			}

			// Check: are all named imports actually exported?
			for _, name := range imp.Names {
				name = strings.TrimSpace(name)
				if name == "" || name == "type" {
					continue
				}
				// Strip "type " prefix from type imports
				name = strings.TrimPrefix(name, "type ")
				name = strings.TrimSpace(name)

				if !exportSet[name] {
					errors = append(errors, ValidationError{
						Severity: "error",
						File:     f.Path,
						Message:  fmt.Sprintf("imports {%s} from %q but it is not exported", name, imp.Path),
					})
				}
			}
		}
	}

	// Step 4: Validate env variables.
	envErrors := validateEnvVars(files, envVars)
	errors = append(errors, envErrors...)

	return errors
}

// buildExportRegistry scans all files and returns a map of path → exported names.
func buildExportRegistry(files []models.ProjectFile) map[string]map[string]bool {
	registry := make(map[string]map[string]bool, len(files))

	for _, f := range files {
		exports := make(map[string]bool)

		// export function X / export const X / export class X / export type X / export interface X
		for _, match := range reExportNamed.FindAllStringSubmatch(f.Content, -1) {
			exports[match[1]] = true
		}

		// export { X, Y, Z }
		for _, match := range reExportBraces.FindAllStringSubmatch(f.Content, -1) {
			for _, name := range strings.Split(match[1], ",") {
				name = strings.TrimSpace(name)
				// Handle "X as Y" — export the aliased name
				if parts := strings.SplitN(name, " as ", 2); len(parts) == 2 {
					exports[strings.TrimSpace(parts[1])] = true
				} else {
					exports[name] = true
				}
			}
		}

		// export default function X / export default class X
		for _, match := range reExportDefault.FindAllStringSubmatch(f.Content, -1) {
			exports[match[1]] = true
			exports["default"] = true
		}

		// X.displayName = 'X' — React.forwardRef pattern
		for _, match := range reDisplayName.FindAllStringSubmatch(f.Content, -1) {
			exports[match[1]] = true
		}

		// export default X (simple)
		if strings.Contains(f.Content, "export default") {
			exports["default"] = true
		}

		registry[f.Path] = exports
	}

	return registry
}

// parseImports extracts all import statements from a file.
func parseImports(filePath, content string) []ImportStatement {
	var imports []ImportStatement

	// Named imports: import { X, Y } from 'path'
	for _, match := range reImportNamed.FindAllStringSubmatch(content, -1) {
		names := strings.Split(match[1], ",")
		cleaned := make([]string, 0, len(names))
		for _, n := range names {
			n = strings.TrimSpace(n)
			if n != "" {
				cleaned = append(cleaned, n)
			}
		}
		imports = append(imports, ImportStatement{
			Names:    cleaned,
			Path:     match[2],
			FilePath: filePath,
		})
	}

	// Default imports: import X from 'path'
	for _, match := range reImportDefault.FindAllStringSubmatch(content, -1) {
		imports = append(imports, ImportStatement{
			Default:  match[1],
			Path:     match[2],
			FilePath: filePath,
		})
	}

	return imports
}

// resolveImportPath converts an import path to a file path relative to project root.
// @/components/ui/Button → src/components/ui/Button
// ./utils → (resolved relative to importer)
func resolveImportPath(importerPath, importPath string) string {
	// @/ alias → src/
	if strings.HasPrefix(importPath, "@/") {
		return "src/" + strings.TrimPrefix(importPath, "@/")
	}

	// Relative imports
	if strings.HasPrefix(importPath, "./") || strings.HasPrefix(importPath, "../") {
		dir := importerPath
		if idx := strings.LastIndex(dir, "/"); idx >= 0 {
			dir = dir[:idx]
		} else {
			dir = ""
		}

		parts := strings.Split(importPath, "/")
		for _, part := range parts {
			switch part {
			case ".":
				// stay
			case "..":
				if idx := strings.LastIndex(dir, "/"); idx >= 0 {
					dir = dir[:idx]
				} else {
					dir = ""
				}
			default:
				if dir == "" {
					dir = part
				} else {
					dir = dir + "/" + part
				}
			}
		}
		return dir
	}

	return "" // npm or unresolvable
}

// resolveAlternatives returns possible file paths for an import
// (TypeScript resolves .tsx, .ts, /index.tsx, /index.ts automatically).
func resolveAlternatives(path string) []string {
	return []string{
		path + ".tsx",
		path + ".ts",
		path + "/index.tsx",
		path + "/index.ts",
	}
}

// isNPMImport returns true for imports from node_modules (no ./ or @/ prefix).
func isNPMImport(path string) bool {
	if strings.HasPrefix(path, "./") || strings.HasPrefix(path, "../") || strings.HasPrefix(path, "@/") {
		return false
	}
	// Scoped npm packages: @radix-ui/*, @tanstack/*, etc.
	if strings.HasPrefix(path, "@") && !strings.HasPrefix(path, "@/") {
		return true
	}
	return true
}

// isTemplateFile returns true for files that exist in the pre-built template
// (not generated by AI, so they won't appear in the files list).
var templateFilePaths = map[string]bool{
	"src/hooks/useApi":                    true,
	"src/hooks/useApi.ts":                 true,
	"src/hooks/useApi.tsx":                true,
	"src/lib/apiUtils":                    true,
	"src/lib/apiUtils.ts":                 true,
	"src/lib/utils":                       true,
	"src/lib/utils.ts":                    true,
	"src/components/shared/AppProviders":  true,
	"src/components/shared/AppProviders.tsx": true,
	"src/config/axios":                    true,
	"src/config/axios.ts":                 true,
}

func isTemplateFile(path string) bool {
	return templateFilePaths[path]
}

// validateEnvVars checks that env variables used in code are defined.
func validateEnvVars(files []models.ProjectFile, envVars map[string]any) []ValidationError {
	var errors []ValidationError

	// Collect all env vars used in code.
	usedVars := make(map[string]string) // var name → first file that uses it
	for _, f := range files {
		for _, match := range reEnvUsage.FindAllStringSubmatch(f.Content, -1) {
			varName := match[1]
			if _, exists := usedVars[varName]; !exists {
				usedVars[varName] = f.Path
			}
		}
	}

	// Check against provided env vars.
	for varName, firstFile := range usedVars {
		if _, defined := envVars[varName]; !defined {
			// Check .env files in the output
			found := false
			for _, f := range files {
				if strings.HasSuffix(f.Path, ".env") || strings.HasSuffix(f.Path, ".env.production") {
					if strings.Contains(f.Content, varName+"=") {
						found = true
						break
					}
				}
			}
			if !found {
				errors = append(errors, ValidationError{
					Severity: "warning",
					File:     firstFile,
					Message:  fmt.Sprintf("uses import.meta.env.%s but it is not defined in env vars or .env files", varName),
				})
			}
		}
	}

	return errors
}

// logValidationResults logs all validation errors and returns counts.
func logValidationResults(errors []ValidationError) (errorCount, warningCount int) {
	for _, e := range errors {
		switch e.Severity {
		case "error":
			errorCount++
			log.Printf("[VALIDATE] ❌ %s: %s", e.File, e.Message)
		case "warning":
			warningCount++
			log.Printf("[VALIDATE] ⚠️  %s: %s", e.File, e.Message)
		}
	}
	if errorCount == 0 && warningCount == 0 {
		log.Printf("[VALIDATE] ✅ All imports/exports verified — 0 issues")
	} else {
		log.Printf("[VALIDATE] Summary: %d errors, %d warnings", errorCount, warningCount)
	}
	return
}

// ============================================================================
// REPAIR LOOP
//
// If validateGeneratedProject returns errors, repairBrokenFiles sends each
// broken file to Haiku with a targeted prompt showing exactly what's wrong
// and what exports are available in the referenced files.
//
// maxDeployErrors: if error count exceeds this threshold the caller should
// refuse deployment entirely (too broken to be useful).
// ============================================================================

// repairFileResult is the tool-use response from Haiku for a single file fix.
type repairFileResult struct {
	Content string `json:"content"`
}

// repairBrokenFiles attempts to fix all files that have validation errors.
// Returns a slice of repaired files (only those that were successfully fixed).
// The caller is responsible for patching these back into the merged file list.
func (p *ChatProcessor) repairBrokenFiles(ctx context.Context, files []models.ProjectFile, validationErrors []ValidationError) []models.ProjectFile {
	exportRegistry := buildExportRegistry(files)

	// Group errors by file path.
	errorsByFile := make(map[string][]string)
	for _, e := range validationErrors {
		if e.Severity == "error" {
			errorsByFile[e.File] = append(errorsByFile[e.File], e.Message)
		}
	}

	// Build path → file index for fast lookup.
	fileMap := make(map[string]models.ProjectFile, len(files))
	for _, f := range files {
		fileMap[f.Path] = f
	}

	var repaired []models.ProjectFile
	for filePath, errs := range errorsByFile {
		f, ok := fileMap[filePath]
		if !ok {
			continue
		}
		fixed, err := p.repairSingleFile(ctx, f, errs, exportRegistry)
		if err != nil {
			log.Printf("[repair] ⚠️ failed to repair %s: %v", filePath, err)
			continue
		}
		log.Printf("[repair] ✅ repaired %s", filePath)
		repaired = append(repaired, fixed)
	}
	return repaired
}

// repairSingleFile sends one broken file to Haiku and returns the fixed version.
func (p *ChatProcessor) repairSingleFile(
	ctx context.Context,
	f models.ProjectFile,
	errs []string,
	exportRegistry map[string]map[string]bool,
) (models.ProjectFile, error) {
	var sb strings.Builder

	sb.WriteString("Fix the TypeScript file below. It has the following import errors:\n\n")
	for _, e := range errs {
		fmt.Fprintf(&sb, "  - %s\n", e)
	}

	// Inject available exports from target files so Haiku knows what's actually there.
	sb.WriteString("\nAVAILABLE EXPORTS in the referenced files (use ONLY these names):\n")
	imports := parseImports(f.Path, f.Content)
	seen := make(map[string]bool)
	for _, imp := range imports {
		resolved := resolveImportPath(f.Path, imp.Path)
		if resolved == "" || isNPMImport(imp.Path) {
			continue
		}
		for _, alt := range append([]string{resolved}, resolveAlternatives(resolved)...) {
			if exports, ok := exportRegistry[alt]; ok && !seen[alt] {
				seen[alt] = true
				names := make([]string, 0, len(exports))
				for name := range exports {
					names = append(names, name)
				}
				fmt.Fprintf(&sb, "  %s → [%s]\n", imp.Path, strings.Join(names, ", "))
				break
			}
		}
	}

	sb.WriteString("\nRULES:\n")
	sb.WriteString("  - Fix ONLY the broken imports. Do not rewrite unrelated code.\n")
	sb.WriteString("  - If a named import does not exist, remove it or replace with the correct name.\n")
	sb.WriteString("  - Output the complete corrected file. Never truncate.\n")

	fmt.Fprintf(&sb, "\nFILE: %s\n```typescript\n%s\n```\n", f.Path, f.Content)

	fixed, err := callWithTool[repairFileResult](
		p, ctx,
		models.AnthropicToolRequest{
			Model:      p.baseConf.ClaudeHaikuModel,
			MaxTokens:  8000,
			System:     "You are a TypeScript import-error repair bot. Fix only the import errors listed. Output the complete corrected file via the repair_file tool.",
			Messages:   []models.ChatMessage{{Role: "user", Content: []models.ContentBlock{{Type: "text", Text: sb.String()}}}},
			Tools:      []models.ClaudeFunctionTool{helper.ToolRepairFile},
			ToolChoice: helper.ForcedTool(helper.ToolRepairFile.Name),
		},
		60*time.Second,
		fmt.Sprintf("Repairing %s", f.Path),
	)
	if err != nil {
		return models.ProjectFile{}, err
	}
	if fixed.Content == "" {
		return models.ProjectFile{}, fmt.Errorf("repair returned empty content")
	}
	return models.ProjectFile{Path: f.Path, Content: fixed.Content}, nil
}

// applyRepairs patches repaired file contents back into the file list in-place.
func applyRepairs(files []models.ProjectFile, repaired []models.ProjectFile) {
	patchMap := make(map[string]string, len(repaired))
	for _, f := range repaired {
		patchMap[f.Path] = f.Content
	}
	for i := range files {
		if newContent, ok := patchMap[files[i].Path]; ok {
			files[i].Content = newContent
		}
	}
}

// buildUIKitAPISummary extracts a compact API reference from generated UI Kit files.
// This is injected into feature chunk prompts so they know exact component APIs.
func buildUIKitAPISummary(uiKitFiles []models.ProjectFile) string {
	if len(uiKitFiles) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("====================================\n")
	sb.WriteString("UI KIT — COMPONENT API REFERENCE\n")
	sb.WriteString("====================================\n")
	sb.WriteString("These components are already generated. Use EXACTLY these names and props.\n\n")

	// Regex for extracting interface/type definitions
	reInterface := regexp.MustCompile(`(?m)export\s+(?:interface|type)\s+(\w+Props)\s*(?:extends\s+[^{]+)?\{`)
	reVariants := regexp.MustCompile(`(?m)export\s+const\s+(\w+Variants)\s*=`)

	for _, f := range uiKitFiles {
		// Get component name from file
		fileName := f.Path
		if idx := strings.LastIndex(fileName, "/"); idx >= 0 {
			fileName = fileName[idx+1:]
		}

		// Find exported names
		var exports []string
		for _, match := range reExportNamed.FindAllStringSubmatch(f.Content, -1) {
			exports = append(exports, match[1])
		}
		for _, match := range reExportBraces.FindAllStringSubmatch(f.Content, -1) {
			for _, name := range strings.Split(match[1], ",") {
				exports = append(exports, strings.TrimSpace(name))
			}
		}

		if len(exports) == 0 {
			continue
		}

		fmt.Fprintf(&sb, "### %s\n", f.Path)
		fmt.Fprintf(&sb, "  Exports: [%s]\n", strings.Join(exports, ", "))

		// Show Props interfaces
		for _, match := range reInterface.FindAllStringSubmatch(f.Content, -1) {
			fmt.Fprintf(&sb, "  Props: %s\n", match[1])
		}

		// Show variant definitions
		for _, match := range reVariants.FindAllStringSubmatch(f.Content, -1) {
			fmt.Fprintf(&sb, "  Variants: %s (exported)\n", match[1])
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
