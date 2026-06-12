import { createRootRoute, Outlet, redirect } from "@tanstack/react-router";
import { useState } from "react";
import { ThemeProvider } from "@/providers/theme";
import { Sidebar } from "@/components/layout/sidebar";
import { MobileSidebar } from "@/components/layout/mobile-sidebar";
import { Header } from "@/components/layout/header";
import { AppToaster } from "@/components/layout/toaster";

function I18nMount({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}

export const Route = createRootRoute({
  beforeLoad: ({ location }) => {
    if (location.pathname === "/") {
      throw redirect({ to: "/dashboard" });
    }
  },
  component: RootComponent,
});

function RootComponent() {
  const [mobileOpen, setMobileOpen] = useState(false);

  return (
    <ThemeProvider>
      <I18nMount>
        <div className="flex h-screen bg-background text-foreground">
          <Sidebar />
          <MobileSidebar open={mobileOpen} onClose={() => setMobileOpen(false)} />
          <div className="flex flex-col flex-1 min-w-0">
            <Header onMenuClick={() => setMobileOpen(true)} />
            <main className="flex-1 overflow-auto p-4">
              <Outlet />
            </main>
          </div>
        </div>
        <AppToaster />
      </I18nMount>
    </ThemeProvider>
  );
}
