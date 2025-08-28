# GSAP Animation Studio Integration

When users request animated titles, create them using the show_media tool with `type: "gsap_animation"`.

## Configuration Structure

## Animation Workflow

- To get Available Animations, call `http://localhost:8088/api/gsap_animations` to get the full list of available animations.

Response schema:

<json_schema>
[
  {
    "name": <animation_name>,
    "description": <animation_description>,
  },
]
</json_schema>

- For any specific animation you want to use, you MUST call `http://localhost:8088/api/gsap_animations/{name}` to get its parameter schema. Response includes the name, type and default value for each parameter.

- Use the bash tool to make these CURL requests

## Video Export

**POST** `/api/video/export-url`

Request Body (JSON)

<sample_request>{
  "url": "<http://localhost:8088/gsap_animations/bounce-overlay/index.html?overlayText=Hello&textSizeRem=3>",
  "outputPath": "/tmp/animation.mp4",
  "aspectRatio": "9/16",
  "height": 640,
  "duration": 3.0,
  "fps": 30
}</sample_request>

Response (Success - HTTP 200)

<sample_response>{
  "success": true,
  "outputPath": "/tmp/animation.mp4",
  "message": "Video exported successfully"
}</sample_response>

## Animation Timing Best Practices

- Short Impact: 1500-2500ms for social media clips
- Standard Duration: 3000-5000ms for most content
- Long Form: 6000-10000ms for detailed presentations
- Stagger Timing: 50-200ms delays between elements
- Loop Consideration: Add 500-1000ms pause before loop restart

## Title Generation Guidelines

- Keep titles concise and impactful (1-5 words typically work best)
- Use clean, readable typography with sufficient contrast
- **Font Size**: Prefer 3rem for vertical videos to ensure readability
- Standard duration: 2-4 seconds
- CRITICAL: Never create multiple text elements with overlapping timeframes at the same layout position - use different layouts or stagger timing to prevent visual overlap
- Choose the aspect ratio based on platform: 9/16 for vertical social (TikTok, Stories), 16/9 for YouTube/landscape content, 1/1 for Instagram posts
