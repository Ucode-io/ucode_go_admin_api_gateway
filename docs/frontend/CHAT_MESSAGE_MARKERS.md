# Chat Message Markers — Protocol Reference

**Audience:** Frontend engineers working with `/v2/ai-chats/{id}/messages`.
**Backend owner:** `ucode_go_admin_api_gateway`

The AI chat persists assistant messages whose `content` is prefixed with one
of a small set of **markers**. Markers carry conversation-state information
that the AI router uses across turns, plus structured payloads that the
frontend renders as specialized UI (question forms, diagrams, error cards).

This document is the source of truth for what each marker means, what the
backend strips for the client, and how the frontend should branch.

---

## 1. The three markers

| Marker prefix              | Persisted on       | Frontend field             | Render as                |
| -------------------------- | ------------------ | -------------------------- | ------------------------ |
| `[QUESTIONS_ASKED] `       | Assistant message  | `EnrichedMessage.questions` | Question form / chips    |
| `[DIAGRAMS_GENERATED] `    | Assistant message  | `EnrichedMessage.plan`     | Diagram cards            |
| `[ERROR] `                 | Assistant message  | `EnrichedMessage.error`    | Error card. See `CHAT_ERROR_MESSAGES.md` |

All markers share the same wire format:

```
[MARKER] <one-line human-readable summary>\n<JSON payload>
```

The backend handler (`enrichMessages` in `api/handlers/v1/ai_chat.go`)
**strips** the marker and the JSON line before returning to the client, so
`EnrichedMessage.content` always holds only the first-line summary. The
parsed JSON payload is exposed via the appropriate typed field.

This means the frontend **never needs to parse marker strings itself** — just
branch on which optional field is non-null:

```ts
if (m.error)     return <ErrorCard ...>;
if (m.questions) return <QuestionsForm ...>;
if (m.plan)      return <DiagramCard ...>;
return <Bubble content={m.content} />;
```

---

## 2. `[QUESTIONS_ASKED]`

### Purpose

The router agent asks the user a structured questionnaire when it needs more
detail to plan a project (which user types, which modules, which integrations,
etc.). The questions are persisted on the assistant turn so that:

- The conversation can be resumed after a reload — the form is reconstructed.
- The router on the next turn can detect that the user is **answering**
  these questions and route directly to `plan_request` (code generation).

### Wire shape

```
[QUESTIONS_ASKED] Please answer a few questions to get started.
[{"id":"module-types","title":"...","type":"multi","options":[...]}, ...]
```

After enrichment:

```json
{
  "role": "assistant",
  "content": "Please answer a few questions to get started.",
  "questions": [
    {
      "id": "module-types",
      "title": "What module types (menus) will be in the platform?",
      "type": "multi",
      "options": ["Dashboard", "Orders", "Reports", ...]
    },
    ...
  ]
}
```

### Frontend responsibilities

1. Render the questions inline in the chat as a form (chips for `single`,
   checkboxes for `multi`).
2. When the user submits, send a new POST with `content` built from the
   selected answers. The recommended format is what the backend recognizes:

   ```
   Question: <title 1>
   User answer: <comma-joined selections>

   Question: <title 2>
   User answer: <comma-joined selections>
   ```

3. The backend router will see the previous `[QUESTIONS_ASKED]` state marker
   and route directly to project generation — no further confirmation needed.

---

## 3. `[DIAGRAMS_GENERATED]`

### Purpose

For some flows the AI returns diagrams (ER/flow) for the user to review
before code generation. Persisted so the diagram view can be re-rendered on
reload.

### Wire shape

```
[DIAGRAMS_GENERATED] Here are the diagrams for your project. Review them and let me know when you're ready to build.
{"er_diagram":"...","flow_diagram":"...", ...}
```

After enrichment:

```json
{
  "role": "assistant",
  "content": "Here are the diagrams for your project. ...",
  "plan": { "er_diagram": "...", "flow_diagram": "..." }
}
```

### Frontend responsibilities

Render the diagrams (Mermaid, react-flow, whatever is in use) when
`message.plan` is non-null. The `content` field still holds the one-line
introduction so a simple text fallback also works.

---

## 4. `[ERROR]`

See `CHAT_ERROR_MESSAGES.md` for the full spec. In short:

- Carries a structured `AiChatError` payload (`code`, `phase`, `message`,
  `details`, `retryable`, `user_action`).
- Persisted whenever the generation pipeline fails, in **both** streaming
  and non-streaming flows.
- Survives client disconnect — visible on reload via `GET /messages`.
- Same payload is delivered live via the SSE `error` event's `data.error`.

---

## 5. State markers and the router (Bug B fix)

Markers aren't just for rendering — they also drive the **router's
conversation state detection**:

- If the most recent assistant message starts with `[QUESTIONS_ASKED]` →
  the user is answering questions → route to `plan_request`.
- If the most recent assistant message starts with `[DIAGRAMS_GENERATED]` →
  the user is reviewing a plan → route accordingly.
- If the most recent assistant message starts with `[ERROR]` → the user is
  retrying or asking what went wrong.

### Backend safeguards

Previous bug: for users who pasted long initial requests (multi-page TZ
documents), the `[QUESTIONS_ASKED]` marker fell out of the router's history
window (only 6 messages), causing the router to re-ask the questions in a
loop.

The backend now:

1. Loads up to **20 recent messages** from chat for context.
2. Sends up to **12 messages** to the router (was 6).
3. **Always preserves** the most recent state-marker assistant message even
   when truncating — it's added at the top of the router transcript as
   `Assistant (carried over from earlier in the conversation, ...)`.
4. **Detects state deterministically** on the full history and injects an
   explicit `CONVERSATION STATE (authoritative)` block into the router
   prompt — the router no longer has to find the marker in free text.

### Frontend implications

No changes required for normal chat rendering — but be aware:

- **Don't filter out** marker messages from the history you upload back to
  the backend. The backend strips JSON for the AI; it expects the markers
  to be present in chat history.
- **Do** render the **first-line summary** (`content`) for older marker
  messages so the conversation reads naturally — questions/diagrams cards
  can be collapsed to their summary line after a few turns.

---

## 6. Why markers (not separate tables)?

Markers are stored as plain text on the existing `Message.content` column.
This keeps the chat-message schema flat — one query, one type, no joins —
and lets the AI router process state the same way it processes any other
assistant message: by reading text. The trade-off is that the backend has
to parse markers on read; the frontend gets the typed payload pre-extracted
in `EnrichedMessage`.

---

## 7. Related files

| Concern              | File                                                            |
| -------------------- | --------------------------------------------------------------- |
| Marker constants     | `api/handlers/ai/jsonutil.go::Marker*`                          |
| Enrich on HTTP read  | `api/handlers/v1/ai_chat.go::enrichMessages`                    |
| Strip on AI context  | `api/handlers/v1/ai_messaging.go::getChatHistory`               |
| Router state detect  | `api/handlers/ai/jsonutil.go::DetectConversationState`          |
| Router prompt build  | `api/handlers/ai/chat_prompts/ai_chat_prompts.go::BuildRouterMessage` |
| Error code catalog   | `api/handlers/v1/ai_chat_error.go::ErrCode*`                    |
