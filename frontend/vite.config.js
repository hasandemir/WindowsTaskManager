import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
export default defineConfig({
    plugins: [react(), tailwindcss()],
    build: {
        outDir: "dist",
        emptyOutDir: true,
        sourcemap: false,
        rollupOptions: {
            output: {
                manualChunks(id) {
                    if (id.includes("node_modules")) {
                        if (id.includes("react") || id.includes("react-dom") || id.includes("react-router")) {
                            return "vendor";
                        }
                        if (id.includes("@tanstack/react-query") || id.includes("zustand") || id.includes("react-hook-form") || id.includes("zod")) {
                            return "data";
                        }
                    }
                    return undefined;
                },
            },
        },
    },
    server: {
        proxy: {
            "/api": "http://127.0.0.1:19876",
        },
    },
    test: {
        environment: "jsdom",
        setupFiles: ["./src/test/setup.ts"],
        css: true,
        globals: true,
    },
});
