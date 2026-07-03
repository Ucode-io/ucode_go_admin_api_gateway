package chat_prompts

import (
	"fmt"
	"strings"

	"ucode/ucode_go_api_gateway/api/models"
)

// chat widget for the fixed u-code builder assistant. The networking layer is
// injected as template files; unlike PromptAgentIntegrator this assistant has no
// agent_id, no runAgent call and no downloadable files — it streams build progress.
var PromptBuilderAgentIntegrator = `You are a senior React/TypeScript engineer. This application ships with a built-in AI builder assistant for its end-users: they chat with it in plain language and it builds out their app's data model for them — creating tables, fields, relations and menu sections, seeding records, and answering questions about their data. Your job is to wire this assistant into the already-built admin panel so end-users can reach it, exactly matching the app's design.

You MUST respond by calling the integrate_agent tool. Never reply with plain text.

====================================
WHAT ALREADY EXISTS (DO NOT RECREATE)
====================================
The networking layer is already provided as project files. They are CORRECT and AUTHORITATIVE — never re-emit, modify, or duplicate them:

1. src/lib/builderAgentClient.ts
     export interface BuilderEvent { type: string; message?: string; value?: string; icon?: string; percent?: number; data?: unknown }
     export interface BuilderSummary { tables: number; login_tables: number; client_types: number; roles: number; fields: number; relations: number; menus: number; items: number }
     export interface BuilderChatMessage { role: 'user' | 'assistant'; content: string }
     export async function sendBuilderMessage(
       content: string,
       history: BuilderChatMessage[],
       handlers?: { onEvent?: (e: BuilderEvent) => void; signal?: AbortSignal },
     ): Promise<{ reply: string; summary?: BuilderSummary }>
   sendBuilderMessage streams the assistant's work: every build step (creating a table, adding a field, …) arrives as a BuilderEvent through onEvent while the request is in flight, and the promise resolves with the final natural-language reply plus a summary of what was built. It authenticates through the app's shared axios instance automatically. Throws on failure.

2. src/hooks/useBuilderAgent.ts
     export interface BuilderMessage { id: string; role: 'user' | 'assistant'; content: string; summary?: BuilderSummary }
     export function useBuilderAgent(): {
       messages: BuilderMessage[];
       steps: BuilderEvent[];   // live build steps for the in-flight turn (cleared when it completes)
       isLoading: boolean;
       error: string | null;
       send: (text: string) => Promise<void>;
       reset: () => void;
     }
   A ready-made chat session hook. It keeps the transcript, exposes send(), and surfaces the live 'steps' of the current turn plus loading/error state. Prefer this for the chat UI — you never manage history or the wire protocol yourself.

Import these with the project's '@/' alias. Do NOT call axios or fetch directly — everything goes through these two files. There is NO agent id to pass; the endpoint is fixed inside the client.

====================================
YOUR JOB
====================================
1. Build a polished, on-brand chat widget on top of useBuilderAgent(): a floating launcher button that opens a chat panel. Render the transcript, a text input, a send button, loading state and error state, and auto-scroll to the latest message.
2. SHOW THE LIVE BUILD STEPS. While a turn is in flight, the hook's 'steps' array fills with BuilderEvent items describing what the assistant is doing right now (e.g. message "Создаю таблицу", value "Клиенты"; message "Добавлено поле", value "phone"). Render them as a lightweight live activity list under the in-progress message — each event's 'icon' is a Lucide icon name and 'message'/'value' are display text. This "watch it build" feedback is the point of the widget; do not hide it. When the turn finishes the steps clear and the final assistant reply stays in the transcript.
3. MOUNT it in the app shell — the layout/root that wraps every page — so it appears on every screen of the admin panel. Find that file in the provided file graph and render the widget there.

Word the widget's empty state and input placeholder around what the assistant does: building tables, fields, relations and menus, adding records, and answering questions about the data. Keep it in the app's language (Russian if the app is Russian).

====================================
DESIGN & QUALITY RULES
====================================
- Match the existing design system: reuse the project's UI-kit components, Tailwind tokens, colors, spacing, radius, shadows and typography. The widget must look native to the app, not bolted on.
- Production-grade and accessible: keyboard submit (Enter), disabled state while loading, visible focus, sensible empty state, graceful error message. Mobile-friendly. The panel must not overlap critical app controls.
- SURGICAL: change only what is needed to add and mount the widget. Do NOT refactor, rename, reformat, or touch unrelated code, styles, or data-fetching. Preserve imports, structure, and comments in files you edit.
- TypeScript must be correct and self-consistent with the existing code (imports resolve, types line up).

====================================
OUTPUT RULES
====================================
- Return the COMPLETE content of every file you create or change — never a diff or snippet.
- Include ONLY files you actually create or modify (e.g. the new widget component plus the one shell file you mount it in). Do NOT re-emit builderAgentClient.ts or useBuilderAgent.ts.
- change_summary: one sentence describing what you added and where you mounted it.`

// BuildBuilderAgentIntegrationMessage renders the user-side message for the
// integrate_agent call for the fixed builder assistant: the pre-injected template
// files the model must reuse, and enough of the project (file graph + selected file
// contents) to place and style the widget correctly.
func BuildBuilderAgentIntegrationMessage(view models.BuilderAgentIntegrationView) string {
	var sb strings.Builder

	sb.WriteString("====================================\n")
	sb.WriteString("THE ASSISTANT TO INTEGRATE\n")
	sb.WriteString("====================================\n")
	sb.WriteString("A built-in AI builder assistant for THIS admin panel's end-users. In plain language it builds the app's data model (tables, fields, relations, menu sections), seeds records, and answers questions about the data. It streams its work step by step. It has no configuration and no id — it always talks to the fixed backend endpoint baked into the provided client.\n")

	if len(view.TemplateFiles) > 0 {
		sb.WriteString("\n====================================\n")
		sb.WriteString("ALREADY-PROVIDED FILES (import & reuse — never re-emit)\n")
		sb.WriteString("====================================\n")
		for _, path := range view.TemplateFiles {
			fmt.Fprintf(&sb, "- %s\n", path)
		}
	}

	sb.WriteString("\n====================================\n")
	sb.WriteString("PROJECT FILE GRAPH (find the app shell / layout to mount the widget)\n")
	sb.WriteString("====================================\n")
	sb.WriteString(view.FileGraphJSON)
	sb.WriteString("\n")

	sb.WriteString("\n====================================\n")
	sb.WriteString("RELEVANT CURRENT FILES (full content — the app shell/layout where a global chat widget must be mounted, so you match its structure and styling)\n")
	sb.WriteString("====================================\n")
	sb.WriteString(view.FilesContext)
	sb.WriteString("\n")

	sb.WriteString("\nIntegrate the builder assistant now by calling the integrate_agent tool. Return ONLY the files you create or change, each with complete content.")
	return sb.String()
}
