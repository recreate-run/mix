Analyzes multimedia files using AI to  generate descriptions, or create transcripts. Assume this tool is able to read all local media files on the machine,  but any URL is not supported. If the user provides a path to a file assume that path is valid. If the user provides a URL, first download the media file to the working directory before using the tool. It is okay to read a file that does not exist; an error will be returned.

Usage notes:

- Supports three media types: "image", "audio", "video" or "pdf"
- Can process single files or entire directories with optional recursive scanning
- Requires either file_path OR directory_path parameter (mutually exclusive)
- All file paths must be absolute paths or URL's for security. Relative paths are not allowed
- This tool can analyze youtube videos directly.
- For audio files: audio_mode parameter required ("transcript" or "description")
- For video files: video_mode parameter required ("description")
- For image files and pdf's: no additional mode parameter needed
- For PDF files: PDFs are automatically truncated to the first 10 pages when no page range is specified via the pdf_pages parameter. To analyze more pages, specify a page range (e.g., pdf_pages: "1-20").
- For video files: Videos are automatically truncated to the first 10 minutes when no time interval is specified via the video_interval parameter. To analyze longer videos, specify a time interval (e.g., video_interval: "00:00:00-00:20:00").
