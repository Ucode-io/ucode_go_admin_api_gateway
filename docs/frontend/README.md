# Frontend Integration Docs

Cross-team specs for the frontend that consumes the AI chat / project
generation API. Read these in order when you join the team or are picking up
chat-related work.

| Doc                         | Read when                                                                                              |
| --------------------------- | ------------------------------------------------------------------------------------------------------ |
| `CHAT_MESSAGE_MARKERS.md`   | Working with chat messages. Explains how `EnrichedMessage.plan/questions/error` are populated.          |
| `CHAT_ERROR_MESSAGES.md`    | Rendering or handling errors anywhere in the chat flow — SSE, HTTP, persisted history.                  |
| `PREVIEW_BUILD_ERRORS.md`   | Working on the in-browser preview runtime. Covers the silent-blank-screen class of failures.            |

## Source of truth

- Marker constants and parsing: `api/handlers/ai/jsonutil.go`,
  `api/handlers/v1/ai_chat.go::enrichMessages`.
- Error codes: `api/handlers/v1/ai_chat_error.go::ErrCode*`.
- SSE event types: `api/handlers/v1/generation_stream.go`.
- Models exposed over HTTP: `api/models/ai_chat_models.go`.

If you need a new error code, a new marker, or a new SSE event type,
raise it on the backend first — these docs follow the backend contract.
