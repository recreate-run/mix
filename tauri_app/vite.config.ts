import tailwindcss from '@tailwindcss/vite';
import { tanstackRouter } from '@tanstack/router-plugin/vite';
import react from '@vitejs/plugin-react';
import path from 'path';
import { defineConfig } from 'vite';
import { consoleForwardPlugin } from './src/vite-console-forward-plugin';

// @ts-expect-error process is a nodejs global
const host = process.env.TAURI_DEV_HOST;

// https://vitejs.dev/config/
export default defineConfig(async () => ({
  plugins: [
    tanstackRouter({
      target: 'react',
      autoCodeSplitting: true,
    }),
    react(),
    tailwindcss(),
    consoleForwardPlugin({
      // Enable console forwarding (default: true in dev mode)
      enabled: true,

      // Custom API endpoint (default: '/api/debug/client-logs')
      endpoint: '/api/debug/client-logs',

      // Which console levels to forward (default: all)
      levels: ['log', 'warn', 'error', 'info', 'debug'],
    }),
  ],
  define: {
    'import.meta.env.VITE_BACKEND_URL': JSON.stringify('http://localhost:8088'),
  },
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
      '@remotion-shared': path.resolve(
        __dirname,
        '../packages/remotion_template/src'
      ),
    },
  },

  // Vite options tailored for Tauri development and only applied in `tauri dev` or `tauri build`
  //
  // 1. prevent vite from obscuring rust errors
  clearScreen: false,
  // 2. tauri expects a fixed port, fail if that port is not available
  server: {
    port: 1420,
    strictPort: true,
    host,
    hmr: host
      ? {
        protocol: 'ws',
        host,
        port: 1421,
      }
      : undefined,
    watch: {
      // 3. tell vite to ignore watching `src-tauri`
      ignored: ['**/src-tauri/**'],
    },
  },
}));
