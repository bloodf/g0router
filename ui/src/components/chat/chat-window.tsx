import * as React from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { ChatMessage } from "./chat-message";
import { useUserStore } from "@/stores/user";

export interface ChatTurn {
  role: string;
  content: string;
}

export interface StreamChatOptions {
  url: string;
  body: Record<string, unknown>;
  onDelta: (delta: string) => void;
  fetchFn?: typeof fetch;
  headers?: Record<string, string>;
}

// Pure plain-fetch streaming reader for the OpenAI-compatible inference route
// (w6-i §1.3). @ai-sdk/react@3's DefaultChatTransport expects the AI SDK
// UI-message stream protocol, not raw OpenAI SSE — so the chosen approach is this
// ReadableStream reader, which maps cleanly to the inference.ts mock chunk shape
// (chat.completion.chunk → choices[].delta.content) and adds NO dependency.
export async function streamChatCompletion(opts: StreamChatOptions): Promise<void> {
  const { url, body, onDelta, fetchFn = fetch, headers = {} } = opts;
  const res = await fetchFn(url, {
    method: "POST",
    headers: { "Content-Type": "application/json", ...headers },
    body: JSON.stringify(body),
  });
  if (!res.body) return;
  const reader = res.body.getReader();
  const decoder = new TextDecoder();
  let buffer = "";
  for (;;) {
    const { value, done } = await reader.read();
    if (done) break;
    buffer += decoder.decode(value, { stream: true });
    const lines = buffer.split("\n");
    buffer = lines.pop() ?? "";
    for (const raw of lines) {
      const line = raw.trim();
      if (!line.startsWith("data:")) continue;
      const data = line.slice(5).trim();
      if (data === "[DONE]") return;
      try {
        const chunk = JSON.parse(data) as {
          choices?: Array<{ delta?: { content?: string } }>;
        };
        const delta = chunk.choices?.[0]?.delta?.content;
        if (delta) onDelta(delta);
      } catch {
        // ignore malformed chunk
      }
    }
  }
}

const MODEL_OPTIONS = [
  { value: "openai/gpt-4o", label: "OpenAI · gpt-4o" },
  { value: "openai/gpt-4o-mini", label: "OpenAI · gpt-4o-mini" },
  { value: "anthropic/claude-sonnet-4", label: "Anthropic · claude-sonnet-4" },
  { value: "groq/llama-3.3-70b-versatile", label: "Groq · llama-3.3-70b" },
];

export function ChatWindow() {
  const [messages, setMessages] = React.useState<ChatTurn[]>([]);
  const [draft, setDraft] = React.useState("");
  const [selection, setSelection] = React.useState(MODEL_OPTIONS[0].value);
  const [sending, setSending] = React.useState(false);
  const token = useUserStore((s) => s.token);

  async function send() {
    const text = draft.trim();
    if (!text || sending) return;
    const [provider, model] = selection.split("/");
    const userTurn: ChatTurn = { role: "user", content: text };
    const history = [...messages, userTurn];
    setMessages(history);
    setDraft("");
    setSending(true);
    // Append an empty assistant turn we stream into.
    setMessages((prev) => [...prev, { role: "assistant", content: "" }]);
    try {
      await streamChatCompletion({
        url: `${window.location.origin}/v1/chat/completions`,
        headers: token ? { Authorization: `Bearer ${token}` } : {},
        body: {
          model,
          provider,
          stream: true,
          messages: history.map((m) => ({ role: m.role, content: m.content })),
        },
        onDelta: (delta) => {
          setMessages((prev) => {
            const next = [...prev];
            const last = next[next.length - 1];
            if (last && last.role === "assistant") {
              next[next.length - 1] = {
                ...last,
                content: last.content + delta,
              };
            }
            return next;
          });
        },
      });
    } finally {
      setSending(false);
    }
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="max-w-xs">
        <Select
          data-testid="chat-model-select"
          aria-label="Model"
          value={selection}
          onChange={(e) => setSelection(e.target.value)}
          options={MODEL_OPTIONS}
        />
      </div>

      <div
        data-testid="chat-message-list"
        className="flex flex-col gap-2 rounded-lg border border-border bg-background p-3"
      >
        {messages.length === 0 ? (
          <p className="text-sm text-muted-foreground">
            Start a conversation with the gateway.
          </p>
        ) : (
          messages.map((m, i) => (
            <ChatMessage key={i} role={m.role} content={m.content} />
          ))
        )}
      </div>

      <form
        className="flex items-end gap-2"
        onSubmit={(e) => {
          e.preventDefault();
          void send();
        }}
      >
        <div className="flex-1">
          <Input
            aria-label="Message"
            placeholder="Type a message…"
            value={draft}
            disabled={sending}
            onChange={(e) => setDraft(e.target.value)}
          />
        </div>
        <Button type="submit" disabled={sending}>
          Send
        </Button>
      </form>
    </div>
  );
}
