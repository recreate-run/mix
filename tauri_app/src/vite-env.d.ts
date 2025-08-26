/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_BACKEND_URL: string
  readonly VITE_GSAP_SERVER_URL: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
