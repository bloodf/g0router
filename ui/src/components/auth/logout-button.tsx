import { useNavigate } from "@tanstack/react-router";
import { LogOut } from "lucide-react";
import { Button } from "@/components/ui/button";
import { logout } from "@/lib/auth";
import { useUserStore } from "@/stores/user";

export function LogoutButton() {
  const navigate = useNavigate();
  const clear = useUserStore((state) => state.clear);

  async function handleLogout() {
    await logout();
    clear();
    navigate({ to: "/login" });
  }

  return (
    <Button
      variant="ghost"
      size="icon"
      aria-label="Log out"
      data-testid="logout-button"
      onClick={handleLogout}
    >
      <LogOut />
    </Button>
  );
}
