package v1

import "ucode/ucode_go_api_gateway/api/models"

// The builder assistant's networking layer, injected verbatim so the model only
// builds the UI that consumes it: an SSE streaming client over the app's shared axios
// instance, and a useBuilderAgent() hook that surfaces the live build steps.
const (
	builderAgentClientFilePath = "src/lib/builderAgentClient.ts"
	useBuilderAgentFilePath    = "src/hooks/useBuilderAgent.ts"
)

// builderAgentTemplateFiles returns the authoritative template files: when merged
// with the model's output, these win over any same-path file it produced.
func builderAgentTemplateFiles() []models.ProjectFile {
	return []models.ProjectFile{
		{Path: builderAgentClientFilePath, Content: builderAgentClientTemplate},
		{Path: useBuilderAgentFilePath, Content: useBuilderAgentTemplate},
	}
}

// builderAgentTemplatePaths lists the injected paths so the model knows not to
// re-emit them.
func builderAgentTemplatePaths() []string {
	return []string{builderAgentClientFilePath, useBuilderAgentFilePath}
}

const builderAgentClientTemplate = `import apiClient from '@/config/axios';

/** One progress event streamed while the builder assistant works. */
export interface BuilderEvent {
  type: string;
  message?: string;
  value?: string;
  icon?: string;
  percent?: number;
  data?: unknown;
}

/** Tally of everything the assistant built during a turn. */
export interface BuilderSummary {
  tables: number;
  login_tables: number;
  client_types: number;
  roles: number;
  fields: number;
  relations: number;
  menus: number;
  items: number;
}

export interface BuilderChatMessage {
  role: 'user' | 'assistant';
  content: string;
}

export interface SendBuilderResult {
  reply: string;
  summary?: BuilderSummary;
}

export interface SendBuilderHandlers {
  /** Called for every progress event as the assistant streams its work. */
  onEvent?: (event: BuilderEvent) => void;
  /** Abort the in-flight request (e.g. when the user closes the widget). */
  signal?: AbortSignal;
}

/**
 * sendBuilderMessage streams one message to the built-in u-code builder assistant.
 * Build steps arrive through handlers.onEvent while the request is in flight; the
 * promise resolves with the assistant's final reply and a summary of what it built.
 * The conversation is stateless — pass prior turns as history for follow-up context.
 *
 * The request goes through the app's shared axios instance (so it authenticates and
 * uses the same base URL as every other call) and reads the SSE stream incrementally
 * via onDownloadProgress. If a proxy buffers the response, the frames are parsed from
 * the final body instead — the reply is delivered either way.
 */
export async function sendBuilderMessage(
  content: string,
  history: BuilderChatMessage[] = [],
  handlers: SendBuilderHandlers = {},
): Promise<SendBuilderResult> {
  let cursor = 0;
  let reply = '';
  let summary: BuilderSummary | undefined;
  let streamError: string | undefined;

  const handleFrame = (frame: string): void => {
    // One SSE frame; only the "data:" line matters (": keepalive" comments are ignored).
    const dataLine = frame.split('\n').find((line) => line.startsWith('data:'));
    if (!dataLine) return;

    const payload = dataLine.slice(5).trim();
    if (!payload) return;

    let event: BuilderEvent;
    try {
      event = JSON.parse(payload) as BuilderEvent;
    } catch {
      return;
    }

    handlers.onEvent?.(event);

    if (event.type === 'done') {
      reply = event.message ?? reply;
      const data = event.data as { reply?: string; summary?: BuilderSummary } | undefined;
      if (data?.reply) reply = data.reply;
      if (data?.summary) summary = data.summary;
    } else if (event.type === 'error') {
      streamError = event.message || 'Builder failed';
    }
  };

  // Consume every complete '\n\n'-terminated frame past the cursor. Called on each
  // progress tick (accumulating body) and once more on the final body, so it is safe
  // to run repeatedly against the same growing text.
  const consume = (text: string): void => {
    let boundary = text.indexOf('\n\n', cursor);
    while (boundary !== -1) {
      handleFrame(text.slice(cursor, boundary));
      cursor = boundary + 2;
      boundary = text.indexOf('\n\n', cursor);
    }
  };

  const response = await apiClient.post('/v2/ai-builder/messages?stream=true', { content, history }, {
    responseType: 'text',
    headers: { Accept: 'text/event-stream' },
    // A build can take a while server-side; never let the client cancel it early.
    timeout: 25 * 60 * 1000,
    signal: handlers.signal,
    onDownloadProgress: (progressEvent) => {
      const xhr = progressEvent.event?.target as XMLHttpRequest | undefined;
      if (typeof xhr?.responseText === 'string') consume(xhr.responseText);
    },
  });

  if (typeof response?.data === 'string') consume(response.data);

  if (streamError) throw new Error(streamError);
  return { reply, summary };
}

export default sendBuilderMessage;
`

const useBuilderAgentTemplate = `import { useCallback, useRef, useState } from 'react';
import {
  sendBuilderMessage,
  type BuilderChatMessage,
  type BuilderEvent,
  type BuilderSummary,
} from '@/lib/builderAgentClient';

export interface BuilderMessage {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  /** Present on the assistant turn that built something: what it created. */
  summary?: BuilderSummary;
}

export interface UseBuilderAgentResult {
  messages: BuilderMessage[];
  /** Live build steps for the in-flight turn; cleared when the turn completes. */
  steps: BuilderEvent[];
  isLoading: boolean;
  error: string | null;
  send: (text: string) => Promise<void>;
  reset: () => void;
}

// The server keeps a bounded context window, so cap the history we replay.
const HISTORY_LIMIT = 20;

let counter = 0;
const nextId = (): string => 'b' + Date.now().toString() + '_' + (counter++).toString();

function toHistory(messages: BuilderMessage[]): BuilderChatMessage[] {
  return messages
    .slice(-HISTORY_LIMIT)
    .map((m) => ({ role: m.role, content: m.content }));
}

/**
 * useBuilderAgent manages a chat session with the built-in u-code builder assistant:
 * the transcript, the live build steps of the current turn, a send() action, and
 * loading/error state.
 */
export function useBuilderAgent(): UseBuilderAgentResult {
  const [messages, setMessages] = useState<BuilderMessage[]>([]);
  const [steps, setSteps] = useState<BuilderEvent[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const messagesRef = useRef<BuilderMessage[]>([]);
  messagesRef.current = messages;

  const send = useCallback(async (text: string) => {
    const trimmed = text.trim();
    if (!trimmed || isLoading) return;

    setError(null);
    setSteps([]);
    setIsLoading(true);

    const history = toHistory(messagesRef.current);
    setMessages((prev) => [...prev, { id: nextId(), role: 'user', content: trimmed }]);

    try {
      const { reply, summary } = await sendBuilderMessage(trimmed, history, {
        onEvent: (event) => {
          if (event.type === 'done' || event.type === 'error' || event.type === 'provider') return;
          if (!event.message && !event.value) return;
          setSteps((prev) => [...prev, event]);
        },
      });
      const assistant: BuilderMessage = { id: nextId(), role: 'assistant', content: reply };
      if (summary) assistant.summary = summary;
      setMessages((prev) => [...prev, assistant]);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Something went wrong');
    } finally {
      setSteps([]);
      setIsLoading(false);
    }
  }, [isLoading]);

  const reset = useCallback(() => {
    setMessages([]);
    setSteps([]);
    setError(null);
  }, []);

  return { messages, steps, isLoading, error, send, reset };
}

export default useBuilderAgent;
`
