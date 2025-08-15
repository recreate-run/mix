#!/usr/bin/env python3
"""
Convert tool description MD files to MDX format with proper frontmatter.
Uses Jinja2 templating for sophisticated content processing.
"""

import os
import re
from pathlib import Path
from jinja2 import Template

# Files to ignore during conversion
IGNORE_FILES = ['kill_bash.md', 'remotion.md']

# Tools to exclude from documentation (backend-only tools)
EXCLUDED_TOOLS = ['todo_write_cc.md', 'web_search.md', 'bash_output.md']

# Jinja2 template for MDX files
MDX_TEMPLATE = Template("""---
title: {{ title }}
description: {{ description }}
category: {{ category }}
---

## Tool Description

```
{{ content }}
```""")

def extract_title(filename):
    """Convert filename to proper title format."""
    # Remove .md extension and convert underscores to spaces
    title = filename.replace('.md', '').replace('_', ' ')
    # Capitalize each word
    return ' '.join(word.capitalize() for word in title.split())

def extract_description(content):
    """Extract a meaningful description from the content."""
    lines = content.strip().split('\n')
    
    # Look for the first substantial line of text
    for line in lines:
        line = line.strip()
        # Skip empty lines, headers, and very short lines
        if not line or line.startswith('#') or len(line) < 20:
            continue
        
        # If line starts with bullet point, clean it up
        if line.startswith('- '):
            line = line[2:]
        elif line.startswith('* '):
            line = line[2:]
        
        # Clean up and truncate if too long
        description = line.strip()
        if len(description) > 150:
            description = description[:147] + '...'
        
        return description
    
    # Fallback description
    return "Tool documentation and usage guide"

def determine_category(filename, content):
    """Determine the category based on filename and content."""
    # Only blender, pixelmator, and notes are App Tools
    app_tools = ['blender', 'pixelmator', 'notes']
    
    base_name = filename.replace('.md', '')
    
    if base_name in app_tools:
        return "App Tools"
    else:
        # Everything else is a System Tool
        return "System Tools"

def convert_md_to_mdx(source_dir, dest_dir):
    """Convert all MD files in source_dir to MDX files in dest_dir."""
    source_path = Path(source_dir)
    dest_path = Path(dest_dir)
    
    # Ensure destination directory exists
    dest_path.mkdir(parents=True, exist_ok=True)
    
    converted_files = []
    
    # Process each .md file
    for md_file in source_path.glob('*.md'):
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
        with open(md_file, 'r', encoding='utf-8') as f:
            content = f.read()
        
        # Extract metadata
        title = extract_title(md_file.name)
        description = extract_description(content)
        category = determine_category(md_file.name, content)
        
        # Generate MDX content using template
        mdx_content = MDX_TEMPLATE.render(
            title=title,
            description=description,
            category=category,
            content=content
        )
        
        # Create category subfolder
        category_folder = category.lower().replace(' ', '-')  # "app-tools" or "system-tools"
        category_path = dest_path / category_folder
        category_path.mkdir(parents=True, exist_ok=True)
        
        # Write to destination as .mdx file in appropriate subfolder
        mdx_filename = md_file.stem + '.mdx'
        mdx_file = category_path / mdx_filename
        
        with open(mdx_file, 'w', encoding='utf-8') as f:
            f.write(mdx_content)
        
        converted_files.append(f"{category_folder}/{mdx_filename}")
        print(f"  → Created {category_folder}/{mdx_filename}")
    
    return converted_files

def main():
    """Main execution function."""
    # Define paths
    source_dir = "/Users/sarathmenon/Documents/startup/image_generation/mix/go_backend/internal/llm/tools/descriptions"
    dest_dir = "/Users/sarathmenon/Documents/startup/image_generation/mix/docs/content/docs/backend/tools"
    
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
    
    print(f"\nNext step: Update meta.json to include these files in the sidebar")

if __name__ == "__main__":
    main()