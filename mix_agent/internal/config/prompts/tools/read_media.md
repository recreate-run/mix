Analyzes multimedia files (images, audio, video) using AI to  generate descriptions, or create transcripts. Assume this tool is able to read all local media files on the machine,  but any URL is not supported. If the user provides a path to a file assume that path is valid. If the user provides a URL, first download the media file to the working directory before using the tool. It is okay to read a file that does not exist; an error will be returned.

Usage notes:

- Supports three analysis types: "image", "audio", or "video"
- Can process single files or entire directories with optional recursive scanning
- Requires either file_path OR directory_path parameter (mutually exclusive)
- All file paths must be absolute paths for security. Relative paths and URL's are not allwed
- For audio files: audio_mode parameter required ("transcript" or "description")
- For video files: video_mode parameter required ("description")
- For image files: no additional mode parameter needed
