import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import { TanStackRouterVite } from "@tanstack/router-plugin/vite";
import tailwindcss from "@tailwindcss/vite";
import tsconfigPaths from "vite-tsconfig-paths";

export default defineConfig({
  plugins: [
    TanStackRouterVite(),
    react(),
    tailwindcss(),
    tsconfigPaths(),
  ],
  build: {
    chunkSizeWarningLimit: 1000,
    sourcemap: true,
  },
  server: {
    proxy: {
      "/api": "http://127.0.0.1:20129",
      "/v1": "http://127.0.0.1:20129",
      "/healthz": "http://127.0.0.1:20129",
    },
  },
});
