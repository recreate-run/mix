export const config = {
  links: {
    github: "https://github.com/recreate-run/mix.git",
    sdkGithub: "https://github.com/recreate-run/mix-typescript-sdk",
    pythonSdkGithub: "https://github.com/recreate-run/mix-python-sdk",
    npm: "https://www.npmjs.com/package/mix-typescript-sdk",
    pypi: "https://pypi.org/project/mix-python-sdk/"
  },
  site: {
    name: "Mix",
    description: "The Production Ready Agents SDK",
    url: process.env.NEXT_PUBLIC_SITE_URL,
  },
} as const;