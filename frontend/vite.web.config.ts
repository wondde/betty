import { defineConfig } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";

// Web build: the same UI as the app, with the Wails bindings and runtime swapped for
// src/web (fetch against the BerryTicket server). Output goes to dist-web, which the
// BerryTicket server serves at "/".
const here = (p: string) => fileURLToPath(new URL(p, import.meta.url));
const version = /const version = "([^"]+)"/.exec(readFileSync(here("../betty.go"), "utf8"))?.[1] ?? "dev";

export default defineConfig({
  mode: "web",
  define: { __BETTY_VERSION__: JSON.stringify(version) },
  build: { outDir: "dist-web", emptyOutDir: true },
  plugins: [
    svelte(),
    {
      name: "betty-web-bindings",
      enforce: "pre",
      resolveId(source) {
        if (source === "@wailsio/runtime") return here("./src/web/runtime.ts");
        if (/bindings\/events$/.test(source)) return here("./src/web/events.ts");
        if (/bindings\/github\.com\/wondde\/betty$/.test(source)) return here("./src/web/index.ts");
        return null;
      },
    },
  ],
});
