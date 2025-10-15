import path from "node:path";
import tailwindcss from "@tailwindcss/vite";
import { tanstackRouter } from "@tanstack/router-plugin/vite";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";
import { consoleForwardPlugin } from "./src/vite-console-forward-plugin";

// https://vitejs.dev/config/
export default defineConfig(async () => ({
	plugins: [
		tanstackRouter({
			target: "react",
			autoCodeSplitting: true,
		}),
		react(),
		tailwindcss(),
		consoleForwardPlugin({
			// Enable console forwarding (default: true in dev mode)
			enabled: true,

			// Custom API endpoint (default: '/api/debug/client-logs')
			endpoint: "/api/debug/client-logs",

			// Which console levels to forward (default: all)
			levels: ["log", "warn", "error", "info", "debug"],
		}),
	],
	resolve: {
		alias: {
			"@": path.resolve(__dirname, "./src"),
		},
	},

	// Development server configuration
	server: {
		port: 3000,
		strictPort: false,
	},
}));
