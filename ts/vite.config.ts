import { defineConfig } from "vite";
import path from "node:path";

export default defineConfig({
  root: path.resolve(import.meta.dirname, ".."),
  server: {
    host: "0.0.0.0",
    port: 4020,
    allowedHosts: ["a4100b3d.naori.cloud"],
  },
  resolve: {
    alias: {
      "@zod2go/export": path.resolve(import.meta.dirname, "src/index.ts"),
    },
  },
  publicDir: false,
});
