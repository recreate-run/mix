A powerful search tool built on ripgrep for finding text patterns in files.

WHEN TO USE THIS TOOL:
- Use when you need to find files containing specific text or patterns
- Great for searching code bases for function names, variable declarations, or error messages
- Useful for finding all files that use a particular API or pattern
- When doing open-ended searches requiring multiple rounds, use the Task tool with Explore agent instead

HOW TO USE:
- Provide a regex pattern to search for within file contents
- Optionally specify a starting directory with 'path' (defaults to current working directory)
- Filter files with 'glob' parameter (e.g., "*.js", "**/*.tsx") or 'type' parameter (e.g., "js", "py", "rust")
- Choose output mode: "content" (shows matching lines), "files_with_matches" (shows only file paths, default), "count" (shows match counts)
- Use context parameters (-A/-B/-C) with output_mode="content" to show lines before/after matches
- Enable multiline mode for patterns that span multiple lines

PARAMETERS:
- pattern (required): The regex pattern to search for
- path (optional): File or directory to search in (defaults to current working directory)
- output_mode (optional): "content", "files_with_matches" (default), or "count"
- glob (optional): Glob pattern to filter files (e.g., "*.js", "*.{ts,tsx}")
- type (optional): File type to search (e.g., "js", "py", "rust", "go") - more efficient than glob for standard types
- -i (optional): Case insensitive search
- -n (optional): Show line numbers (requires output_mode="content")
- -A (optional): Number of lines to show after each match (requires output_mode="content")
- -B (optional): Number of lines to show before each match (requires output_mode="content")
- -C (optional): Number of lines to show before and after each match (requires output_mode="content")
- multiline (optional): Enable multiline mode where . matches newlines and patterns can span lines (default: false)
- head_limit (optional): Limit output to first N lines/entries across all output modes

REGEX PATTERN SYNTAX:
- Uses ripgrep syntax (not standard grep)
- 'function' searches for the literal text "function"
- 'log.*Error' finds text starting with "log" and containing "Error"
- 'import\s+.*\s+from' finds import statements in JavaScript/TypeScript
- Literal braces need escaping: use 'interface\{\}' to find 'interface{}' in Go code
- By default patterns match within single lines only - use multiline=true for cross-line patterns

COMMON TYPE EXAMPLES:
- 'js' - JavaScript files
- 'ts' - TypeScript files
- 'py' - Python files
- 'rust' - Rust files
- 'go' - Go files
- 'java' - Java files

COMMON GLOB PATTERN EXAMPLES:
- '*.js' - Only search JavaScript files
- '*.{ts,tsx}' - Only search TypeScript files
- '**/*.go' - Search all Go files recursively

OUTPUT MODES:
- files_with_matches: Returns only file paths (default, fastest)
- content: Returns matching lines with optional context and line numbers
- count: Returns count of matches per file

TIPS:
- Always use the Grep tool, NEVER invoke 'grep' or 'rg' as a Bash command
- For faster searches, use 'type' parameter instead of 'glob' when searching standard file types
- Use output_mode="files_with_matches" first to find relevant files, then search with output_mode="content" for details
- When doing iterative exploration requiring multiple search rounds, use the Task tool instead
- Use multiline mode for patterns like 'struct \{[\s\S]*?field' that need to match across lines
- The tool has been optimized for correct permissions and access
