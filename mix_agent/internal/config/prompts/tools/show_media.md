Display media files, markdown content, and code blocks prominently in the conversation with large previews.

## When to Use This Tool

Use for displaying/viewing media files and formatted content:

1. Visual Display Requests:
   - "Show me the image", "Display this video", "Let me see the file"
   - "Preview the media", "View this content", "Present the file"
   - Showcasing completed media outputs and deliverables

2. File Presentation (not content analysis):
   - When users want to SEE the actual media file
   - Presenting finished work or media deliverables
   - Showing examples and reference materials for visual context
   - Displaying PDF documents, reports, and documentation
   - Showcasing CSV data files and spreadsheets

3. Formatted Content Display:
   - Display markdown-formatted documentation, guides, or explanations
   - Show syntax-highlighted code blocks with copy functionality
   - Present structured information with proper formatting

4. After Media Processing Tasks - Display media outputs from:
   - Video editing (trimming, effects, filters, color correction, edited clips)
   - Image processing (enhancement, filters, transformations, processed images)
   - Audio processing (mixing, effects, mastering, processed audio)
   - Video creation/editing (marketing videos, montages, edited clips)
   - Image generation/processing (posters, logos, processed images)
   - Audio production (soundtracks, voice-overs, audio effects)
   - Multi-media projects and complete deliverables
   - Exporting videos

5. Video Highlights - Display video segments with timestamps instead of creating separate files

## When NOT to Use This Tool

Skip using this tool when:

1. Content Analysis Requests: "What's in this image?", "Describe this file", "Analyze this"
2. Information Extraction: "Explain what you see", "What does this show?"
3. File doesn't exist: Media file paths are invalid or inaccessible
4. Non-media content: Text documents or other non-visual/audio/data content

## Examples of When to Use Media Showcase

<example>
User: Show me $<server_url>/api/sessions/550e8400-e29b-41d4-a716-446655440000/files/logo.png
Assistant: Let me display that image for you.

*Uses show_media tool with:*

- Path: <$<server_url>/api/sessions/550e8400-e29b-41d4-a716-446655440000/files/logo.png>
- Type: image
- Title: Project Logo
- Description: Logo image from session storage
</example>

<example>
User: Show me this image: $<server_url>/api/sessions/6ba7b810-9dad-11d1-80b4-00c04fd430c8/files/banner.jpg
Assistant: I'll display that image from the URL for you.

*Uses show_media tool with:*

- Path: <$<server_url>/api/sessions/6ba7b810-9dad-11d1-80b4-00c04fd430c8/files/banner.jpg>
- Type: image
- Title: Banner Image
- Description: Banner image from session storage
</example>

<example>
User: Create a 30-second marketing video for our product
Assistant: [After performing video creation steps, gathering assets, editing clips, adding music, and rendering the final output]

I've successfully created your marketing video. Let me showcase the final result.

*Uses show_media tool with:*

- Path: <https://example.com/outputs/product_marketing_20241201_143022.mp4>
- Type: video
- Title: Product Marketing Video
- Description: 30-second promotional video with product highlights and call-to-action
</example>

<example>
User: I need to see the reference audio file before we start editing
Assistant: I'll display the reference audio file so you can review it.

*Uses show_media tool with:*

- Path: <https://example.com/references/background_music.mp3>
- Type: audio
- Title: Reference Background Music
- Description: Original audio file for editing reference
</example>

<example>
User: Display this video from YouTube: https://www.youtube.com/watch?v=example123
Assistant: I'll show you that YouTube video as an embedded player.

*Uses show_media tool with:*

- Path: <https://www.youtube.com/watch?v=example123>
- Type: video
- Title: YouTube Reference Video
- Description: External video content for reference
</example>

<example>
User: Show me the section from 1:30 to 2:00 of this YouTube video: https://www.youtube.com/watch?v=example456
Assistant: I'll show you that specific 30-second segment from the YouTube video.

*Uses show_media tool with:*

- Path: <https://www.youtube.com/watch?v=example456>
- Type: video
- Title: YouTube Video Highlight
- Description: Tutorial section showing key concept
- StartTime: 90
- Duration: 30
</example>

<example>
User: Show this audio track: https://soundcloud.com/artist/track
Assistant: I'll display that audio track from the URL.

*Uses show_media tool with:*

- Path: <https://soundcloud.com/artist/track>
- Type: audio
- Title: SoundCloud Track
- Description: Remote audio content from SoundCloud
</example>

<example>
User: Display this project documentation: https://example.com/docs/project-spec.pdf
Assistant: I'll show you that PDF document for review.

*Uses show_media tool with:*

- Path: <https://example.com/docs/project-spec.pdf>
- Type: pdf
- Title: Project Specification
- Description: Technical documentation for project requirements
</example>

<example>
User: Show me the sales data CSV file: https://example.com/data/sales-report.csv
Assistant: I'll display that CSV file for you to review.

*Uses show_media tool with:*

- Path: <https://example.com/data/sales-report.csv>
- Type: csv
- Title: Sales Report Data
- Description: Monthly sales figures and analytics
</example>

<example>
User: Show me a markdown-formatted guide for using this API
Assistant: I'll create and display a formatted API guide for you.

*Uses show_media tool with:*

- Type: markdown
- Title: API Usage Guide
- Description: Comprehensive guide for API integration
- Path: (markdown content string with headings, code examples, lists, etc.)
</example>

<example>
User: Display the authentication function I just wrote
Assistant: Here's the authentication function with syntax highlighting.

*Uses show_media tool with:*

- Type: code
- Title: Authentication Function
- Description: User authentication with JWT tokens
- Path: (code content string)
- Config: { "language": "typescript" }
</example>

<example>
User: Give me three 5-second highlights from this demo video
Assistant: I've identified key segments from your demo video.

*Uses show_media tool with:*

- Type: video, Title: "Product Overview", Path: <https://example.com/videos/demo.mp4>, StartTime: 15, Duration: 5
- Type: video, Title: "UI Walkthrough", Path: <https://example.com/videos/demo.mp4>, StartTime: 85, Duration: 5
- Type: video, Title: "Results", Path: <https://example.com/videos/demo.mp4>, StartTime: 130, Duration: 5
</example>

## Parameters

outputs (required): Array of media outputs to showcase

- path: HTTP/HTTPS URL (for image/video/audio/pdf/csv) OR content string (for markdown/code)
- type: "image", "video", "audio", "pdf", "csv", "markdown", or "code"
- title: Display title
- description: Project context (optional)
- config: Configuration object (optional for code type: { "language": "typescript" })
- startTime: Start time in seconds for video/audio segments (optional)
- duration: Duration in seconds for video/audio segments (optional)

**Description Field Notes:**

- Use for WHY you're showing the media, not WHAT'S IN it
- Examples: "Reference for redesign", "Generated as requested", "Option 1"  
- Never describe media content - that's not the purpose of this tool

## Tool Behavior

1. Validates all URLs - Ensures URLs are valid HTTP/HTTPS format
2. Groups video highlights from same source into unified playback with timestamp navigation
3. Frontend Integration - Media outputs are displayed prominently with large previews

## Usage Notes

- **Media files**: Always use HTTP/HTTPS URLs - Local or absolute file paths are not supported
- Use session asset server format: `$<server_url>/api/sessions/{session-id}/files/{filename}` to show local files in session storage
- URLs supported - HTTP/HTTPS URLs work for image, video, audio, pdf, and csv types
- **Markdown/Code**: Pass content directly in the path field as a string
- **Code syntax highlighting**: Specify language in config (e.g., `{"language": "typescript"}`)
- Include meaningful titles - Help users understand what they're viewing
- Add descriptions for context - Especially useful for complex or reference materials
- Multiple outputs supported - Display multiple related media/content items at once
- Use this tool to show previews, references, and formatted content
- Always convert Excel files (.xlsx, .xls) to CSV format before displaying with this tool

This tool transforms media URLs and formatted content into beautiful displays in the conversation interface.
