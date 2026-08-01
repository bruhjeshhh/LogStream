import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

export default defineConfig({
  plugins: [react(), tailwindcss()],
  base: process.env.GH_PAGES === "true" ? "/LogStream/" : "/",
  server: {
    port: 5173,
  },
});
