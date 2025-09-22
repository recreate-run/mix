Analyzes multimedia files (images, audio, video) using Gemini AI to extract insights,
generate descriptions, or create transcripts based on custom prompts.

Usage notes:

- Supports three analysis types: "image", "audio", or "video"
- Can process single files or entire directories with optional recursive scanning
- Requires either file_path OR directory_path parameter (mutually exclusive)
- All file paths must be absolute paths for security
- Requests user permission before accessing each file

Supported file formats:
- Images: jpg, jpeg, png, gif, webp, bmp
- Audio: mp3, wav, m4a, aac, ogg, flac
- Video: mp4, avi, mov, wmv, flv, webm, mkv

Analysis modes:
- For audio files: audio_mode parameter required ("transcript" or "description")
- For video files: video_mode parameter required ("description")
- For image files: no additional mode parameter needed

Response configuration:
- word_count parameter controls analysis length (50-1000 words, default 200)
- Returns structured JSON with individual file results and summary
- Each result includes file path, analysis type, content, and any errors

Processing options:
- Single file: specify file_path parameter
- Directory batch: specify directory_path parameter
- Use recursive parameter to process subdirectories when using directory_path
- Files not matching the specified analysis_type are automatically skipped

The tool uses Gemini 2.5 Pro model for multimodal analysis and requires a valid
Gemini API key to be configured in the system.