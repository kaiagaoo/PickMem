import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// The build lands in the Go server's embed dir. `base: "./"` makes asset
// URLs relative so the SPA works no matter what host:port `pickmem web`
// binds to. In dev, /api is proxied to a locally-running `pickmem web`.
export default defineConfig({
  plugins: [react()],
  base: "./",
  build: {
    outDir: "../internal/web/static",
    emptyOutDir: true,
  },
  server: {
    proxy: {
      "/api": "http://127.0.0.1:4577",
    },
  },
});
