When users request animated titles, create them using the media_showcase tool with `type: "remotion_title"`:

Title Generation Guidelines:

- Keep titles concise and impactful (1-5 words typically work best)
- Use clean, readable typography with sufficient contrast
- Default to professional, modern styling unless specified otherwise
- Standard duration: 3-5 seconds (90-150 frames at 30fps)
- Default format: "horizontal" (1920x1080) for high quality output
- CRITICAL: Never create multiple text elements with overlapping timeframes at the same layout position - use different layouts or stagger timing to prevent visual overlap

Element Types:

- `type: "text"`: Text elements with animations, strokes, and styling
- `type: "image"`: Background image that fills the frame (place first in elements array)
- `type: "video"`: Background video that plays during composition (place first in elements array)

File Path Format:

- Convert absolute file paths to: `http://localhost:8088/<relative_path_from_workdir>`
- Example: `http://localhost:8088/input/videos/egg.mp4`

Animation Types:

- `fadeIn`: Element fades in from transparent to opaque (30-45 frame duration)
- `fadeOut`: Element fades out from opaque to transparent (30-45 frame duration)  
- `slideIn`: Element slides in from the left (20-30 frame duration)
- `slideOut`: Element slides out to the right (20-30 frame duration)
- `typing`: Text appears character by character like a typewriter (60-90 frame duration)
- `tiktokEntrance`: Spring-based entrance with bounce and scale effect (30-60 frame duration)

Stroke Configuration:
Add visual impact with text strokes. Available options:

- `stroke: {"width": 2, "color": "#000000"}`: Standard thin outline
- `stroke: {"width": 20, "color": "#000000"}`: TikTok-style thick black outline
- `stroke: {"width": 20, "color": "#ffffff"}`: TikTok-style thick white outline  
- `stroke: {"width": 15, "color": "#ff1493"}`: TikTok-style neon pink outline
- `stroke: {"width": 15, "color": "#00ffff"}`: TikTok-style neon cyan outline

Word-Level Timing:
For karaoke-style highlighting, use `wordTimings` to specify when each word should be highlighted:

```json
"wordTimings": [
  {"word": "Hello ", "start": 0, "end": 30},
  {"word": "World", "start": 30, "end": 60}
]
```

Words are highlighted in bright green (#39E508) during their active timing window.

Social Media Optimization:
For TikTok and Instagram content:

- Use `tiktokEntrance` animation for viral-style entrances (40-50 frame duration)
- Apply TikTok stroke presets for platform-native styling (width: 15-20)
- Keep titles punchy and readable (2-4 words max)
- Use high contrast colors (white text with thick black stroke)
- Use large font sizes (90-100px) for maximum readability on mobile
- Use format: "vertical" (1080x1920) for TikTok and Instagram Stories
- Keep duration concise (90-120 frames / 3-4 seconds at 30fps)

<config_structure>
{
  "composition": {
    "durationInFrames": 120,
    "fps": 30,
    "format": "horizontal"
  },
  "elements": [
    {
      "type": "video",
      "content": "http://localhost:8088/input/videos/egg.mp4",
      "compositionStartFrame": 0,
      "compositionDuration": 120,
      "sourceStartFrame": 60
    },
    {
      "type": "text",
      "content": "Title Text",
      "compositionStartFrame": 0,
      "compositionDuration": 90,
      "layout": "top-center",
      "style": {"fontSize": 72, "color": "#ffffff"},
      "animation": {"type": "fadeIn", "duration": 30},
      "stroke": {"width": 2, "color": "#000000"}
    }
  ]
}
</json_config>

**Element Timing Properties:**

- `compositionStartFrame`: Frame in the composition timeline when this element appears
- `compositionDuration`: How many frames this element is visible in the composition
- `sourceStartFrame`: For video elements only - which frame in the source video to start playing from

**Example Video Timing:**

<sample_timing_config>
{
  "type": "video",
  "content": "<http://localhost:8088/input/videos/long-video.mp4>",
  "compositionStartFrame": 30,    // Shows at frame 30 of composition (1 sec in)
  "compositionDuration": 90,      // Visible for 90 frames (3 seconds)
  "sourceStartFrame": 180         // Starts from frame 180 of source (6 seconds into original)
}
</sample_timing_config>

Video Format Options:

The `format` property controls video dimensions and aspect ratio:

- `format: "horizontal"`: 1920x1080 - Standard landscape format for YouTube, presentations, and general use
- `format: "vertical"`: 1080x1920 - Portrait format optimized for TikTok, Instagram Stories, and mobile content

Layout Configurations controls where text appears on screen using the `layout` property:

- `layout: "top-center"`: Top center positioning (recommended default)
- `layout: "bottom-center"`: Bottom center positioning

Best Practices:

- Use top layout `layout: "top-center"` for main titles and headlines
- Use bottom layout `layout: "bottom-center"` for credits, watermarks, or call-to-action text
- Use white (#ffffff) text on transparent background for maximum versatility
- Stagger multiple text elements by 20-40 frames for smooth flow
- Keep animations subtle and professional unless specifically requested otherwise
- Use fontSize between 48-96 for optimal readability across devices

Background Guidelines:

- Use only one background element per composition (either image OR video, never both)
- Place background elements first in elements array (render behind text)
- Set background duration to match composition duration for full coverage
- Choose high-contrast backgrounds for optimal text readability

Platform-Specific Guidelines:

- Professional/Corporate: Use `fadeIn` or `slideIn` with minimal stroke (width: 2) or no stroke
- Social Media/TikTok: Use `tiktokEntrance` with thick strokes (width: 15-20) for maximum impact
- Karaoke/Lyric Videos: Implement `wordTimings` for synchronized highlighting
- Long-form Content: Use `typing` animation for dramatic text reveals

Stroke Usage Guidelines:

- No stroke: Clean, minimal look for professional content
- Thin stroke (2-4px): Subtle outline for better text readability
- Thick stroke (15-20px): Bold, social media style for maximum visibility
- Colored strokes: Use neon colors (#ff1493, #00ffff) for energetic, youthful content
