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
- Structure (comprehensive reference format):
  1. Installation - Package installation instructions
  2. API Selection Guide - Comparison table helping users choose between different APIs (e.g., `query()` vs `ClaudeSDKClient`)
  3. Functions - Top-level functions with parameters, return types, examples
  4. Classes - Main classes with methods, lifecycle, context manager support
  5. Types - All configuration types, message types, content blocks, errors, hooks with complete property tables
  6. Tool Input/Output Types - Complete schemas for all built-in tools (document structure, not exportable types)
  7. Advanced Features - Real examples: continuous conversations, hooks, progress monitoring, custom tools
  8. Example Usage - Common patterns with complete working code
- Use progressive disclosure: simple APIs → configuration → advanced patterns
- Include comparison tables for API/method selection decisions
- Document ALL parameters with tables (parameter, type, default, description)
- Provide working examples at each complexity level
- Link to [mix-cookbooks](https://github.com/recreate-run/mix-cookbooks/tree/main/) for additional recipes under installation/quickstart

Architecture pages:

- Purely conceptual - NO code snippets, NO how-to instructions
- Explain system design, patterns, and decisions only

Guides pages:

- Practical step-by-step tutorials with minimal paragraph intro (NOT bulleted)
- Self-contained, concise, non-ambiguous steps
- NO pros/cons sections
- MUST include: 3+ screenshot/video placeholders

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

Steps (sequential instructions):

```markdown
import { Step, Steps } from 'fumadocs-ui/components/steps';

<Steps>
<Step>
### Step Title
Content here
</Step>
</Steps>
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
