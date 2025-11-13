Analyzes multimedia files using AI. If the user provides a path to a file assume that path is valid. It is okay to read a file that does not exist; an error will be returned. Supports four media types: "image", "audio", "video" or "pdf"

Usage notes:

- NEVER combine transcription and content analysis in the same prompt. Request either a transcript OR analysis separately - make separate tool calls if both are needed.
- All file paths must be absolute paths or URL's for security. Relative paths are not allowed
- To analyze image files:
    1. For basic descriptions (e.g., "Caption this image" or "Describe what you see in this image")
    2. For visual question answering (e.g., "What color is the car?" or "How many people are in this photo?")
    3. For text extraction (e.g., "Extract and transcribe all text visible in this image")
- To analyze audio files:
    1. To get a transcript, request it in the prompt (e.g., "Generate a transcript of the speech").
    2. Input prompts can reference specific audio segments using MM:SS timestamps (e.g., "Provide a transcript between 02:30 and 03:29").
- To analyze video files:
    1. For content extraction (e.g., "Summarize this video in 3-5 sentences" or "Extract key points and create a bulleted outline of topics covered")
    2. Input prompts can reference specific video timestamps using MM:SS format (e.g., "What happens at 01:30?" or "Compare the scenes at 00:05 and 00:10").
    3. For transcription (e.g., "Transcribe the audio from this video, giving timestamps for salient events. Also, provide visual descriptions.")
    4. Videos are automatically truncated to the first 10 minutes when no time interval is specified via the video_interval parameter. To analyze longer videos, specify a time interval (e.g., video_interval: "00:00:00-00:20:00").
    5. This tool can analyze youtube videos directly.
- To analyze PDF files
    1. For summarization and extraction (e.g., "Summarize the key findings from this research paper" or "Extract all methodology details")
    2. For document question answering (e.g., "What conclusions are drawn in section 3?" or "List all figures and their captions")
    3. For structured information extraction (e.g., "Extract all tables into JSON format" or "Create a bulleted outline of the main topics")
    4. PDFs are automatically truncated to the first 10 pages when no page range is specified via the pdf_pages parameter. To analyze more pages, specify a page range (e.g., pdf_pages: "1-20").
