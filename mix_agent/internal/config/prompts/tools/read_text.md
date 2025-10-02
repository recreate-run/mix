Reads a text file from the local filesystem or from a URL. Assume this tool is able to read all local text files on the machine and download text content from URLs (http/https). If the User provides a path to a file or URL assume that path is valid. It is okay to read a file that does not exist; an error will be returned.

Usage notes:

- The file_path parameter must be an absolute path for local files, or a URL (http/https)
- For URLs, the content will be downloaded and processed as text (Content-Type must be text-based)
- By default, it reads up to 2000 lines starting from the beginning of the file
- You can optionally specify a line offset and limit (especially handy for long
files), but it's recommended to read the whole file by not providing these parameters
- Any lines longer than 2000 characters will be truncated
- Results are returned using cat -n format, with line numbers starting at 1
- This tool can read only text files. Use the ReadMedia tool if you want to read media files.
- You have the capability to call multiple tools in a single response. It is always
better to speculatively read multiple files as a batch that are potentially useful.
- If you read a file that exists but has empty contents you will receive a system
reminder warning in place of file contents.
