# Multimodal Analyzer tool

AI-powered media analysis tool (CLI interface) using multiple LLM providers through LiteLLM. Analyze images, audio, and video files with customizable prompts and output formats.

## Features
- **Image, Audio & Video Analysis**: Single files or batch process entire directories
- **Hybrid File Input**: Specify files by directory path OR explicit file lists from multiple locations
- **Automatic Image Preprocessing**: Images > 500KB are automatically converted to JPEG for optimal processing
- **Concurrent Processing**: Configurable concurrency with progress tracking
- **Multiple Output Formats**: JSON, Markdown, and Text export
- **Custom Prompts**: Flexible analysis with custom or predefined prompts

## Usage notes
- You MUST ALWAYS use Multimodal Analyzer to analyze media files, NEVER read images directly
- ALWAYS Use with the `bash` tool - this is a standalone CLI command
- ALWAYS use batch processing for analyzing multiple files

## When to Use This Tool

Use this tool for ANALYZING/UNDERSTANDING media content:

1. Content Analysis Requests:
   - "What's in this image?", "Describe this media", "Analyze this file"
   - "Explain what you see", "What does this show?"

2. Information Extraction:
   - Transcribing audio, describing video content
   - Understanding visual elements, text recognition
   - Any AI-powered content interpretation

## Trigger Phrases for Analysis
- "What's in...", "Describe...", "Analyze...", "Explain..."
- "What does this show?", "What can you see?"
- "Extract from...", "Understand...", "Interpret..."

## When NOT to Use This Tool
- When users just want to VIEW the media, not analyze its content
- Simple Display Requests: "Show me the file", "Display this image", "Let me see .."

## Hybrid File Input Support

The Multimodal Analyzer CLI supports two flexible input modes:

### Directory Path Mode (`--path`)
Use `--path` to analyze files from directories or single files:

```bash
# Single file
multimodal-analyzer --type image  --path photo.jpg

# Directory (all supported files)
multimodal-analyzer --type image  --path ./photos/

# Recursive directory scan
multimodal-analyzer --type image  --path ./dataset/ --recursive
```

### Explicit File List Mode (`--files`)
Use `--files` to specify exact files from multiple locations:

```bash
# Multiple files from different directories
multimodal-analyzer --type image  \
  --files /home/user/photo1.jpg \
  --files /work/project/chart.png \
  --files ./local/screenshot.jpg

# Audio files from various locations
multimodal-analyzer --type audio  \
  --files recording1.mp3 \
  --files /meetings/call.wav \
  --audio-mode transcript
```

### When to Use Each Mode

- **Use `--path`** for processing all files in a directory or subdirectories
- **Use `--files`** for selective processing of specific files from multiple locations
- **Cannot use both** `--path` and `--files` simultaneously (mutually exclusive)

## Image Analysis Usage

### Basic Image Commands

```bash
# Analyze single image
multimodal-analyzer --type image  --path photo.jpg

# Batch process directory
multimodal-analyzer --type image --model azure/gpt-4.1-mini --path ./photos/ --output markdown

# Development installation (prefix with uv run)
uv run multimodal-analyzer --type image  --path photo.jpg
```

### Advanced Image Analysis

```bash
# Custom prompt with word count
multimodal-analyzer --type image --model claude-3-sonnet-20240229 --path chart.jpg \
  --prompt "Analyze this chart focusing on data insights" --word-count 300

# Recursive batch processing
multimodal-analyzer --type image --model gpt-4o-mini --path ./dataset/ \
  --recursive --concurrency 5 --output json --output-file results.json

# Analyze specific images from multiple directories
multimodal-analyzer --type image --model gpt-4o-mini \
  --files ./screenshots/chart1.png \
  --files ./photos/diagram.jpg \
  --files /tmp/analysis_image.png \
  --prompt "Compare these visuals" --word-count 200
```

## Audio Analysis Usage

### Basic Audio Commands

```bash
# Transcribe audio
multimodal-analyzer --type audio --model whisper-1 --path audio.mp3 --audio-mode transcript

# Analyze audio content
multimodal-analyzer --type audio --model gpt-4o-mini --path podcast.wav --audio-mode description
```

### Advanced Audio Processing

```bash
# Batch transcription
multimodal-analyzer --type audio --model whisper-1 --path ./audio/ \
  --audio-mode transcript --output text --output-file transcripts.txt

# Content analysis with custom prompts
multimodal-analyzer --type audio --model gpt-4o-mini --path podcast.wav \
  --audio-mode description --prompt "Summarize key insights" --word-count 200

# Transcribe specific audio files from different locations
multimodal-analyzer --type audio --model whisper-1 \
  --files ./meetings/standup.mp3 \
  --files ./interviews/candidate1.wav \
  --files /recordings/conference_call.m4a \
  --audio-mode transcript --output markdown --output-file transcripts.md
```

## Video Analysis Usage

**Note**: Video analysis is currently restricted to Gemini models only due to native multimodal video support.

### Basic Video Commands

```bash
# Analyze video content
multimodal-analyzer --type video  --path video.mp4 --video-mode description
```

### Advanced Video Analysis

```bash
# Single video analysis
multimodal-analyzer --type video  --path presentation.mp4 \
  --video-mode description --word-count 150

# Batch video processing with custom prompts
multimodal-analyzer --type video  --path ./videos/ \
  --video-mode description --prompt "Describe the visual content and any audio" \
  --recursive --output markdown --output-file video_analysis.md

# Video analysis with detailed output
multimodal-analyzer --type video  --path tutorial.mp4 \
  --video-mode description --verbose --word-count 200

# Analyze specific videos from multiple projects
multimodal-analyzer --type video  \
  --files ./project1/demo.mp4 \
  --files ./project2/presentation.avi \
  --files /shared/training_video.mov \
  --video-mode description --prompt "Focus on key features demonstrated" \
  --word-count 300 --output json --output-file video_summaries.json
```

## Output Schema

### JSON Output Format (Batch Mode)

Results are returned as an array of objects, one per analyzed file:

### Error Handling
Failed analyses include error details