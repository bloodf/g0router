import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/chat")({
  component: ChatPage,
});

function ChatPage() {
  return <h1>Chat</h1>;
}
