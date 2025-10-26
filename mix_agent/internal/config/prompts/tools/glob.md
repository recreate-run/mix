Fast file pattern matching tool that works with any codebase size, finding files by name and pattern.

WHEN TO USE THIS TOOL:
- Use when you need to find files by name patterns or extensions
- Great for finding specific file types across a directory structure
- Useful for discovering files that match certain naming conventions
- When doing open-ended searches requiring multiple rounds, use the Task tool with Explore agent instead

HOW TO USE:
- Provide a glob pattern to match against file paths
- Optionally specify a starting directory with 'path' (defaults to current working directory)
- Results are returned sorted by modification time (most recently modified first)

PARAMETERS:
- pattern (required): The glob pattern to match files against
- path (optional): The directory to search in (defaults to current working directory)
  - IMPORTANT: Omit this field to use the default directory
  - DO NOT enter "undefined" or "null" - simply omit it for default behavior
  - Must be a valid directory path if provided

GLOB PATTERN SYNTAX:
- '*' matches any sequence of non-separator characters
- '**' matches any sequence of characters, including path separators (for recursive search)
- '?' matches any single non-separator character
- '[...]' matches any character in the brackets
- '[!...]' matches any character not in the brackets
- '{a,b}' matches either 'a' or 'b'

COMMON PATTERN EXAMPLES:
- '*.js' - Find all JavaScript files in the current directory
- '**/*.js' - Find all JavaScript files in any subdirectory (recursive)
- 'src/**/*.{ts,tsx}' - Find all TypeScript files in the src directory
- '*.{html,css,js}' - Find all HTML, CSS, and JS files
- '**/test_*.py' - Find all Python test files recursively
- 'components/**/*.tsx' - Find all TSX files in components directory

TIPS:
- Always better to speculatively perform multiple searches in parallel if potentially useful
- Combine with Grep tool for powerful searches: first find files with Glob, then search their contents with Grep
- When doing iterative exploration requiring multiple search rounds, use the Task tool with Explore agent instead
- Use this tool for file searches, NOT the 'find' or 'ls' bash commands
- Returns matching file paths sorted by modification time
- Works efficiently with any codebase size
