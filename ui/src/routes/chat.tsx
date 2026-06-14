import { createFileRoute } from "@tanstack/react-router";
import { Card } from "@/components/ui/card";
import { ChatWindow } from "@/components/chat/chat-window";

export const Route = createFileRoute("/chat")({
  component: ChatPage,
});

function ChatPage() {
  return (
    <div className="flex flex-col gap-6">
      <h1 className="text-2xl font-semibold text-foreground">Chat</h1>
      <Card>
        <ChatWindow />
      </Card>
    </div>
  );
}
