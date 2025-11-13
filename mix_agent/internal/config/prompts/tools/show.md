Display content to users: media files, formatted text, and status updates.

## When to Use This Tool

Use for presenting content to users:

1. Display Requests:
   - "Show me the image", "Display this video", "Present the results"
   - "Preview the content", "View this file"
   - Showcasing completed outputs and deliverables

2. Content Presentation (not analysis):
   - When users want to SEE or REVIEW content
   - Presenting finished work and deliverables
   - Showing examples and reference materials
   - Displaying documents (PDF), data (CSV), or reports

3. Formatted Text:
   - Markdown-formatted documentation, guides, or explanations
   - Structured information with proper formatting

4. After Processing Tasks - Display outputs from:
   - Video editing (trimming, effects, color correction)
   - Image processing (enhancement, filters, transformations)
   - Audio processing (mixing, effects, mastering)
   - Content generation (videos, images, audio)
   - Multi-format projects and deliverables

5. Video Segments - Display video highlights with timestamps instead of creating separate files

6. Status Updates - Present progress, completion states, or operation results

## When NOT to Use This Tool

Skip this tool for:

1. Analysis: "What's in this image?", "Describe this file", "Analyze this"
2. Extraction: "Explain what you see", "What does this show?"
3. Invalid paths: File paths are inaccessible or don't exist

## Examples

<example>
User: Show me $<server_url>/api/sessions/550e8400-e29b-41d4-a716-446655440000/files/logo.png
Assistant: Let me display that image for you.

*Uses show tool with:*

- Path: <$<server_url>/api/sessions/550e8400-e29b-41d4-a716-446655440000/files/logo.png>
- Type: image
- Title: Project Logo
- Description: Logo image from session storage
</example>

<example>
User: Show me this image: $<server_url>/api/sessions/6ba7b810-9dad-11d1-80b4-00c04fd430c8/files/banner.jpg
Assistant: I'll display that image from the URL for you.

*Uses show tool with:*

- Path: <$<server_url>/api/sessions/6ba7b810-9dad-11d1-80b4-00c04fd430c8/files/banner.jpg>
- Type: image
- Title: Banner Image
- Description: Banner image from session storage
</example>

<example>
User: Create a 30-second marketing video for our product
Assistant: [After performing video creation steps, gathering assets, editing clips, adding music, and rendering the final output]

I've successfully created your marketing video. Let me showcase the final result.

*Uses show tool with:*

- Path: <https://example.com/outputs/product_marketing_20241201_143022.mp4>
- Type: video
- Title: Product Marketing Video
- Description: 30-second promotional video with product highlights and call-to-action
</example>

<example>
User: I need to see the reference audio file before we start editing
Assistant: I'll display the reference audio file so you can review it.

*Uses show tool with:*

- Path: <https://example.com/references/background_music.mp3>
- Type: audio
- Title: Reference Background Music
- Description: Original audio file for editing reference
</example>

<example>
User: Display this video from YouTube: https://www.youtube.com/watch?v=example123
Assistant: I'll show you that YouTube video as an embedded player.

*Uses show tool with:*

- Path: <https://www.youtube.com/watch?v=example123>
- Type: video
- Title: YouTube Reference Video
- Description: External video content for reference
</example>

<example>
User: Show me the section from 1:30 to 2:00 of this YouTube video: https://www.youtube.com/watch?v=example456
Assistant: I'll show you that specific 30-second segment from the YouTube video.

*Uses show tool with:*

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

*Uses show tool with:*

- Path: <https://soundcloud.com/artist/track>
- Type: audio
- Title: SoundCloud Track
- Description: Remote audio content from SoundCloud
</example>

<example>
User: Display this project documentation: https://example.com/docs/project-spec.pdf
Assistant: I'll show you that PDF document for review.

*Uses show tool with:*

- Path: <https://example.com/docs/project-spec.pdf>
- Type: pdf
- Title: Project Specification
- Description: Technical documentation for project requirements
</example>

<example>
User: Show me the sales data CSV file: https://example.com/data/sales-report.csv
Assistant: I'll display that CSV file for you to review.

*Uses show tool with:*

- Path: <https://example.com/data/sales-report.csv>
- Type: csv
- Title: Sales Report Data
- Description: Monthly sales figures and analytics
</example>

<example>
User: Show me a markdown-formatted guide for using this API
Assistant: I'll create and display a formatted API guide for you.

*Uses show tool with:*

- Type: markdown
- Title: API Usage Guide
- Data: (markdown content string with headings, examples, lists, etc.)
</example>

<example>
User: Show me the user profile data as JSON. Schema: {"type": "object", "properties": {"id": {"type": "number"}, "name": {"type": "string"}, "email": {"type": "string"}, "roles": {"type": "array", "items": {"type": "string"}}}, "required": ["id", "name", "email"]}
Assistant: I'll display the user profile data in a structured format according to your schema.

*Uses show tool with:*

- Type: json
- Title: User Profile Data
- Data: {"id": 123, "name": "John Doe", "email": "john@example.com", "roles": ["user", "admin"]}
</example>

<example>
User: Show me the processing status
Assistant: I'll display the current status.

*Uses show tool with:*

- Type: status
- Title: Processing Status
- Data: Processing completed successfully. 150 files processed, 0 errors.
</example>

<example>
User: Process all images in the uploads folder
Assistant: [After completing the batch processing]

I've finished processing all images.

*Uses show tool with:*

- Type: status
- Description: Successfully processed 45 images: 23 enhanced, 12 resized, 10 converted to WebP
</example>

<example>
User: Give me three 5-second highlights from this demo video
Assistant: I've identified key segments from your demo video.

*Uses show tool with:*

- Type: video, Title: "Product Overview", Path: <https://example.com/videos/demo.mp4>, StartTime: 15, Duration: 5
- Type: video, Title: "UI Walkthrough", Path: <https://example.com/videos/demo.mp4>, StartTime: 85, Duration: 5
- Type: video, Title: "Results", Path: <https://example.com/videos/demo.mp4>, StartTime: 130, Duration: 5
</example>

## Parameters

outputs (required): Array of content to display

- type: "image", "video", "audio", "pdf", "csv", "markdown", "json", or "status"
- path: HTTP/HTTPS URL (for image/video/audio/pdf/csv). Not used for markdown, json, or status.
- data: Inline content string (for markdown/json/status types). For json, this should be a JSON string. For status, this is the status message.
- title: Display title for the content.
- startTime: Start time in seconds for video/audio segments (optional)
- duration: Duration in seconds for video/audio segments (optional)

Status Type:
- Uses data field for the status message
- Requires title and data fields
- Used for displaying status updates or notifications

JSON Type:
- Requires data field with JSON string
- Used for displaying structured data

## Tool Behavior

1. Validates all URLs - Ensures valid HTTP/HTTPS format
2. Groups video segments from same source into unified playback with timestamp navigation
3. Displays content prominently with large previews

## Usage Notes

- Files: Use HTTP/HTTPS URLs only - Local paths are not supported
- Session files: Use `$<server_url>/api/sessions/{session-id}/files/{filename}` format for session storage
- Inline content: Pass markdown/json/status content directly in data field as string
- Status updates: Use data field for the status message
- JSON data: Pass JSON string in data field
- Include meaningful titles - Help users understand what they're viewing
- Multiple outputs supported - Display related items together
- Convert Excel files (.xlsx, .xls) to CSV before displaying

This tool presents content with prominent, beautiful displays to users.
