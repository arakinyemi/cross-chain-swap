import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// The Go binary serves the built interface, so the bundle lands where
// internal/ui embeds it. In dev, Vite proxies the API to the running server.
//
// Keys starting with "^" are treated as regexes. /assets has to be exact:
// as a prefix it would also swallow the bundle's own /assets/* requests.
const target = "http://localhost:8080";
const apiRoutes = ["^/assets$", "^/quote$", "^/prices$", "^/qr$", "^/swap(/.*)?$"];

export default defineConfig({
  plugins: [react()],
  build: {
    outDir: "../internal/ui/dist",
    emptyOutDir: true,
  },
  server: {
    port: 5173,
    proxy: Object.fromEntries(
      apiRoutes.map((route) => [route, { target, changeOrigin: true }]),
    ),
  },
});
