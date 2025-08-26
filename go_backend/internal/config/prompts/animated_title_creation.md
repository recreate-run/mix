# GSAP Animation Studio Integration

When users request animated titles, create them using the media_showcase tool with `type: "gsap_animation"`.

## Configuration Structure

## Animation Workflow

- To get Available Animations, call `http://localhost:3000/api/animations` to get the full list of available animations.

Response schema:

<json_schema>
[
  {
    "name": <animation_name>,
    "description": <animation_description>,
  },
]
</json_schema>

- For any specific animation you want to use, you MUST call `http://localhost:3000/api/animations/{name}` to get its parameter schema. Response includes the name, type and default value for each parameter.

- Use the bash tool to make these CURL requests

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
