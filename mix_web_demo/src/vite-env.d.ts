/// <reference types="vite/client" />

interface ImportMetaEnv {
	readonly VITE_BACKEND_URL: string;
	readonly VITE_GSAP_URL?: string;
	// more env variables...
}

interface ImportMeta {
	readonly env: ImportMetaEnv;
}
