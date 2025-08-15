# mix-docs

This is a Next.js application generated with
[Create Fumadocs](https://github.com/fuma-nama/fumadocs).

Run development server:

```bash
npm run dev
# or
pnpm dev
# or
yarn dev
```

Open <http://localhost:3000> with your browser to see the result.

## Explore

In the project, you can see:

- `lib/source.ts`: Code for content source adapter, [`loader()`](https://fumadocs.dev/docs/headless/source-api) provides the interface to access your content.
- `app/layout.config.tsx`: Shared options for layouts, optional but preferred to keep.

| Route                     | Description                                            |
| ------------------------- | ------------------------------------------------------ |
| `app/(home)`              | The route group for your landing page and other pages. |
| `app/docs`                | The documentation layout and pages.                    |
| `app/api/search/route.ts` | The Route Handler for search.                          |

### Fumadocs MDX

A `source.config.ts` config file has been included, you can customise different options like frontmatter schema.

Read the [Introduction](https://fumadocs.dev/docs/mdx) for further details.

## Learn More

To learn more about Next.js and Fumadocs, take a look at the following
resources:

- [Next.js Documentation](https://nextjs.org/docs) - learn about Next.js
  features and API.
- [Learn Next.js](https://nextjs.org/learn) - an interactive Next.js tutorial.
- [Fumadocs](https://fumadocs.vercel.app) - learn about Fumadocs

## Tool Documentation

Auto-generates tool documentation from Go backend descriptions.

```bash
uv run --with jinja2 scripts/convert_tools_docs.py
```

## Creating Sidebar Subgroups with Popover Behavior

To create expandable sidebar subgroups (like "System Tools" under "Tools"):

1. **Create a root folder** with `meta.json` containing `"root": true`:
   ```json
   {
     "title": "System Tools",
     "description": "Core system tools for file management",
     "icon": "Terminal",
     "root": true,
     "pages": ["bash", "ls", "grep", "edit"]
   }
   ```

2. **Move related pages** into the subfolder (e.g., `tools/system-tools/`)

3. **Update parent meta.json** to reference the folder:
   ```json
   {
     "pages": ["tools/system-tools"]
   }
   ```

4. **Add icon to source.ts** if needed:
   ```ts
   import { Terminal } from 'lucide-react';
   const icons = { Terminal };
   ```

The fumadocs DocsLayout will automatically detect root folders and create popover navigation.
