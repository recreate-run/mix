export const config = {
  links: {
    github: "https://github.com/recreate-run/mix.git",
    sdkGithub: "https://github.com/recreate-run/mix-typescript-sdk",
    npm: "https://www.npmjs.com/package/mix-typescript-sdk"
  },
  site: {
    name: "Mix",
    description: "Claude Code for Multimodal tasks",
    url: process.env.NEXT_PUBLIC_SITE_URL,
  },
} as const;