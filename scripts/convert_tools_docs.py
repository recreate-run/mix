#!/usr/bin/env python3
"""
Convert tool description MD files to MDX format with proper frontmatter.
Uses Jinja2 templating for sophisticated content processing.
"""

from pathlib import Path
from jinja2 import Template
import shutil

# Files to ignore during conversion
IGNORE_FILES = ["kill_bash.md"]

# Tools to exclude from documentation (backend-only tools)
EXCLUDED_TOOLS = [
    "bash_output.md",
]

# Jinja2 template for MDX files
MDX_TEMPLATE = Template("""---
title: {{ title }}
description: {{ description }}
---

## Tool Description

```
{{ content }}
```""")


def extract_title(filename):
    """Convert filename to proper title format."""
    # Remove .md extension and convert underscores to spaces
    title = filename.replace(".md", "").replace("_", " ")
    # Capitalize each word
    return " ".join(word.capitalize() for word in title.split())


def extract_description(content):
    """Extract a meaningful description from the content."""
    lines = content.strip().split("\n")

    # Look for the first substantial line of text
    for line in lines:
        line = line.strip()
        # Skip empty lines, headers, and very short lines
        if not line or line.startswith("#") or len(line) < 20:
            continue

        # If line starts with bullet point, clean it up
        if line.startswith("- "):
            line = line[2:]
        elif line.startswith("* "):
            line = line[2:]

        # Clean up and truncate if too long
        description = line.strip()
        if len(description) > 150:
            description = description[:147] + "..."

        return description

    # Fallback description
    return "Tool documentation and usage guide"


def convert_md_to_mdx(source_dir, dest_dir):
    """Convert all MD files in source_dir to MDX files in dest_dir."""
    source_path = Path(source_dir)
    dest_path = Path(dest_dir)

    # Remove existing destination directory if it exists
    if dest_path.exists():
        print(f"Removing existing directory: {dest_path}")
        shutil.rmtree(dest_path)

    # Create fresh destination directory
    dest_path.mkdir(parents=True, exist_ok=True)

    converted_files = []

    # Process each .md file
    for md_file in source_path.glob("*.md"):
        # Skip ignored files
        if md_file.name in IGNORE_FILES:
            print(f"Skipping {md_file.name} (ignored)")
            continue

        # Skip excluded tools
        if md_file.name in EXCLUDED_TOOLS:
            print(f"Skipping {md_file.name} (excluded)")
            continue

        print(f"Processing {md_file.name}...")

        # Read the original content
        with open(md_file, "r", encoding="utf-8") as f:
            content = f.read()

        # Extract metadata
        title = extract_title(md_file.name)
        description = extract_description(content)

        # Generate MDX content using template
        mdx_content = MDX_TEMPLATE.render(
            title=title, description=description, content=content
        )

        # Write directly to destination directory (no subfolders)
        mdx_filename = md_file.stem.replace("_", "-") + ".mdx"
        mdx_file = dest_path / mdx_filename

        with open(mdx_file, "w", encoding="utf-8") as f:
            f.write(mdx_content)

        converted_files.append(mdx_filename)
        print(f"  → Created {mdx_filename}")

    return converted_files


def main():
    """Main execution function."""
    # Define paths
    source_dir = "mix_agent/internal/config/prompts/tools"
    dest_dir = "docs/content/docs/mix/usage/tools"

    print("Converting MD files to MDX with Jinja2 templating...")
    print(f"Source: {source_dir}")
    print(f"Destination: {dest_dir}")
    print("-" * 50)

    # Convert files
    converted_files = convert_md_to_mdx(source_dir, dest_dir)

    print("-" * 50)
    print(f"Conversion complete! Converted {len(converted_files)} files:")
    for filename in sorted(converted_files):
        print(f"  ✓ {filename}")

    print("\nNext step: Update meta.json to include these files in the sidebar")


if __name__ == "__main__":
    main()
