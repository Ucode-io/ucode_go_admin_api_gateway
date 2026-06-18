# Preview Build Errors — Frontend Surfacing

**Audience:** Frontend engineers who own the in-browser preview runtime.
**Status:** Open frontend work. Backend is not responsible for the fix.
**Why this exists:** We have a class of failures where backend generation
**succeeds** (files are produced and published) but the user sees a **blank
preview screen** with no error message. ~42% of generation-failure tickets
fall into this bucket.

---

## 1. The problem

Backend flow (success path):

```
Architect → Provisioning → Manifest → Codegen → Validation/Repair → Publish ✅
                                                                       │
                                                                       ▼
                                                       Files uploaded to GitLab + function svc
                                                                       │
                                                                       ▼
                                                  Frontend pulls file tree into virtual FS
                                                                       │
                                                                       ▼
                                                  Browser compiles & runs the generated app
```

The browser-side compilation (virtual FS) can fail for reasons the backend
**cannot** detect — the most common ones we've seen:

| Error pattern                                                                   | Real cause                                                                              |
| ------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------- |
| `File not found in virtual FS: /src/App`                                        | AI omitted `src/App.tsx` or named it incorrectly.                                       |
| `No matching export in "virtual:/src/App" for import "default"`                 | `src/App.tsx` exists but has no `export default`.                                       |
| `No matching export in "virtual:/src/hooks/use-mobile" for import "useMobile"`  | Export name mismatch (`useIsMobile` vs `useMobile`, etc.).                              |
| `File not found in virtual FS: /src/components/shared/AppProviders` (and 8+ more) | AI imported components it never generated.                                              |
| Build succeeds, screen is blank, no error in console                            | Runtime error swallowed by error boundary, or component returns `null`.                 |

The backend has multiple layers of repair for these (`ensureAppEntryDefaultExport`,
`injectMissingCriticalFiles`, `validateAndRepairGeneratedProject`), but the
recovery is not complete. Some broken projects still reach the user.

**The critical UX problem is not that builds fail — it's that the user can't
tell that they failed.** A blank screen looks identical to "still loading,"
which is why users hit "regenerate" many times instead of reporting an issue.

---

## 2. What the frontend should do

### 2.1 Surface compile errors in the chat as `[ERROR]` cards

When the browser-side compile fails, post it back to the chat as an error
the user can see and act on. Use the same `AiChatError` shape the backend
uses for pipeline errors (see `CHAT_ERROR_MESSAGES.md`) so the UI is
consistent. Suggested mapping:

```ts
{
  code: "PREVIEW_BUILD_FAILED",          // Frontend-only code, not in backend catalog
  phase: "preview",                       // Frontend-only phase
  message: "Не удалось собрать предпросмотр проекта.",
  details: <raw esbuild / vite error>,    // For "Show details" toggle
  retryable: true,
  user_action: "Попробуйте перегенерировать проект или сообщить нам об ошибке."
}
```

The chat does not need to send this back to the backend — it's a
client-side render. But please log it (with the error text) so we can
track frequency and improve backend repair coverage.

### 2.2 Add an inline "fix this" affordance

For the most common patterns — `No matching export`, `File not found in
virtual FS` — offer a one-click button that posts a recovery message back
to the chat. The backend will route this to `runMicrofrontendEdit()` which
already handles such repairs:

```ts
const repairPrompt = `Build failed with the following error. Please fix it:\n\n${error}`;
postChatMessage(repairPrompt);
```

We've seen users do this manually with success — automating it removes a
big UX papercut.

### 2.3 Distinguish "loading" from "broken"

After the file tree is pulled into the virtual FS, gate the preview with
an explicit ready state:

| State              | UI                                                       |
| ------------------ | -------------------------------------------------------- |
| `fetching_files`   | Spinner with "Загружаю файлы проекта"                    |
| `compiling`        | Spinner with "Собираю проект"                            |
| `ready`            | The iframe renders normally                              |
| `compile_error`    | Error card with details + Retry / "Fix with AI" buttons. |
| `runtime_error`    | Same card, but for thrown errors after compile.          |
| `blank_after_load` | After ready + 5s with no DOM rendered, show a soft warning. |

The `blank_after_load` state catches the "silent" failure mode where
compile succeeds but the app renders nothing (typical when the AI generates
a router with no matching route for `/`).

### 2.4 Stream the build log to "Show details"

Esbuild / Vite emit useful messages during the build. Capture them and
surface behind a collapsed "Show build log" toggle. Users who DM us a
screenshot of "blank screen" rarely include the console — making the log
one click away improves bug reports dramatically.

---

## 3. What the backend already does

So you know what's been ruled out:

- `ensureAppEntryDefaultExport` — guarantees `src/App.tsx` has
  `export default App`. (in `app_entry_contract.go`)
- `injectMissingCriticalFiles` — drops a stub `src/App.tsx` if one is
  missing. (in `ai_messaging_agents.go`)
- `dedupTsTsxPairs` — removes duplicate `.ts`/`.tsx` shadows.
- `validateAndRepairGeneratedProject` — up to 3 passes of validation +
  AI-driven repair.
- `mergeTemplateScaffold` — injects pre-built utility files
  (`apiUtils.ts`, `useApi.ts`, etc.) that AI should not generate from
  scratch.

What the backend **does not** do (and probably can't, reliably):

- Detect that the user's browser **environment** is missing a font, an
  external CDN script, or has a CSP that blocks something.
- Detect runtime errors thrown after mount.
- Detect that a page renders to an empty `<div />` because the route
  table doesn't match the URL.

These are inherently client-side.

---

## 4. Telemetry we'd love

If you instrument the preview runtime, please ship:

1. `preview_compile_failed` events with `{ chat_id, microfrontend_id, code, error_text }`.
2. `preview_blank_after_load` events (the 5s soft-warning class).
3. `preview_retry_clicked` and `preview_repair_clicked` events.

We can correlate these with backend `[ERROR]` messages to track the
end-to-end success rate of the generation pipeline.

---

## 5. Future work / coordination

If we see consistent patterns of preview failures **for projects where
backend validation passed**, that's a signal we should add a new
validator. Open a ticket against the backend with the failing
microfrontend ID — `validateAndRepairGeneratedProject` is the right place
to plug in a new check.

Backend-side files for that:

- `api/handlers/v1/ai_generation_validator.go` — validator implementations.
- `api/handlers/v1/ai_messaging_agents.go::validateAndRepairGeneratedProject` —
  orchestrator.
- `api/handlers/v1/app_entry_contract.go` — App.tsx export contract.
