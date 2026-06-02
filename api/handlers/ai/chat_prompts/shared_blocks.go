package chat_prompts

// ExportConventionBlock is the single source of truth for page exports and
// App.tsx lazy-loading. Injected into Foundation/UI Kit/Feature/Repair prompts.
//
// Named + lazy (not default + lazy) chosen because:
//   - every lazy block has the same shape — deterministic for codegen and repair
//   - `m.PageName === undefined` becomes an unambiguous signal of a missing named export
//   - `export default function X` silently loses X at the module boundary
func ExportConventionBlock() string {
	return `====================================
EXPORT CONVENTION FOR PAGES (CRITICAL — same everywhere)
====================================

EVERY page file MUST use a NAMED function export — never default export:
  ✅ export function HomePage() { ... }
  ❌ export default function HomePage() { ... }   // breaks lazy resolver below

App.tsx MUST lazy-load every page via this exact shape:
  ✅ const HomePage = lazy(() =>
       import('@/pages/HomePage').then(m => ({ default: m.HomePage }))
     );
  ❌ const HomePage = lazy(() => import('@/pages/HomePage'));        // pages have no default
  ❌ import HomePage from '@/pages/HomePage';                         // no code-splitting, breaks routing
  ❌ const HomePage = lazy(() => import('@/pages/Home'));             // file/name mismatch

App.tsx imports lazy+Suspense from 'react':
  import { lazy, Suspense } from 'react';

App.tsx wraps <Routes> in <Suspense fallback={<PageLoader />}>.

The named export's identifier MUST be identical to the filename without .tsx:
  src/pages/HomePage.tsx       → export function HomePage
  src/pages/ProductDetail.tsx  → export function ProductDetail
  src/pages/AIProposal.tsx     → export function AIProposal

If you violate this contract, the preview crashes with React error #306 the moment
the user navigates to that route (because m.<Name> resolves to undefined).
====================================
`
}

// FeaturePageExemplar gives feature-chunk prompts a copy-pasteable skeleton
// that wires useApiQuery + extractList + useApiMutation + useAppForm correctly.
// Names are placeholders — models replicate the shape with project-specific names.
func FeaturePageExemplar() string {
	return `====================================
FEATURE PAGE EXEMPLAR (copy the SHAPE, replace names from your manifest)
====================================

import { useState } from 'react';
import { z } from 'zod';
import { useApiQuery, useApiMutation } from '@/hooks/useApi';
import { useAppForm } from '@/hooks/useAppForm';
import { extractList } from '@/lib/apiUtils';
import { Button } from '@/components/ui/button';
import { DataTable } from '@/components/shared/DataTable';
import { FormModal } from '@/components/shared/FormModal';
import { PageHeader } from '@/components/shared/PageHeader';
import type { Entity } from '@/types';

const entitySchema = z.object({
  name: z.string().min(1, 'Required'),
});
type EntityInput = z.infer<typeof entitySchema>;

export function EntityPage() {
  const [isOpen, setOpen] = useState(false);

  const { data, isLoading, error } = useApiQuery<unknown>(['entities'], '/v2/items/entities');
  const entities = extractList<Entity>(data);

  const createMutation = useApiMutation<Entity, EntityInput>({
    url: '/v2/items/entities',
    method: 'POST',
    successMessage: 'Created',
    invalidateKeys: [['entities']],
    options: { onSuccess: () => setOpen(false) },
  });

  const form = useAppForm(entitySchema, { name: '' });

  if (isLoading) return <DataTableSkeleton />;     // use the UI Kit skeleton
  if (error) return <ErrorState error={error} />;  // use the UI Kit error state

  return (
    <>
      <PageHeader title="Entities" action={<Button onClick={() => setOpen(true)}>New</Button>} />
      <DataTable columns={columns} data={entities} />
      <FormModal open={isOpen} onClose={() => setOpen(false)} onSubmit={form.handleSubmit((v) => createMutation.mutate(v))}>
        {/* form fields */}
      </FormModal>
    </>
  );
}

KEY POINTS (CRITICAL):
  - Component uses NAMED export 'export function EntityPage' (App.tsx lazy resolver depends on it).
  - useApiQuery<unknown> + extractList<Entity>(data) — never index data manually.
  - useApiMutation takes one config object (not positional args).
  - useAppForm wraps react-hook-form with Zod resolver — never useForm directly.
  - Loading + error states ALWAYS appear before reading list.
====================================
`
}

// TemplateAPIDigest is a compact signature reference for template helpers,
// used in feature-chunk prompts INSTEAD of the full source. Foundation prompts
// still ship the full source because they author the initial wiring.
func TemplateAPIDigest() string {
	return `====================================
PRE-BUILT TEMPLATE API (already in the project — import, never re-implement)
====================================

src/hooks/useApi.ts
  useApiQuery<T>(queryKey: unknown[], url: string, axiosConfig?, queryOptions?) — wraps useQuery
  useApiMutation<TData, TVars>({ url, method?, successMessage?, invalidateKeys?, options? }) — single config-object
  useApiInfiniteQuery<T>(queryKey, getUrl: (page) => string, options?) — paginated infinite
  interface PaginatedResponse<T> { data: T[]; total: number; page: number; limit: number; totalPages: number }

src/hooks/useAppForm.ts
  useAppForm<TSchema>(schema: ZodSchema, defaultValues?, options?) — react-hook-form + zodResolver + mode:'onBlur'

src/lib/apiUtils.ts
  extractList<T>(data): T[]      — safely reads data.data.response as array; returns [] on any nullish chain
  extractSingle<T>(data): T|null — safely reads single entity
  extractCount(data): number     — safely reads data.data.count

src/lib/utils.ts
  cn(...inputs): string                            — clsx + twMerge
  formatDate(date: Date|string, fmt?): string
  formatCurrency(amount: number, currency?): string
  getInitials(name: string): string
  truncate(text: string, max: number): string

src/components/shared/AppProviders.tsx
  AppProviders({ children }) — wraps QueryClientProvider + Toaster (sonner)

src/components/shared/PageLoader.tsx
  PageLoader() — full-screen spinner (used by App.tsx Suspense fallback)

USAGE RULES:
  - NEVER create src/lib/api.ts or a duplicate axios instance — use @/config/axios.
  - NEVER re-implement extractList/extractSingle/extractCount — those are the ONLY safe accessors for our API shape.
  - useApiMutation takes ONE config object — not positional args.
====================================
`
}

// LazyAppTsxExemplar is a copy-pasteable App.tsx skeleton demonstrating the
// named-lazy pattern. Foundation prompts inject it as the structure to copy.
func LazyAppTsxExemplar() string {
	return `EXACT SHAPE for src/App.tsx (copy structure, replace pages with manifest.Routes):

import { lazy, Suspense } from 'react';
import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { AppProviders } from '@/components/shared/AppProviders';
import { PageLoader } from '@/components/shared/PageLoader';

const HomePage = lazy(() => import('@/pages/HomePage').then(m => ({ default: m.HomePage })));
const AboutPage = lazy(() => import('@/pages/AboutPage').then(m => ({ default: m.AboutPage })));
// ...one lazy const per page from manifest.Routes

export default function App() {
  return (
    <BrowserRouter>
      <AppProviders>
        <Suspense fallback={<PageLoader />}>
          <Routes>
            <Route path="/" element={<HomePage />} />
            <Route path="/about" element={<AboutPage />} />
            {/* one <Route> per entry in manifest.Routes */}
          </Routes>
        </Suspense>
      </AppProviders>
    </BrowserRouter>
  );
}
`
}
