import { useCallback, useEffect, useRef, useState } from "react";
import { getControlPlaneKey, listProviderModels, listProviders, type ProviderMatrixEntry, type ProviderModel } from "../api";
import { EmptyState, ErrorState, Panel } from "../components/Primitives";

type Message = {
  role: "user" | "assistant" | "system";
  content: string;
};

type ChatState =
  | { status: "idle" }
  | { status: "loading"; message: string }
  | { status: "streaming"; message: string }
  | { status: "error"; message: string };

export function ChatPage() {
  const [providers, setProviders] = useState<ProviderMatrixEntry[]>([]);
  const [models, setModels] = useState<ProviderModel[]>([]);
  const [selectedProvider, setSelectedProvider] = useState("");
  const [selectedModel, setSelectedModel] = useState("");
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState("");
  const [chatState, setChatState] = useState<ChatState>({ status: "idle" });
  const [loadError, setLoadError] = useState<string | null>(null);
  const abortRef = useRef<AbortController | null>(null);
  const messagesEndRef = useRef<HTMLDivElement>(null);

  // Load providers on mount
  useEffect(() => {
    let cancelled = false;
    async function load() {
      try {
        const data = await listProviders();
        if (!cancelled) {
          const inferenceProviders = data.filter((p) => p.inference);
          setProviders(inferenceProviders);
          if (inferenceProviders.length > 0) {
            setSelectedProvider(inferenceProviders[0].id);
          }
        }
      } catch (err) {
        if (!cancelled) {
          setLoadError(err instanceof Error ? err.message : String(err));
        }
      }
    }
    void load();
    return () => {
      cancelled = true;
    };
  }, []);

  // Load models when provider changes
  useEffect(() => {
    if (!selectedProvider) {
      setModels([]);
      setSelectedModel("");
      return;
    }
    let cancelled = false;
    async function load() {
      try {
        const data = await listProviderModels(selectedProvider);
        if (!cancelled) {
          setModels(data);
          if (data.length > 0) {
            setSelectedModel(data[0].id);
          } else {
            setSelectedModel("");
          }
        }
      } catch {
        if (!cancelled) {
          setModels([]);
          setSelectedModel("");
        }
      }
    }
    void load();
    return () => {
      cancelled = true;
    };
  }, [selectedProvider]);

  // Scroll to bottom on new messages
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView?.({ behavior: "smooth" });
  }, [messages]);

  const sendMessage = useCallback(async () => {
    const text = input.trim();
    if (!text || !selectedModel) return;

    const key = getControlPlaneKey();
    if (!key) {
      setChatState({ status: "error", message: "Save a control-plane API key first." });
      return;
    }

    const userMessage: Message = { role: "user", content: text };
    const newMessages = [...messages, userMessage];
    setMessages(newMessages);
    setInput("");
    setChatState({ status: "streaming", message: "" });

    const modelId = `${selectedProvider}/${selectedModel}`;
    const abort = new AbortController();
    abortRef.current = abort;

    try {
      const response = await fetch("/v1/chat/completions", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${key}`
        },
        body: JSON.stringify({
          model: modelId,
          messages: newMessages,
          stream: true
        }),
        signal: abort.signal
      });

      if (!response.ok) {
        const payload = await response.json().catch(() => ({ error: "Unknown error" }));
        setChatState({ status: "error", message: payload.error || `HTTP ${response.status}` });
        return;
      }

      const reader = response.body?.getReader();
      if (!reader) {
        setChatState({ status: "error", message: "No response body" });
        return;
      }

      const decoder = new TextDecoder();
      let buffer = "";
      let assistantContent = "";

      try {
        while (true) {
          const { done, value } = await reader.read();
          if (done) break;
          buffer += decoder.decode(value, { stream: true });

          const lines = buffer.split("\n");
          buffer = lines.pop() ?? "";

          for (const line of lines) {
            const trimmed = line.trim();
            if (!trimmed || trimmed.startsWith(":")) continue;
            if (trimmed === "data: [DONE]") continue;
            if (trimmed.startsWith("data: ")) {
              const json = trimmed.slice(6);
              try {
                const parsed = JSON.parse(json) as {
                  choices?: Array<{ delta?: { content?: string } }>;
                };
                const delta = parsed.choices?.[0]?.delta?.content;
                if (delta) {
                  assistantContent += delta;
                  setChatState({ status: "streaming", message: assistantContent });
                }
              } catch {
                // Ignore malformed JSON frames.
              }
            }
          }
        }
      } finally {
        reader.releaseLock();
      }

      if (assistantContent) {
        setMessages((prev) => [...prev, { role: "assistant", content: assistantContent }]);
      }
      setChatState({ status: "idle" });
    } catch (err) {
      if (abort.signal.aborted) {
        setChatState({ status: "idle" });
        return;
      }
      setChatState({ status: "error", message: err instanceof Error ? err.message : String(err) });
    } finally {
      abortRef.current = null;
    }
  }, [input, selectedModel, selectedProvider, messages]);

  const handleStop = useCallback(() => {
    abortRef.current?.abort();
    abortRef.current = null;
    setChatState({ status: "idle" });
  }, []);

  const handleKeyDown = useCallback(
    (event: React.KeyboardEvent<HTMLTextAreaElement>) => {
      if (event.key === "Enter" && !event.shiftKey) {
        event.preventDefault();
        void sendMessage();
      }
    },
    [sendMessage]
  );

  const isStreaming = chatState.status === "streaming";
  const isLoading = chatState.status === "loading";

  if (loadError) {
    return <ErrorState title="Could not load providers" message={loadError} />;
  }

  return (
    <Panel title="Playground" description="Test chat completions with any configured provider and model.">
      <div className="flex flex-col gap-4">
        {/* Provider + Model selectors */}
        <div className="flex flex-col gap-3 sm:flex-row">
          <label className="block flex-1 text-sm font-medium text-zinc-700">
            Provider
            <select
              value={selectedProvider}
              onChange={(e) => setSelectedProvider(e.target.value)}
              className="mt-1 block w-full rounded-md border border-zinc-300 px-3 py-2 text-sm outline-none focus:border-zinc-400"
            >
              {providers.length === 0 && <option value="">No providers</option>}
              {providers.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.id}
                </option>
              ))}
            </select>
          </label>
          <label className="block flex-1 text-sm font-medium text-zinc-700">
            Model
            <select
              value={selectedModel}
              onChange={(e) => setSelectedModel(e.target.value)}
              className="mt-1 block w-full rounded-md border border-zinc-300 px-3 py-2 text-sm outline-none focus:border-zinc-400"
            >
              {models.length === 0 && <option value="">No models</option>}
              {models.map((m) => (
                <option key={m.id} value={m.id}>
                  {m.id}
                </option>
              ))}
            </select>
          </label>
        </div>

        {/* Messages */}
        <div className="flex h-[28rem] flex-col gap-3 overflow-y-auto rounded-md border border-zinc-200 bg-white p-4">
          {messages.length === 0 && (
            <EmptyState
              title="Start a conversation"
              description="Select a provider and model, then type a message to test inference."
            />
          )}
          {messages.map((msg, index) => (
            <div
              key={index}
              className={`max-w-[80%] rounded-lg px-4 py-2 text-sm ${
                msg.role === "user"
                  ? "self-end bg-zinc-950 text-white"
                  : "self-start bg-zinc-100 text-zinc-900"
              }`}
            >
              <p className="text-xs font-semibold opacity-70 mb-1">
                {msg.role === "user" ? "You" : "Assistant"}
              </p>
              <div className="whitespace-pre-wrap">{msg.content}</div>
            </div>
          ))}
          {isStreaming && (
            <div className="max-w-[80%] self-start rounded-lg bg-zinc-100 px-4 py-2 text-sm text-zinc-900">
              <p className="text-xs font-semibold opacity-70 mb-1">Assistant</p>
              <div className="whitespace-pre-wrap">{chatState.message}</div>
              <span className="inline-block h-4 w-1 animate-pulse bg-zinc-400" />
            </div>
          )}
          <div ref={messagesEndRef} />
        </div>

        {/* Input */}
        <div className="flex flex-col gap-2">
          {chatState.status === "error" && (
            <ErrorState title="Chat error" message={chatState.message} />
          )}
          <div className="flex gap-2">
            <textarea
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder="Type a message... (Enter to send, Shift+Enter for new line)"
              rows={2}
              disabled={isStreaming || isLoading}
              className="min-h-[3.5rem] flex-1 resize-y rounded-md border border-zinc-300 px-3 py-2 text-sm outline-none focus:border-zinc-400 disabled:bg-zinc-50"
            />
            <div className="flex flex-col gap-2">
              {isStreaming ? (
                <button
                  type="button"
                  onClick={handleStop}
                  className="min-h-10 rounded-md border border-zinc-300 px-4 py-2 text-sm font-semibold text-zinc-700 hover:bg-zinc-100"
                >
                  Stop
                </button>
              ) : (
                <button
                  type="button"
                  onClick={() => void sendMessage()}
                  disabled={!input.trim() || !selectedModel}
                  className="min-h-10 rounded-md bg-zinc-950 px-4 py-2 text-sm font-semibold text-white disabled:cursor-not-allowed disabled:bg-zinc-400"
                >
                  Send
                </button>
              )}
            </div>
          </div>
        </div>
      </div>
    </Panel>
  );
}
