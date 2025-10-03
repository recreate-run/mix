# Documentation Guidelines

## Structure

```
docs/content/docs/
├── mix/           # Platform docs
│   ├── tools/
│   ├── python-sdk/
│   └── meta.json  # Required for folders with multiple pages
└── api/           # Auto-generated from OpenAPI
```

## Files

- Naming: `kebab-case.mdx` (NOT `.md`)
- Frontmatter: Required `title` and `description` (1-2 sentences max)

```yaml
---
title: Page title
description: Brief description
---
```

## Navigation (meta.json)

```json
{
  "title": "Mix",
  "icon": "BookOpen",
  "root": true,
  "pages": [
    "---Getting Started---",
    "index",
    "quickstart",
    "---Architecture---",
    "architecture-overview"
  ]
}
```

Key rules:

- Use `---Section Name---` for section headers
- Page names match filenames WITHOUT `.mdx` extension
- `root: true` for top-level sections

## Content Organization

Follow `meta.json` section structure: Getting Started → SDKs → Usage → Development → Others. Follow these rules strictly:

SDK pages:

- Single .mdx file per language (NOT a folder)
- Structure: Installation → Quickstart → Code snippets for various tasks
- NO "Examples" section - link to [mix-cookbooks](https://github.com/recreate-run/mix-cookbooks/tree/main/) under Quickstart instead

Architecture pages:

- Purely conceptual - NO code snippets, NO how-to instructions
- Explain system design, patterns, and decisions only

Guides pages:

- Practical step-by-step tutorials with minimal paragraph intro (NOT bulleted)
- Self-contained, concise, non-ambiguous steps
- NO pros/cons sections
- MUST include: 3+ screenshot/video placeholders
- troubleshooting section at the very end

## Content

### Headings

- NEVER use H1 (`#`) - Auto-generated from frontmatter
- Start with H2 (`##`)

### Code Blocks

- Always specify language: ` ```bash `, ` ```python `, ` ```typescript `

### Links

- Internal: Must start with `/docs/` → `[link](/docs/mix/quickstart)`
- External: Full URL → `[link](https://github.com/recreate-run/mix)`

### Components

Cards (Next Steps sections):

```markdown
<Cards>
  <Card title="Title" href="/docs/path">
    Description
  </Card>
</Cards>
```

Mermaid (diagrams):

```markdown
<Mermaid
  chart="
graph TB
    A[Start] --> B[End]
"
/>
```

## API Documentation

DO NOT manually edit `content/docs/api/` - Auto-generated from OpenAPI specs.

To update API docs:

1. Modify backend in `mix_agent/internal/http/`
2. ALWAYS update `mix_agent/internal/http/rest_docs.go`
3. Regenerate docs

## Style

- Active voice, present tense
- Start with practical examples, not theory
- Include "Next Steps" section with Cards
- NO emojis anywhere in documentation
