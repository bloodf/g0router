import { createFileRoute } from "@tanstack/react-router";
import { useState, useRef, useEffect } from "react";
import { useQuery } from "@tanstack/react-query";
import { apiFetch } from "@/lib/api/client";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Icon } from "@/components/common/Icon";
import { PageHeader } from "@/components/common/PageHeader";
import { streamMockChat } from "@/lib/mocks/streams";
import { ListRowsSkeleton } from "@/components/common/Skeletons";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import type { ChatSession, Provider } from "@/lib/mocks/types";

export const Route = createFileRoute("/_app/chat")({ component: ChatPage });

type Msg = { role: "user" | "assistant"; content: string };

function ChatPage() {
  const [messages, setMessages] = useState<Msg[]>([]);
  const [input, setInput] = useState("");
  const [streaming, setStreaming] = useState(false);
  const [provider, setProvider] = useState("openai");
  const [model, setModel] = useState("gpt-4o");
  const abortRef = useRef<AbortController | null>(null);
  const scrollRef = useRef<HTMLDivElement>(null);

  const providersQ = useQuery<Provider[]>({
    queryKey: ["providers"],
    queryFn: () => apiFetch("/api/providers"),
  });
  const providers = providersQ.data ?? [];
  const sessionsQ = useQuery<ChatSession[]>({
    queryKey: ["chat-sessions"],
    queryFn: () => apiFetch("/api/chat-sessions"),
  });
  const sessions = sessionsQ.data ?? [];
  const sessionsLoading = sessionsQ.isLoading;

  useEffect(() => {
    scrollRef.current?.scrollTo({
      top: scrollRef.current.scrollHeight,
      behavior: "smooth",
    });
  }, [messages]);

  const send = async () => {
    if (!input.trim() || streaming) return;
    const userMsg = { role: "user" as const, content: input };
    setMessages((m) => [...m, userMsg, { role: "assistant", content: "" }]);
    setInput("");
    setStreaming(true);
    const ctl = new AbortController();
    abortRef.current = ctl;
    streamMockChat(
      input,
      (delta) =>
        setMessages((m) => {
          const next = [...m];
          next[next.length - 1] = {
            ...next[next.length - 1],
            content: next[next.length - 1].content + delta,
          };
          return next;
        }),
      () => setStreaming(false),
      ctl.signal,
    );
  };

  return (
    <div>
      <PageHeader
        title="Chat playground"
        description="Test any connected model with streaming, sessions and tools."
        icon="chat"
      />

      <div className="grid grid-cols-1 lg:grid-cols-[260px_1fr] gap-4 h-[calc(100vh-220px)]">
        <Card className="card-elev border-border p-3 flex flex-col">
          <Button
            onClick={() => setMessages([])}
            className="w-full mb-3"
            variant="outline"
          >
            <Icon name="add" size={16} className="mr-1.5" />
            New chat
          </Button>
          <div className="text-[10px] uppercase tracking-wider text-text-muted mb-1.5 px-1">
            Recent
          </div>
          <div className="space-y-1 overflow-y-auto custom-scrollbar flex-1">
            {sessionsLoading ? (
              <ListRowsSkeleton rows={5} />
            ) : sessions.length === 0 ? (
              <div className="text-xs text-text-muted px-1 py-2">No recent chats</div>
            ) : (
              sessions.map((s) => (
                <button
                  key={s.id}
                  className="w-full text-left p-2 rounded-lg hover:bg-surface-2 text-sm truncate"
                >
                  <div className="truncate font-medium">{s.title}</div>
                  <div className="text-[10px] text-text-muted truncate">
                    {s.provider} · {s.model}
                  </div>
                </button>
              ))
            )}
          </div>
        </Card>

        <Card className="card-elev border-border flex flex-col overflow-hidden">
          <div className="border-b border-border p-3 flex items-center gap-2 flex-wrap">
            <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-brand-400 to-brand-600 flex items-center justify-center text-white font-bold text-sm">
              g0
            </div>
            <select
              value={provider}
              onChange={(e) => setProvider(e.target.value)}
              className="bg-surface-2 border border-border rounded-lg px-2 py-1.5 text-xs"
            >
              {providers.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.display_name}
                </option>
              ))}
            </select>
            <select
              value={model}
              onChange={(e) => setModel(e.target.value)}
              className="bg-surface-2 border border-border rounded-lg px-2 py-1.5 text-xs font-mono"
            >
              <option value="gpt-4o">gpt-4o</option>
              <option value="claude-sonnet-4">claude-sonnet-4</option>
              <option value="gemini-2.5-pro">gemini-2.5-pro</option>
            </select>
          </div>

          <div ref={scrollRef} className="flex-1 overflow-y-auto custom-scrollbar p-6">
            {!messages.length && (
              <div className="text-center text-text-muted py-12">
                <div className="w-14 h-14 rounded-2xl bg-gradient-to-br from-brand-400 to-brand-600 mx-auto mb-3 flex items-center justify-center text-white font-bold text-lg shadow-warm">
                  g0
                </div>
                <p className="text-sm">Ask anything to test {model}</p>
              </div>
            )}
            <div className="space-y-4 max-w-3xl mx-auto">
              {messages.map((m, i) => (
                <div
                  key={i}
                  className={
                    m.role === "user" ? "flex justify-end" : "flex justify-start"
                  }
                >
                  {m.role === "user" ? (
                    <div className="max-w-[80%] bg-primary text-primary-foreground rounded-2xl rounded-tr-md px-4 py-2.5 text-sm">
                      {m.content}
                    </div>
                  ) : (
                    <div className="max-w-full text-sm prose prose-sm dark:prose-invert">
                      {m.content ? (
                        <ReactMarkdown remarkPlugins={[remarkGfm]}>
                          {m.content}
                        </ReactMarkdown>
                      ) : (
                        <span className="text-text-muted italic">Thinking…</span>
                      )}
                    </div>
                  )}
                </div>
              ))}
            </div>
          </div>

          <div className="border-t border-border p-3">
            <div className="flex items-end gap-2 max-w-3xl mx-auto">
              <Input
                value={input}
                onChange={(e) => setInput(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter" && !e.shiftKey) {
                    e.preventDefault();
                    send();
                  }
                }}
                placeholder="Type a message…"
                disabled={streaming}
                autoFocus
              />
              {streaming ? (
                <Button onClick={() => abortRef.current?.abort()} variant="outline" size="icon">
                  <Icon name="stop" />
                </Button>
              ) : (
                <Button onClick={send} size="icon" className="btn-cta">
                  <Icon name="send" />
                </Button>
              )}
            </div>
          </div>
        </Card>
      </div>
    </div>
  );
}
