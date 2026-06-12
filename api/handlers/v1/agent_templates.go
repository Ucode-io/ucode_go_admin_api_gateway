package v1

import "ucode/ucode_go_api_gateway/api/models"

// Agent integration template files. These are the correctness-critical networking
// layer for talking to a server-side agent from the generated frontend: a thin
// runAgent() client over the project's shared axios instance, and a useAgent()
// chat-session hook. We own and inject them verbatim so the model never has to
// (re)invent the wire protocol — it only builds the UI that consumes them.
const (
	agentClientFilePath = "src/lib/agentClient.ts"
	useAgentFilePath    = "src/hooks/useAgent.ts"
)

// agentTemplateFiles returns the template files to inject into the project. They are
// authoritative: when merged with the model's output, these win over any same-path
// file the model might have produced.
func agentTemplateFiles() []models.ProjectFile {
	return []models.ProjectFile{
		{Path: agentClientFilePath, Content: agentClientTemplate},
		{Path: useAgentFilePath, Content: useAgentTemplate},
	}
}

// agentTemplatePaths lists the injected file paths, for telling the model which
// files already exist and must not be re-emitted.
func agentTemplatePaths() []string {
	return []string{agentClientFilePath, useAgentFilePath}
}

const agentClientTemplate = `import apiClient from '@/config/axios';

export interface AgentRunStep {
  index: number;
  tool_name: string;
  tool_input: Record<string, unknown>;
  tool_result: string;
  is_error: boolean;
}

export interface AgentRun {
  id: string;
  agent_id: string;
  status: 'running' | 'succeeded' | 'failed';
  output: string;
  steps: AgentRunStep[];
  tokens_used: number;
  error: string;
}

export interface RunAgentResult {
  reply: string;
  run: AgentRun;
}

/**
 * runAgent sends a single message to a server-side AI agent and resolves with its
 * reply. Pass optional structured context (e.g. the record the user is viewing) so
 * the agent can ground its answer.
 */
export async function runAgent(
  agentId: string,
  message: string,
  context?: Record<string, unknown>,
): Promise<RunAgentResult> {
  const res = await apiClient.post('/v2/agents/' + agentId + '/run', { message, context });
  const run: AgentRun = res?.data?.data ?? ({} as AgentRun);
  if (run.status === 'failed') {
    throw new Error(run.error || 'Agent run failed');
  }
  return { reply: run.output ?? '', run };
}

export default runAgent;
`

const useAgentTemplate = `import { useCallback, useRef, useState } from 'react';
import { runAgent } from '@/lib/agentClient';

export interface AgentMessage {
  id: string;
  role: 'user' | 'assistant';
  content: string;
}

export interface UseAgentOptions {
  /** Optional structured context sent with every message (e.g. the current record). */
  context?: Record<string, unknown>;
}

export interface UseAgentResult {
  messages: AgentMessage[];
  isLoading: boolean;
  error: string | null;
  send: (text: string) => Promise<void>;
  reset: () => void;
}

let counter = 0;
const nextId = (): string => 'm' + Date.now().toString() + '_' + (counter++).toString();

/**
 * useAgent manages a chat session with a server-side AI agent identified by agentId.
 * It keeps the running transcript, exposes a send() action, and surfaces loading and
 * error state for the UI.
 */
export function useAgent(agentId: string, options: UseAgentOptions = {}): UseAgentResult {
  const [messages, setMessages] = useState<AgentMessage[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const contextRef = useRef(options.context);
  contextRef.current = options.context;

  const send = useCallback(
    async (text: string) => {
      const trimmed = text.trim();
      if (!trimmed || isLoading) return;

      setError(null);
      setIsLoading(true);
      setMessages((prev) => [...prev, { id: nextId(), role: 'user', content: trimmed }]);

      try {
        const { reply } = await runAgent(agentId, trimmed, contextRef.current);
        setMessages((prev) => [...prev, { id: nextId(), role: 'assistant', content: reply }]);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Something went wrong');
      } finally {
        setIsLoading(false);
      }
    },
    [agentId, isLoading],
  );

  const reset = useCallback(() => {
    setMessages([]);
    setError(null);
  }, []);

  return { messages, isLoading, error, send, reset };
}

export default useAgent;
`
