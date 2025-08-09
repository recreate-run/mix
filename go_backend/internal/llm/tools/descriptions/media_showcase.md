Use this tool to display media files prominently in the conversation interface. This shows images, videos, and audio with large previews in the conversation.

## When to Use This Tool

Use this tool for DISPLAYING/VIEWING media files visually:

1. Visual Display Requests:
   - "Show me the image", "Display this video", "Let me see the file"
   - "Preview the media", "View this content", "Present the file"
   - Showcasing completed creative outputs and deliverables

2. File Presentation (not content analysis):
   - When users want to SEE the actual media file
   - Presenting finished work or creative deliverables
   - Showing examples and reference materials for visual context

3. After Creative Tasks - Display media outputs from:
   - Video creation/editing (marketing videos, montages, edited clips)
   - Image generation/processing (posters, logos, processed images)
   - Audio production (soundtracks, voice-overs, audio effects)
   - Multi-media projects and complete creative deliverables

## When NOT to Use This Tool

Skip using this tool when:
1. Content Analysis Requests: "What's in this image?", "Describe this file", "Analyze this"
2. Information Extraction: "Explain what you see", "What does this show?"
3. Understanding Content: Any request to interpret or analyze media content
4. No media files: Task only involved text or code generation  
5. File doesn't exist: Media file paths are invalid or inaccessible
6. Non-media content: Documents, data files, or other non-visual/audio content

## Examples of When to Use Media Showcase

<example>
User: Show me /Users/project/assets/logo.png
Assistant: Let me display that image for you.

*Uses media_showcase tool with:*
- Path: /Users/project/assets/logo.png
- Type: image
- Title: Project Logo
- Description: Current logo file from assets folder
</example>

<example>
User: Create a 30-second marketing video for our product
Assistant: [After performing video creation steps, gathering assets, editing clips, adding music, and rendering the final output]

I've successfully created your marketing video. Let me showcase the final result.

*Uses media_showcase tool with:*
- Path: /output/video/product_marketing_20241201_143022.mp4
- Type: video
- Title: Product Marketing Video
- Description: 30-second promotional video with product highlights and call-to-action
</example>

<example>
User: I need to see the reference audio file before we start editing
Assistant: I'll display the reference audio file so you can review it.

*Uses media_showcase tool with:*
- Path: /Users/project/references/background_music.mp3
- Type: audio
- Title: Reference Background Music
- Description: Original audio file for editing reference
</example>

<example>
User: Create a simple animated title saying "Welcome"
Assistant: [After generating the title configuration]

I've created an animated title for you.

*Uses media_showcase tool with:*
- Path: "" (not required for remotion_title)
- Type: remotion_title
- Title: Welcome Title Animation
- Description: Animated welcome title with fade-in effect
- Config: {"composition": {"durationInFrames": 90, "fps": 30, "width": 1920, "height": 1080}, "elements": [{"type": "text", "content": "Welcome", "from": 0, "durationInFrames": 90, "position": {"x": 0, "y": 0}, "style": {"fontSize": 72, "color": "#ffffff"}, "animation": {"type": "fadeIn", "duration": 30}}]}
</example>

## Parameters

outputs (required): Array of media outputs to showcase
- path (required): Absolute file path to the media file (not required for remotion_title type)
- type (required): Media type - "image", "video", "audio", or "remotion_title"
- title (required): Display title for the media
- description (optional): Human-provided project context or metadata (NOT content analysis)
- config (optional): For remotion_title type, provide JSON configuration with composition settings and animated elements

**Description Field Notes:**
- Use for WHY you're showing the media, not WHAT'S IN it
- Examples: "Reference for redesign", "Generated as requested", "Option 1"  
- Never describe media content - that's not the purpose of this tool

## Tool Behavior

1. Validates all file paths - Ensures files exist and are accessible
2. Checks file extensions - Verifies extensions match the specified media type
3. Frontend Integration - Media outputs are displayed prominently with large previews

## Usage Notes

- Always use absolute paths - Relative paths will be rejected
- Include meaningful titles - Help users understand what they're viewing  
- Add descriptions for context - Especially useful for complex or reference materials
- Multiple outputs supported - Display multiple related media files at once
- Use for any media display - Not limited to creative outputs; great for previews and references

This tool transforms file paths into beautiful media displays in the conversation interface.