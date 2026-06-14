import { Badge } from "@/components/ui/badge";

export interface ChatMessageProps {
  role: string;
  content: string;
}

// Renders one {role, content} turn. The assistant turns carry a data-testid the
// chat e2e asserts on (chat.spec.ts).
export function ChatMessage({ role, content }: ChatMessageProps) {
  const isAssistant = role === "assistant";
  return (
    <div
      data-testid={`chat-message-${role}`}
      className="flex flex-col gap-1 rounded-lg border border-border/50 bg-card px-3 py-2"
    >
      <Badge variant={isAssistant ? "primary" : "neutral"} size="sm">
        {role}
      </Badge>
      <p className="whitespace-pre-wrap text-sm text-foreground">{content}</p>
    </div>
  );
}
