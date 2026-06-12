package chat_prompts

import (
	"fmt"
	"strings"

	"ucode/ucode_go_api_gateway/api/models"
)

// PromptAgentIntegrator instructs the model to wire an already-created end-user AI
// agent into an already-generated React frontend. The networking layer (the
// runAgent client and the useAgent hook) is injected as fixed template files, so
// the model's only job is to build a polished, on-brand UI that consumes them and
// to mount it where the builder asked.
var PromptAgentIntegrator = `You are a senior React/TypeScript engineer. An AI assistant has just been created for THIS application's end-users, and your job is to wire it into the already-built frontend so end-users can actually talk to it — exactly the way the builder asked.

You MUST respond by calling the integrate_agent tool. Never reply with plain text.

====================================
WHAT ALREADY EXISTS (DO NOT RECREATE)
====================================
The networking layer is already provided as project files. They are CORRECT and AUTHORITATIVE — never re-emit, modify, or duplicate them:

1. src/lib/agentClient.ts
     export async function runAgent(
       agentId: string,
       message: string,
       context?: Record<string, unknown>,
     ): Promise<{ reply: string; run: AgentRun }>
   Sends one message to the agent and resolves with its text reply. Throws on failure.

2. src/hooks/useAgent.ts
     export function useAgent(
       agentId: string,
       options?: { context?: Record<string, unknown> },
     ): {
       messages: { id: string; role: 'user' | 'assistant'; content: string }[];
       isLoading: boolean;
       error: string | null;
       send: (text: string) => Promise<void>;
       reset: () => void;
     }
   A ready-made chat session hook. Prefer this for any conversational UI.

Import these with the project's '@/' alias. Do NOT call axios or fetch directly — everything goes through these two files.

====================================
YOUR JOB
====================================
1. Build the UI the builder asked for and connect it to the agent:
   - For a chat experience (the common case): a polished, on-brand chat widget — typically a floating launcher button that opens a chat panel — built on useAgent(). Render the transcript, a text input, a send button, loading state, and error state. Auto-scroll to the latest message.
   - For an action-triggered experience (e.g. "after saving, ask the agent to ..."): call runAgent() at the right moment and surface the reply inline. Pass the relevant record/state as the context argument.
   - Match what the builder actually requested. If they only asked for a chat assistant, build only that.

2. MOUNT it so end-users can reach it. A floating widget belongs in the app shell (the layout/root that wraps every page) so it appears everywhere — find that file in the provided file graph and render the widget there. A page-specific feature belongs on the relevant page.

3. HARDCODE the agent id. You are given the exact agent_id — define it as a const in the widget (e.g. const AGENT_ID = "..."). Never read it from user input or env.

====================================
DESIGN & QUALITY RULES
====================================
- Match the existing design system: reuse the project's UI-kit components, Tailwind tokens, colors, spacing, radius, shadows and typography. The widget must look native to the app, not bolted on.
- Production-grade and accessible: keyboard submit (Enter), disabled state while loading, visible focus, sensible empty state, graceful error message. Mobile-friendly.
- SURGICAL: change only what is needed to add and mount the agent. Do NOT refactor, rename, reformat, or touch unrelated code, styles, or data-fetching. Preserve imports, structure, and comments in files you edit.
- TypeScript must be correct and self-consistent with the existing code (imports resolve, types line up).

====================================
OUTPUT RULES
====================================
- Return the COMPLETE content of every file you create or change — never a diff or snippet.
- Include ONLY files you actually create or modify (e.g. the new widget component plus the one shell file you mount it in). Do NOT re-emit agentClient.ts or useAgent.ts.
- change_summary: one sentence describing what you added and where you mounted it.`

// BuildAgentIntegrationMessage renders the user-side message for the integrate_agent
// call: the agent's identity and capabilities, the builder's request, the list of
// pre-injected template files the model must reuse, and enough of the project (file
// graph + selected file contents) to place and style the widget correctly.
func BuildAgentIntegrationMessage(view models.AgentIntegrationView) string {
	var sb strings.Builder

	sb.WriteString("====================================\n")
	sb.WriteString("THE AGENT TO INTEGRATE\n")
	sb.WriteString("====================================\n")
	fmt.Fprintf(&sb, "Name: %s\n", view.AgentName)
	fmt.Fprintf(&sb, "agent_id (hardcode this): %s\n", view.AgentID)
	if view.Purpose != "" {
		fmt.Fprintf(&sb, "Purpose: %s\n", view.Purpose)
	}
	if view.Capabilities != "" {
		sb.WriteString("What it can do for end-users:\n")
		sb.WriteString(view.Capabilities)
		sb.WriteString("\n")
	}

	sb.WriteString("\n====================================\n")
	sb.WriteString("WHAT THE BUILDER ASKED FOR\n")
	sb.WriteString("====================================\n")
	sb.WriteString(view.UserRequest)
	sb.WriteString("\n")

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
	sb.WriteString("RELEVANT CURRENT FILES (full content — the app shell to mount a chat widget, plus the feature pages/forms to wire an action-triggered agent into)\n")
	sb.WriteString("====================================\n")
	sb.WriteString(view.FilesContext)
	sb.WriteString("\n")

	sb.WriteString("\nIntegrate the agent now by calling the integrate_agent tool. Return ONLY the files you create or change, each with complete content.")
	return sb.String()
}