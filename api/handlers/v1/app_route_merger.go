package v1

import (
	"fmt"
	"log"
	"strings"

	"ucode/ucode_go_api_gateway/api/models"
)

// resolvedRoute is a manifest route that has been verified against the
// generated file set: the file exists and the named export is present.
type resolvedRoute struct {
	Path       string
	PageName   string
	FilePath   string
	ImportFrom string // value used inside lazy(import('...')), without .tsx, with @/ prefix
}

// stripPageTsx drops `src/Page.tsx` from the file list. The model sometimes
// invents it as a microfrontend entry with MemoryRouter, which conflicts with
// the federation contract (`./App` → `./src/App.tsx`) and steals `import './index.css'`.
func stripPageTsx(files []models.ProjectFile) []models.ProjectFile {
	out := files[:0]
	for _, f := range files {
		if f.Path == "src/Page.tsx" {
			log.Printf("[merge-routes] dropping stray src/Page.tsx (federation entry is src/App.tsx)")
			continue
		}
		out = append(out, f)
	}
	return out
}

const pageLoaderSource = `export function PageLoader() {
  return (
    <div className="flex h-screen w-full items-center justify-center">
      <div className="h-10 w-10 animate-spin rounded-full border-4 border-muted border-t-primary" />
    </div>
  );
}
`

// mergeAppRoutes deterministically rebuilds src/App.tsx from the manifest
// after parallel chunk generation has produced all page files.
//
// Why: Foundation emits App.tsx BEFORE the page files exist, so it has to
// guess at lazy-import names. A single typo (Home vs HomePage, default vs
// named) crashes the runtime with React error #306 on navigation. Rebuilding
// from manifest.Routes + real page exports as ground truth eliminates the
// whole class of bugs.
//
// Gated on manifest.ExportStyle == "named-lazy". Legacy manifests fall through.
func mergeAppRoutes(files []models.ProjectFile, manifest *models.ProjectManifest) []models.ProjectFile {
	if manifest == nil || manifest.ExportStyle != "named-lazy" {
		return files
	}
	if len(manifest.Routes) == 0 {
		return files
	}

	// Strip any stray src/Page.tsx the model invented as an alternate microfrontend entry.
	// Vite federation exposes './App' → './src/App.tsx'; Page.tsx is dead code and
	// occasionally absorbs `import './index.css'`, which then never reaches the host bundle.
	files = stripPageTsx(files)

	appIdx := -1
	for i, f := range files {
		if f.Path == "src/App.tsx" {
			appIdx = i
			break
		}
	}
	if appIdx == -1 {
		log.Printf("[merge-routes] src/App.tsx not found, skipping")
		return files
	}

	registry := buildExportRegistry(files)

	var resolved []resolvedRoute
	var skipped []string

	for _, route := range manifest.Routes {
		filePath := route.FilePath
		if filePath == "" {
			skipped = append(skipped, fmt.Sprintf("%s (no file_path)", route.Path))
			continue
		}
		exports, ok := registry[filePath]
		if !ok {
			for _, alt := range resolveAlternatives(filePath) {
				if found, exists := registry[alt]; exists {
					exports = found
					filePath = alt
					ok = true
					break
				}
			}
			if !ok {
				skipped = append(skipped, fmt.Sprintf("%s (file %s missing)", route.Path, route.FilePath))
				continue
			}
		}
		if !exports[route.PageName] {
			skipped = append(skipped, fmt.Sprintf("%s (file exists but missing named export %s)", route.Path, route.PageName))
			continue
		}

		importFrom := strings.TrimSuffix(filePath, ".tsx")
		importFrom = strings.TrimSuffix(importFrom, ".ts")
		importFrom = "@/" + strings.TrimPrefix(importFrom, "src/")

		resolved = append(resolved, resolvedRoute{
			Path:       route.Path,
			PageName:   route.PageName,
			FilePath:   filePath,
			ImportFrom: importFrom,
		})
	}

	if len(resolved) == 0 {
		log.Printf("[merge-routes] no routes resolved against generated files, leaving App.tsx as-is (skipped: %v)", skipped)
		return files
	}

	var sb strings.Builder
	// index.css must be imported here — App.tsx is the federation entry (vite.config.ts
	// exposes './App'). Without this the host loads the remote without Tailwind variables.
	sb.WriteString("import './index.css';\n")
	sb.WriteString("import { lazy, Suspense } from 'react';\n")
	sb.WriteString("import { BrowserRouter, Routes, Route } from 'react-router-dom';\n")
	sb.WriteString("import { AppProviders } from '@/components/shared/AppProviders';\n")
	sb.WriteString("import { PageLoader } from '@/components/shared/PageLoader';\n\n")

	for _, route := range resolved {
		fmt.Fprintf(&sb, "const %s = lazy(() => import('%s').then((m) => ({ default: m.%s })));\n",
			route.PageName, route.ImportFrom, route.PageName)
	}

	sb.WriteString("\nexport default function App() {\n")
	sb.WriteString("  return (\n")
	sb.WriteString("    <BrowserRouter>\n")
	sb.WriteString("      <AppProviders>\n")
	sb.WriteString("        <Suspense fallback={<PageLoader />}>\n")
	sb.WriteString("          <Routes>\n")
	for _, route := range resolved {
		fmt.Fprintf(&sb, "            <Route path=%q element={<%s />} />\n", route.Path, route.PageName)
	}
	sb.WriteString("          </Routes>\n")
	sb.WriteString("        </Suspense>\n")
	sb.WriteString("      </AppProviders>\n")
	sb.WriteString("    </BrowserRouter>\n")
	sb.WriteString("  );\n")
	sb.WriteString("}\n")

	files[appIdx].Content = sb.String()

	// Foundation occasionally drops PageLoader; the rebuilt App.tsx imports it, so re-inject if missing.
	if _, ok := registry["src/components/shared/PageLoader.tsx"]; !ok {
		files = append(files, models.ProjectFile{
			Path:    "src/components/shared/PageLoader.tsx",
			Content: pageLoaderSource,
		})
	}

	log.Printf("[merge-routes] rebuilt App.tsx with %d routes (skipped %d: %v)", len(resolved), len(skipped), skipped)
	return files
}
