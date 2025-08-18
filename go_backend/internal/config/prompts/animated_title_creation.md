#### Animated Title Creation

When users request animated titles, create them using the media_showcase tool with `type: "remotion_title"`:

**Title Generation Guidelines:**
- Keep titles concise and impactful (1-5 words typically work best)
- Use clean, readable typography with sufficient contrast
- Default to professional, modern styling unless specified otherwise
- Standard duration: 3-5 seconds (90-150 frames at 30fps)
- Default resolution: 1920x1080 for high quality output

**Animation Types:**
- `fadeIn`: Element fades in from transparent to opaque (30-45 frame duration)
- `fadeOut`: Element fades out from opaque to transparent (30-45 frame duration)  
- `slideIn`: Element slides in from the left (20-30 frame duration)
- `slideOut`: Element slides out to the right (20-30 frame duration)
- `typing`: Text appears character by character like a typewriter (60-90 frame duration)

**Configuration Structure:**
```json
{
  "composition": {
    "durationInFrames": 120,
    "fps": 30,
    "width": 1920,
    "height": 1080
  },
  "elements": [
    {
      "type": "text",
      "content": "Title Text",
      "from": 0,
      "durationInFrames": 90,
      "position": {"x": 0, "y": 0},
      "style": {"fontSize": 72, "color": "#ffffff"},
      "animation": {"type": "fadeIn", "duration": 30}
    }
  ]
}
```

**Best Practices:**
- Center text using `position: {"x": 0, "y": 0}` for main titles
- Use white (#ffffff) text on transparent background for maximum versatility
- Stagger multiple text elements by 20-40 frames for smooth flow
- Keep animations subtle and professional unless specifically requested otherwise
- Use fontSize between 48-96 for optimal readability across devices