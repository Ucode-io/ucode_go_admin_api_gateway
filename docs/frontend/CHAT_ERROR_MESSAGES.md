# Chat Error Messages — Frontend Integration Spec

**Audience:** Frontend engineers integrating the AI chat UI.
**Backend owner:** `ucode_go_admin_api_gateway`
**Scope:** How pipeline failures are surfaced through SSE and chat history so
the user always sees a clear, actionable error — even after a page reload or
client disconnect.

---

## 1. Why this exists

Previously, when AI project generation failed mid-pipeline:

- The error was emitted **only** as a transient SSE `EvError` event.
- If the user disconnected (closed the tab, lost network, navigated away) the
  error was lost — nothing was persisted to chat history.
- On the next page load, the user saw only their last question with no clue
  what went wrong.
- The frontend had no structured payload to render specialized UI (retry
  buttons, top-up prompts, support links) — only a free-form `message` string.

This led to confusing UX where the chat looked "stuck": users would re-send
the same message multiple times, hoping for a different result.

The backend now **always** persists a structured `[ERROR]` message into chat
history when the pipeline fails, and the SSE `EvError` event carries the same
structured payload.

---

## 2. The contract

Every chat message returned by the backend (`GET /v2/ai-chats/{id}/messages`
or via the `done`/`error` SSE event) is shaped as `EnrichedMessage`:

```ts
interface EnrichedMessage {
  id: string;
  chat_id: string;
  role: "user" | "assistant";
  content: string;          // Human-readable summary, marker stripped.
  images: string[];
  has_files: boolean;
  tokens_used: number;
  created_at: string;
  like_count: number;
  dislike_count: number;
  current_user_reaction: string;

  // Optional payloads — present depending on the message kind:
  plan?: HaikuPlan;          // From [DIAGRAMS_GENERATED] messages.
  questions?: AiQuestion[];  // From [QUESTIONS_ASKED] messages.
  error?: AiChatError;       // From [ERROR] messages. ← NEW
}
```

When `error` is non-null, the message is a **failure record** and should be
rendered as an error card — not as a normal assistant reply.

### 2.1 `AiChatError` shape

```ts
interface AiChatError {
  code: AiChatErrorCode;     // Machine-readable identity. Switch on this.
  phase: AiChatErrorPhase;   // WHERE the failure happened in the pipeline.
  message: string;           // User-facing one-liner, already localized.
  details?: string;          // Internal Go error string. NOT for main UI —
                             // surface behind a "Show details" toggle.
  retryable: boolean;        // If true, render a "Try again" button.
  user_action?: string;      // Optional suggested next step, localized.
}
```

### 2.2 Error code catalog

| `code`                 | Meaning                                              | `retryable` | Suggested UI                                       |
| ---------------------- | ---------------------------------------------------- | :---------: | -------------------------------------------------- |
| `TOKEN_LIMIT_EXCEEDED` | Project hit its daily/monthly AI token budget.       | `false`     | Top-up / upgrade-plan CTA. Hide Retry.             |
| `AI_MAX_TOKENS`        | Model produced more tokens than its hard cap allows. | `true`      | Retry. Suggest splitting the request.              |
| `TIMEOUT`              | Inner HTTP call exceeded its deadline.               | `true`      | Retry button. Optional "Simplify request" hint.    |
| `ROUTER_FAILED`        | AI router (intent classifier) errored or timed out.  | `true`      | Retry. Suggest rephrasing.                         |
| `ARCHITECT_FAILED`     | Architect agent failed to produce the plan.          | `true`      | Retry. Suggest providing more detail.              |
| `PROVISIONING_FAILED`  | Backend project/env/MCP creation failed.             | `true`      | Retry. Surface "Contact support" if persistent.    |
| `MANIFEST_FAILED`      | File-plan generation failed.                         | `true`      | Retry — usually transient.                         |
| `CODEGEN_FAILED`       | Code generation chunks all failed.                   | `true`      | Retry. Suggest simplifying.                        |
| `PUBLISH_FAILED`       | Microfrontend publish step failed.                   | `true`      | Retry — files were generated, can re-publish.      |
| `VALIDATION_FAILED`    | Generated project failed the quality gate.           | `true`      | Retry.                                             |
| `INTERNAL_ERROR`       | Catch-all for anything else.                         | `true`      | Retry. Surface "Contact support" if persistent.    |

`code` is **stable across releases** — safe to switch on. Always include a
default branch that renders a generic error card for codes you don't recognize
(forward compatibility).

### 2.3 Phase catalog

`phase` tells you which step of the pipeline failed. Useful for diagnostics
and progressive disclosure ("Your project was designed but couldn't be
published — try again to deploy it").

| `phase`         | Description                                              |
| --------------- | -------------------------------------------------------- |
| `routing`       | Intent classification (router agent).                    |
| `architect`     | Plan generation (Architect agent).                       |
| `provisioning` | Backend project / environment / API key creation.        |
| `manifest`      | File-plan generation.                                    |
| `codegen`       | Code generation (Foundation / UIKit / feature chunks).   |
| `publish`       | Microfrontend publish to GitLab + function service.      |
| `validation`    | Quality gate.                                            |
| `unknown`       | Couldn't be classified — treat as generic error.         |

---

## 3. How errors arrive at the frontend

There are **three** entry points for the same `AiChatError` payload. Implement
the same rendering logic for all three so the user sees a consistent UI
regardless of how the error was delivered.

### 3.1 SSE `error` event (live failure)

When the user is connected via `?stream=true` and the pipeline fails:

```json
event: data
data: {
  "type": "error",
  "icon": "alert-circle",         // or "ban" for TOKEN_LIMIT_EXCEEDED
  "message": "Не удалось сгенерировать код проекта.",
  "data": {
    "error": { /* full AiChatError */ },
    "token_limit": { ... }         // Present only for TOKEN_LIMIT_EXCEEDED.
  }
}
```

`event.message` is a convenience copy of `event.data.error.message` for
renderers that only need the headline. Everything else lives under
`event.data.error` — switch on `event.data.error.code`.

After receiving an `error` event, the stream closes. The same error is now
**also** persisted to chat history, so any subsequent `GET /messages` call
will see it as an assistant message with `error` populated.

### 3.2 HTTP error response (non-streaming)

When the chat handler is called without `?stream=true` and the pipeline fails:

```http
HTTP/1.1 500
Content-Type: application/json

{
  "code": 12,                     // gRPC error wrapper
  "msg": "...",
  "data": {
    "error": { /* full AiChatError */ },
    "message": "Не удалось сгенерировать код проекта."
  }
}
```

For `TOKEN_LIMIT_EXCEEDED` only, the response keeps the existing
`status 402 Payment Required` path with the `TokenLimitData` payload —
this preserves billing-UI compatibility. The same `[ERROR]` message is still
persisted to chat history, so refreshing the chat shows the failure card.

### 3.3 Persisted chat message (after reload)

When the frontend later fetches `GET /v2/ai-chats/{id}/messages`, failed
turns appear as assistant `EnrichedMessage`s with `error` set:

```json
{
  "id": "abc-...",
  "chat_id": "...",
  "role": "assistant",
  "content": "Не удалось сгенерировать код проекта.",
  "error": {
    "code": "CODEGEN_FAILED",
    "phase": "codegen",
    "message": "Не удалось сгенерировать код проекта.",
    "details": "chunked: all 7 feature chunks failed",
    "retryable": true,
    "user_action": "Попробуйте ещё раз..."
  },
  ...
}
```

**Always check `message.error` before rendering a normal assistant bubble.**

---

## 4. Rendering recipe

Recommended render logic, in priority order:

```tsx
function renderMessage(m: EnrichedMessage) {
  if (m.error) return <ErrorCard error={m.error} onRetry={handleRetry} />;
  if (m.questions?.length) return <QuestionsCard questions={m.questions} />;
  if (m.plan) return <DiagramCard plan={m.plan} />;
  return <MarkdownBubble role={m.role} content={m.content} />;
}
```

### 4.1 `<ErrorCard>` checklist

- Red/destructive surface, alert-circle icon.
- Heading: `error.message` (already localized).
- Body: `error.user_action` if present.
- Primary button: **Retry** (only if `error.retryable === true`).
- Secondary affordance: **Show technical details** → toggles `error.details`.
- For `TOKEN_LIMIT_EXCEEDED` specifically: replace Retry with a
  **Top-up / Upgrade plan** CTA.
- Reflect the failed turn in the chat timeline (do **not** delete the user
  message that triggered it).

### 4.2 Retry semantics

"Retry" = re-send the **same** original user message that caused the failure.

- Locate the most recent `role: "user"` message above the error card.
- POST it again to `/v2/ai-chats/{id}/messages` with the same payload.
- Visually pair the new exchange directly below the failed one — don't
  remove the failed card so the user retains context.

### 4.3 Streaming UI states

```
[idle] ─send─▶ [streaming] ─error─▶ [error]   ← render <ErrorCard inline>
                          ─done──▶  [ready]
```

When the SSE `error` event arrives:

1. Stop the heartbeat / progress indicator.
2. Render the error card in place of the in-flight assistant bubble.
3. Re-enable the input.

---

## 5. Backwards compatibility

- Existing messages without `error` are unaffected — frontend logic
  branching on `error == null` keeps working.
- Token-limit billing UI keeps working: the `402` response shape is
  unchanged for that specific case.
- The `EvError` event still carries `message` and `icon` at the top level,
  so any legacy renderer treating SSE errors as plain strings keeps showing
  a meaningful message.

---

## 6. Related markers

The chat protocol uses three other marker prefixes on the `content` string,
all handled by the same parsing logic:

| Marker                  | Field exposed             | Spec                                  |
| ----------------------- | ------------------------- | ------------------------------------- |
| `[QUESTIONS_ASKED] ...` | `EnrichedMessage.questions` | See `CHAT_MESSAGE_MARKERS.md`        |
| `[DIAGRAMS_GENERATED] ...` | `EnrichedMessage.plan` | See `CHAT_MESSAGE_MARKERS.md`        |
| `[ERROR] ...`           | `EnrichedMessage.error`   | This document                         |

All three follow the format `"[MARKER] <text summary>\n<json payload>"` and
the backend strips the marker from `content` before returning to the client.

---

## 7. Questions / changes

Backend file: `api/handlers/v1/ai_chat_error.go`.
Model: `api/models/ai_chat_models.go::AiChatError`.
Persistence: `ChatProcessor.persistPipelineError()` is called from both
streaming and non-streaming error paths in `ai_messaging_handler.go`.

If you need a new error code, raise it on the backend first — the catalog in
§2.2 is the source of truth.
