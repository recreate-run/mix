Display media files prominently in the conversation with large previews.

## When to Use This Tool

Use for displaying/viewing media files:

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

3. After Media Processing Tasks - Display media outputs from:
   - Video editing (trimming, effects, filters, color correction, edited clips)
   - Image processing (enhancement, filters, transformations, processed images)
   - Audio processing (mixing, effects, mastering, processed audio)
   - Video creation/editing (marketing videos, montages, edited clips)
   - Image generation/processing (posters, logos, processed images)
   - Audio production (soundtracks, voice-overs, audio effects)
   - Multi-media projects and complete deliverables
   - Exporting videos

4. Video Highlights - Display video segments with timestamps instead of creating separate files

5. MANDATORY animation workflow: When creating animations:
   - Show animation preview first
   - STOP and wait for explicit user approval/feedback
   - Never automatically proceed to directly export/render the output video, even if the users says "create a video" etc.

## When NOT to Use This Tool

Skip using this tool when:

1. Content Analysis Requests: "What's in this image?", "Describe this file", "Analyze this"
2. Information Extraction: "Explain what you see", "What does this show?"
3. File doesn't exist: Media file paths are invalid or inaccessible
4. Non-media content: Text documents or other non-visual/audio/data content

## Examples of When to Use Media Showcase

<example>
User: Show me http://localhost:8088/api/sessions/550e8400-e29b-41d4-a716-446655440000/files/logo.png
Assistant: Let me display that image for you.

*Uses show_media tool with:*

- Path: <http://localhost:8088/api/sessions/550e8400-e29b-41d4-a716-446655440000/files/logo.png>
- Type: image
- Title: Project Logo
- Description: Logo image from session storage
</example>

<example>
User: Show me this image: http://localhost:8088/api/sessions/6ba7b810-9dad-11d1-80b4-00c04fd430c8/files/banner.jpg
Assistant: I'll display that image from the URL for you.

*Uses show_media tool with:*

- Path: <http://localhost:8088/api/sessions/6ba7b810-9dad-11d1-80b4-00c04fd430c8/files/banner.jpg>
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
- Type: youtube
- Title: YouTube Reference Video
- Description: External video content for reference
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
User: Create a GSAP animation with bouncing text saying "AMAZING!"
Assistant: [After configuring the GSAP animation]

I've created a dynamic GSAP animation for you.

*Uses show_media tool with:*

- Type: gsap_animation
- Title: AMAZING Text Animation
- Description: TikTok-style bouncing text with bold styling
- Config: {
    "url": <iframe_url>,
    "animationType": <animation_name>
  }
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

- path: HTTP/HTTPS URL (required except for gsap_animation). For video/audio segments, this is the source media URL
- type: "image", "video", "audio", "gsap_animation", "youtube", "pdf", or "csv"
- title: Display title
- description: Project context (optional)
- config: Animation configuration JSON (required for gsap_animation)
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

- Always use HTTP/HTTPS URLs - Local of absolute file paths are not supported. Use `http://` (not https) for localhost URLs to avoid SSL certificate issues
- Use session asset server format: `http://localhost:8088/api/sessions/{session-id}/files/{filename}` to show local files in session storage
- URLs supported - HTTP/HTTPS URLs work for image, video, and audio types
- Include meaningful titles - Help users understand what they're viewing  
- Add descriptions for context - Especially useful for complex or reference materials
- Multiple outputs supported - Display multiple related media files at once
- Useit it to show previews and references
- Always use `type: "youtube"` for YouTube URLs (youtube.com, youtu.be). This enables proper YouTube iframe embedding with controls and full-screen support

This tool transforms file paths into beautiful media displays in the conversation interface.
