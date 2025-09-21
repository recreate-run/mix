# GSAP Animation Studio Integration

When users request animated titles, create them using the show_media tool with `type: "gsap_animation"`.

## Animation Workflow

- To get Available Animations, call `http://localhost:8089/animations` to get the full list of available animations.

Response schema:

<json_schema>
[
  {
    "name": <animation_name>,
    "description": <animation_description>,
    "directory": <animation_directory>
  },
]
</json_schema>

- For any specific animation you want to use, you MUST call `http://localhost:8089/animations/{animation_name}/schema` to get its parameter schema. Response includes the name, type and default value for each parameter.

- Use the bash tool to make these CURL requests

## Video Export

**POST** `/export`

Request Body (JSON)

**Required Fields:**

- `url`: string - URL to capture/record

**Optional Fields:**

- `s3Url`: string - S3 presigned URL for upload
- `fps`: integer - Frame rate (default: 30, range: 1-120)
- `aspectRatio`: string - Format like "16/9", "4/3" (default: "9/16")
- `height`: integer - Video height in pixels (default: 640, range: 1-4096)
- `duration`: float - Recording duration in seconds (default: 3.0, range: 0.1-60)

<sample_request>{
  "url": "<http://localhost:8089/animations/bounce-overlay/preview?overlayText=Hello&textSizeRem=3>",
  "s3Url": "<https://presigned-s3-url.com/upload>" // Optional: for S3 upload
  "aspectRatio": "9/16",
  "height": 640,
  "duration": 3.0,
  "fps": 30
}</sample_request>

**Success Response (HTTP 200):**

<sample_response>{
  "success": true,
  "url": "<http://localhost:8089/storage/video_20240315_143022_abc123.mp4>",
  "s3Url": "<https://s3.amazonaws.com/bucket/video.mp4>", // Only if S3 upload requested
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
- Prefer 3rem title font size for "9/16" aspect ratio
- Standard duration: 2-4 seconds
- Never create multiple text elements with overlapping timeframes at the same layout position - use different layouts or stagger timing to prevent visual overlap
- Choose the aspect ratio based on platform: 9/16 for vertical social (TikTok, Stories), 16/9 for YouTube/landscape content, 1/1 for Instagram posts
