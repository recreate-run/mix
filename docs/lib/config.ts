export const config = {
  links: {
    github: "https://github.com/recreate-run/mix.git",
  },
  site: {
    name: "Mix",
    description: "Claude Code for Complex Multimodal Workflows",
    url: process.env.NEXT_PUBLIC_SITE_URL || "http://localhost:3000",
  },
} as const;