package v1

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
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

	// Matches: import X from '@/path' (default import, PascalCase)
	reImportDefault = regexp.MustCompile(`import\s+([A-Z]\w+)\s+from\s*['"]([^'"]+)['"]`)

	// Matches: import X, { Y, Z } from '@/path' (mixed default + named)
	// Must run BEFORE reImportNamed/reImportDefault to avoid double-counting.
	reImportMixed = regexp.MustCompile(`import\s+([A-Za-z]\w*)\s*,\s*\{([^}]+)\}\s*from\s*['"]([^'"]+)['"]`)

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

	// Matches: const X, let X, var X, function X, class X — local declarations
	reLocalDecl = regexp.MustCompile(`(?:const|let|var|function|class)\s+([A-Z]\w+)`)
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

			// Check: default import — target file must have a default export.
			if imp.Default != "" && !exportSet["default"] {
				errors = append(errors, ValidationError{
					Severity: "error",
					File:     f.Path,
					Message:  fmt.Sprintf("default-imports %q from %q but the file has no default export", imp.Default, imp.Path),
				})
			}

			// Check: named imports — each must be exported by the target file.
			// Names are already cleaned (aliases stripped, "type " removed) by parseImports.
			for _, name := range imp.Names {
				if name == "" || name == "type" {
					continue
				}
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

	// Step 4: Check for orphaned displayName assignments (e.g. Texarea.displayName where Texarea is not defined).
	// These cause ReferenceError at module load time — the whole page crashes before React renders.
	for _, f := range files {
		if !strings.HasSuffix(f.Path, ".tsx") && !strings.HasSuffix(f.Path, ".ts") {
			continue
		}
		// Collect all locally-declared names (PascalCase only — component names)
		declared := make(map[string]bool)
		for _, m := range reLocalDecl.FindAllStringSubmatch(f.Content, -1) {
			declared[m[1]] = true
		}
		// Also treat imported names as "declared"
		for _, imp := range parseImports(f.Path, f.Content) {
			if imp.Default != "" {
				declared[imp.Default] = true
			}
			for _, n := range imp.Names {
				declared[strings.TrimSpace(n)] = true
			}
		}
		// Check every X.displayName = '...' — X must be declared
		for _, m := range reDisplayName.FindAllStringSubmatch(f.Content, -1) {
			name := m[1]
			// Skip known globals
			if name == "React" || name == "module" || name == "exports" {
				continue
			}
			if !declared[name] {
				errors = append(errors, ValidationError{
					Severity: "error",
					File:     f.Path,
					Message:  fmt.Sprintf("%s.displayName is assigned but %s is not declared in this file (likely a typo in component name)", name, name),
				})
			}
		}
	}

	// Step 5: Validate env variables.
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

// cleanImportNames normalises a raw comma-separated names string from inside { }.
// It strips TypeScript "type " prefix and "as Alias" renaming so only the
// exported identifier (the name the target file must actually export) remains.
func cleanImportNames(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		n := strings.TrimSpace(p)
		// "type X" / "type X as Y" — strip leading "type " keyword
		n = strings.TrimPrefix(n, "type ")
		n = strings.TrimSpace(n)
		// "X as Y" — we want to check the exported name X, not the local alias Y
		if idx := strings.Index(n, " as "); idx >= 0 {
			n = strings.TrimSpace(n[:idx])
		}
		if n != "" {
			out = append(out, n)
		}
	}
	return out
}

// parseImports extracts all import statements from a file.
func parseImports(filePath, content string) []ImportStatement {
	var imports []ImportStatement

	// Track ranges already consumed by reImportMixed so we don't double-count.
	mixedRanges := make([][2]int, 0)

	// Mixed imports FIRST: import Default, { X, Y } from 'path'
	for _, loc := range reImportMixed.FindAllStringSubmatchIndex(content, -1) {
		match := reImportMixed.FindStringSubmatch(content[loc[0]:loc[1]])
		if match == nil {
			continue
		}
		mixedRanges = append(mixedRanges, [2]int{loc[0], loc[1]})
		imports = append(imports, ImportStatement{
			Default:  match[1],
			Names:    cleanImportNames(match[2]),
			Path:     match[3],
			FilePath: filePath,
		})
	}

	isMixed := func(start, end int) bool {
		for _, r := range mixedRanges {
			if start >= r[0] && end <= r[1] {
				return true
			}
		}
		return false
	}

	// Named imports: import { X, Y } from 'path'
	for _, loc := range reImportNamed.FindAllStringSubmatchIndex(content, -1) {
		if isMixed(loc[0], loc[1]) {
			continue
		}
		match := reImportNamed.FindStringSubmatch(content[loc[0]:loc[1]])
		if match == nil {
			continue
		}
		imports = append(imports, ImportStatement{
			Names:    cleanImportNames(match[1]),
			Path:     match[2],
			FilePath: filePath,
		})
	}

	// Default imports: import X from 'path'
	for _, loc := range reImportDefault.FindAllStringSubmatchIndex(content, -1) {
		if isMixed(loc[0], loc[1]) {
			continue
		}
		match := reImportDefault.FindStringSubmatch(content[loc[0]:loc[1]])
		if match == nil {
			continue
		}
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
	"src/hooks/useApi":                       true,
	"src/hooks/useApi.ts":                    true,
	"src/hooks/useApi.tsx":                   true,
	"src/hooks/useAppForm":                   true,
	"src/hooks/useAppForm.ts":                true,
	"src/lib/apiUtils":                       true,
	"src/lib/apiUtils.ts":                    true,
	"src/lib/utils":                          true,
	"src/lib/utils.ts":                       true,
	"src/components/shared/AppProviders":     true,
	"src/components/shared/AppProviders.tsx": true,
	"src/config/axios":                       true,
	"src/config/axios.ts":                    true,
	"src/config/env":                         true,
	"src/config/env.ts":                      true,
	"src/config/queryClient":                 true,
	"src/config/queryClient.ts":              true,
	"src/types/common":                       true,
	"src/types/common.ts":                    true,
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

	type repairResult struct {
		file models.ProjectFile
		ok   bool
	}
	results := make(chan repairResult, len(errorsByFile))

	var wg sync.WaitGroup
	for filePath, errs := range errorsByFile {
		f, ok := fileMap[filePath]
		if !ok {
			continue
		}
		wg.Add(1)
		go func(f models.ProjectFile, errs []string) {
			defer wg.Done()
			p.emitter().Emit(SSEEvent{Type: EvRepair, Message: "Исправляю: " + f.Path, Percent: 86})
			fixed, err := p.repairSingleFile(ctx, f, errs, exportRegistry)
			if err != nil {
				log.Printf("[repair] ⚠️ failed to repair %s: %v", f.Path, err)
				results <- repairResult{ok: false}
				return
			}
			log.Printf("[repair] ✅ repaired %s", f.Path)
			results <- repairResult{file: fixed, ok: true}
		}(f, errs)
	}

	wg.Wait()
	close(results)

	var repaired []models.ProjectFile
	for r := range results {
		if r.ok {
			repaired = append(repaired, r.file)
		}
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

	sb.WriteString("Fix the TypeScript/TSX file below. It has the following errors:\n\n")
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
	sb.WriteString("  - Fix ONLY the listed errors. Do not rewrite unrelated code.\n")
	sb.WriteString("  - For import errors: use correct exported names from the AVAILABLE EXPORTS list above.\n")
	sb.WriteString("  - For 'X.displayName assigned but X not declared': it is a typo in the component name — rename the const/variable to match the displayName assignment, or fix the displayName to match the const name.\n")
	sb.WriteString("  - Output the complete corrected file. Never truncate.\n")

	// Inject types.ts content so Haiku can fix TypeScript type mismatches accurately.
	if f.Path != "src/types.ts" {
		for path, exports := range exportRegistry {
			if path == "src/types.ts" && len(exports) > 0 {
				// Find actual file content from the fileMap in the caller
				sb.WriteString("\nTYPES REFERENCE (src/types.ts — entity interfaces for accurate type fixing):\n")
				sb.WriteString("  Available types: [")
				names := make([]string, 0, len(exports))
				for name := range exports {
					if name != "default" {
						names = append(names, name)
					}
				}
				sb.WriteString(strings.Join(names, ", "))
				sb.WriteString("]\n")
				break
			}
		}
	}

	fmt.Fprintf(&sb, "\nFILE: %s\n```typescript\n%s\n```\n", f.Path, f.Content)

	fixed, err := callWithTool[repairFileResult](
		p, ctx,
		models.AnthropicToolRequest{
			Model:      p.baseConf.ClaudeHaikuModel,
			MaxTokens:  16000,
			System:     "You are a TypeScript error repair bot. Fix only the listed errors (import errors, typos in component names, orphaned displayName assignments). Output the complete corrected file via the repair_file tool.",
			Messages:   []models.ChatMessage{{Role: "user", Content: []models.ContentBlock{{Type: "text", Text: sb.String()}}}},
			Tools:      []models.ClaudeFunctionTool{helper.ToolRepairFile},
			ToolChoice: helper.ForcedTool(helper.ToolRepairFile.Name),
		},
		90*time.Second,
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

// buildUIKitAPISummary extracts a compact API reference from generated UI Kit files
// (both ui/* primitives and components/shared/* composite patterns).
// Injected into feature chunk prompts so they know exact component APIs and variant values.
func buildUIKitAPISummary(uiKitFiles []models.ProjectFile) string {
	if len(uiKitFiles) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("====================================\n")
	sb.WriteString("UI KIT + SHARED PATTERNS — API REFERENCE\n")
	sb.WriteString("====================================\n")
	sb.WriteString("Already generated. Use EXACTLY these names, props, and variant values.\n\n")

	reInterface := regexp.MustCompile(`(?m)export\s+(?:interface|type)\s+(\w+(?:Props|Column|State))\s*(?:[<{]|extends)`)
	reVariants := regexp.MustCompile(`(?m)export\s+const\s+(\w+Variants)\s*=`)
	// Extracts variant KEYS from cva variant blocks: variant: { default: '...', outline: '...' }
	reVariantBlock := regexp.MustCompile(`(?s)variants\s*:\s*\{(.+?)\}\s*,?\s*defaultVariants`)
	reVariantEntry := regexp.MustCompile(`(?m)^\s*(\w+)\s*:\s*\{([^}]+)\}`)
	reVariantKeys := regexp.MustCompile(`(?m)^\s*(\w+)\s*:`)

	for _, f := range uiKitFiles {
		var exports []string
		for _, match := range reExportNamed.FindAllStringSubmatch(f.Content, -1) {
			exports = append(exports, match[1])
		}
		for _, match := range reExportBraces.FindAllStringSubmatch(f.Content, -1) {
			for _, name := range strings.Split(match[1], ",") {
				if n := strings.TrimSpace(name); n != "" {
					exports = append(exports, n)
				}
			}
		}
		if len(exports) == 0 {
			continue
		}

		fmt.Fprintf(&sb, "### %s\n", f.Path)
		fmt.Fprintf(&sb, "  Exports: [%s]\n", strings.Join(exports, ", "))

		for _, match := range reInterface.FindAllStringSubmatch(f.Content, -1) {
			fmt.Fprintf(&sb, "  Props: %s\n", match[1])
		}

		// Show variant definitions with actual key values so chunks use correct variant names.
		for _, varMatch := range reVariants.FindAllStringSubmatch(f.Content, -1) {
			fmt.Fprintf(&sb, "  Variants const: %s\n", varMatch[1])
		}
		if blockMatch := reVariantBlock.FindStringSubmatch(f.Content); len(blockMatch) > 1 {
			for _, entryMatch := range reVariantEntry.FindAllStringSubmatch(blockMatch[1], -1) {
				variantName := entryMatch[1]
				var keys []string
				for _, keyMatch := range reVariantKeys.FindAllStringSubmatch(entryMatch[2], -1) {
					keys = append(keys, keyMatch[1])
				}
				if len(keys) > 0 {
					fmt.Fprintf(&sb, "  %s values: [%s]\n", variantName, strings.Join(keys, ", "))
				}
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// lucideValidIcons is the verified safe list for lucide-react@0.441.0.
var lucideValidIcons = map[string]bool{
	// Navigation
	"Home": true, "LayoutDashboard": true, "LayoutGrid": true, "Menu": true, "PanelLeft": true, "Sidebar": true,
	// Users
	"User": true, "Users": true, "UserPlus": true, "UserCheck": true, "UserX": true,
	"UserCircle": true, "Building": true, "Building2": true, "Briefcase": true,
	// CRUD
	"Plus": true, "Pencil": true, "Trash": true, "Trash2": true, "Edit": true,
	"Save": true, "Copy": true, "Eye": true, "EyeOff": true, "Download": true,
	"Upload": true, "Send": true, "RefreshCw": true,
	// Arrows
	"ArrowLeft": true, "ArrowRight": true, "ArrowUp": true, "ArrowDown": true,
	"ChevronLeft": true, "ChevronRight": true, "ChevronDown": true, "ChevronUp": true,
	"ChevronsLeft": true, "ChevronsRight": true, "ExternalLink": true,
	// Search
	"Search": true, "Filter": true, "SlidersHorizontal": true, "ListFilter": true,
	// Status
	"Check": true, "CheckCircle": true, "CheckCircle2": true, "X": true,
	"XCircle": true, "AlertCircle": true, "AlertTriangle": true, "Info": true,
	"Bell": true, "BellRing": true,
	// Charts
	"BarChart": true, "BarChart2": true, "BarChart3": true, "LineChart": true,
	"PieChart": true, "TrendingUp": true, "TrendingDown": true, "Activity": true,
	// Files
	"File": true, "FileText": true, "FileCheck": true, "FilePlus": true,
	"Folder": true, "FolderOpen": true, "Paperclip": true, "BookOpen": true,
	// Time
	"Calendar": true, "CalendarDays": true, "Clock": true, "Timer": true,
	// Money
	"DollarSign": true, "CreditCard": true, "Wallet": true, "Receipt": true,
	"ShoppingCart": true, "Package": true, "Banknote": true,
	// Settings
	"Settings": true, "Settings2": true, "Wrench": true, "Key": true,
	"Lock": true, "Shield": true, "ShieldCheck": true,
	// UI
	"MoreHorizontal": true, "MoreVertical": true, "Maximize": true, "Minimize": true,
	"ZoomIn": true, "ZoomOut": true, "Move": true, "GripVertical": true,
	// Misc
	"Star": true, "Tag": true, "Hash": true, "Globe": true, "MapPin": true,
	"Database": true, "Server": true, "Loader2": true, "Sun": true, "Moon": true,
	"Image": true, "Zap": true, "Flame": true, "Sparkles": true, "Target": true,
	"Award": true, "ThumbsUp": true, "Phone": true, "Mail": true,
	"Truck": true, "Layers": true, "Layout": true, "Code": true, "Code2": true,
	"Terminal": true, "Cpu": true, "Wifi": true, "Link": true, "Link2": true,
	"Unlink": true, "RefreshCcw": true, "RotateCcw": true, "RotateCw": true,
	"LogOut": true, "LogIn": true, "Grid": true, "List": true, "Table": true,
	"Columns": true, "Rows": true, "LayoutList": true, "SquareStack": true,
	"Inbox": true, "MessageSquare": true, "MessageCircle": true, "HelpCircle": true,
	"PlayCircle": true, "StopCircle": true, "PauseCircle": true,
	"Volume2": true, "VolumeX": true, "Mic": true, "MicOff": true,
	"Video": true, "VideoOff": true, "Camera": true,
	"Lightbulb": true, "Compass": true, "Navigation": true, "Map": true,
	"Flag": true, "Bookmark": true, "Heart": true, "HeartOff": true,
}

// lucideFallbacks maps known non-existent icon names to safe alternatives.
var lucideFallbacks = map[string]string{
	"LayoutKanban":     "LayoutGrid",
	"Kanban":           "LayoutGrid",
	"KanbanSquare":     "LayoutGrid",
	"LayoutColumns":    "Columns",
	"LayoutRows":       "Rows",
	"TableProperties":  "Table",
	"TableCellsMerge":  "Table",
	"UserCog":          "Settings",
	"UserSettings":     "Settings",
	"Users2":           "Users",
	"UsersRound":       "Users",
	"PersonStanding":   "User",
	"Contact":          "User",
	"ContactRound":     "User",
	"BadgeCheck":       "CheckCircle",
	"BadgeAlert":       "AlertCircle",
	"CircleCheck":      "CheckCircle",
	"CircleX":          "XCircle",
	"CircleAlert":      "AlertCircle",
	"OctagonAlert":     "AlertTriangle",
	"TriangleAlert":    "AlertTriangle",
	"ShoppingBag":      "ShoppingCart",
	"Store":            "ShoppingCart",
	"PackageOpen":      "Package",
	"PackageCheck":     "Package",
	"PackagePlus":      "Package",
	"PackageSearch":    "Package",
	"PenLine":          "Pencil",
	"PenSquare":        "Edit",
	"PencilLine":       "Pencil",
	"FilePen":          "FileText",
	"FileEdit":         "FileText",
	"FileSearch":       "FileText",
	"FileSpreadsheet":  "FileText",
	"FileJson":         "FileText",
	"FileCode":         "Code2",
	"FolderPlus":       "FolderOpen",
	"FolderSync":       "FolderOpen",
	"BookMarked":       "BookOpen",
	"BookCopy":         "BookOpen",
	"CalendarCheck":    "Calendar",
	"CalendarClock":    "Calendar",
	"CalendarPlus":     "Calendar",
	"CalendarRange":    "CalendarDays",
	"CalendarX":        "Calendar",
	"ClockAlert":       "Clock",
	"Hourglass":        "Timer",
	"TimerOff":         "Timer",
	"Banknote":         "DollarSign",
	"PiggyBank":        "Wallet",
	"Coins":            "DollarSign",
	"HandCoins":        "DollarSign",
	"BadgeDollarSign":  "DollarSign",
	"ShieldAlert":      "AlertTriangle",
	"ShieldOff":        "Shield",
	"ShieldPlus":       "ShieldCheck",
	"LockOpen":         "Lock",
	"LockKeyhole":      "Key",
	"Fingerprint":      "Key",
	"ScanLine":         "Search",
	"QrCode":           "Hash",
	"Barcode":          "Hash",
	"NotepadText":      "FileText",
	"ClipboardList":    "FileText",
	"ClipboardCheck":   "FileCheck",
	"ClipboardPlus":    "FilePlus",
	"Clipboard":        "FileText",
	"ScrollText":       "FileText",
	"NotebookText":     "FileText",
	"WandSparkles":     "Sparkles",
	"BrainCircuit":     "Cpu",
	"BrainCog":         "Cpu",
	"Brain":            "Cpu",
	"Bot":              "Cpu",
	"Headphones":       "Volume2",
	"Speaker":          "Volume2",
	"Radio":            "Wifi",
	"Satellite":        "Wifi",
	"NetworkIcon":      "Server",
	"Network":          "Server",
	"HardDrive":        "Database",
	"HardDriveUpload":  "Upload",
	"CloudUpload":      "Upload",
	"CloudDownload":    "Download",
	"Cloud":            "Globe",
	"CloudOff":         "Globe",
	"Globe2":           "Globe",
	"Earth":            "Globe",
	"Map":              "MapPin",
	"Locate":           "MapPin",
	"MapPinOff":        "MapPin",
	"RouteOff":         "Navigation",
	"Route":            "Navigation",
	"Gauge":            "Activity",
	"GaugeCircle":      "Activity",
	"AreaChart":        "BarChart3",
	"ScatterChart":     "BarChart3",
	"CandlestickChart": "BarChart3",
	"ChartBar":         "BarChart3",
	"ChartLine":        "LineChart",
	"ChartPie":         "PieChart",
}

var lucideImportRe = regexp.MustCompile(`import\s*\{([^}]+)\}\s*from\s*['"]lucide-react['"]`)

// fixLucideImports scans all files and replaces invalid lucide-react icon names
// with verified alternatives. Modifies files in-place.
func fixLucideImports(files []models.ProjectFile) int {
	fixed := 0
	for i, f := range files {
		if !strings.HasSuffix(f.Path, ".tsx") && !strings.HasSuffix(f.Path, ".ts") {
			continue
		}
		updated, count := fixLucideInContent(f.Content)
		if count > 0 {
			files[i].Content = updated
			fixed += count
			log.Printf("[lucide-fix] %s: replaced %d invalid icon(s)", f.Path, count)
		}
	}
	return fixed
}

func fixLucideInContent(content string) (string, int) {
	totalFixed := 0
	result := lucideImportRe.ReplaceAllStringFunc(content, func(match string) string {
		sub := lucideImportRe.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		rawNames := sub[1]
		parts := strings.Split(rawNames, ",")
		changed := false
		for j, p := range parts {
			trimmed := strings.TrimSpace(p)
			// handle aliased imports: "LayoutKanban as KanbanIcon"
			name := trimmed
			alias := ""
			if idx := strings.Index(trimmed, " as "); idx != -1 {
				name = strings.TrimSpace(trimmed[:idx])
				alias = trimmed[idx:]
			}
			replacement, bad := lucideFallbacks[name]
			if !bad {
				if !lucideValidIcons[name] && name != "" {
					replacement = "LayoutGrid" // generic fallback for anything unknown
					bad = true
				}
			}
			if bad {
				parts[j] = " " + replacement + alias
				// also replace usages of the old name in the file body (only if no alias)
				if alias == "" && name != replacement {
					content = strings.ReplaceAll(content, name, replacement)
				}
				changed = true
				totalFixed++
			}
		}
		if !changed {
			return match
		}
		return "import {" + strings.Join(parts, ",") + "} from 'lucide-react'"
	})
	return result, totalFixed
}
