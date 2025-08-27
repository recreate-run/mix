# GSAP Animations

Standardized parametric GSAP animations with video capture support.

## Structure

```
animation-name/
├── index.html    # Animation implementation
└── schema.json   # Parameter definitions
```

## Schema Pattern

```json
{
  "name": "my-animation",
  "description": "Animation description", 
  "parameters": [
    {"name": "displayText", "type": "string", "default": "Hello"},
    {"name": "animationDurationSeconds", "type": "number", "default": 3},
    {"name": "primaryColor", "type": "string", "default": "#fff"},
    {"name": "autoloop", "type": "boolean", "default": false}
  ]
}
```

**Parameter Conventions:**

- Duration: `*DurationSeconds`, `*DelaySeconds`
- Size: `*SizePixels`, `*SizeRem`, `*SizeCSS`  
- Colors: `*Color` (hex format)
- CSS: `*CSS` suffix for complex values

## HTML Template

```html
<!DOCTYPE html>
<html>
<head>
    <link rel="stylesheet" href="../shared/base.css">
    <style>/* Your styles */</style>
</head>
<body>
    <!-- Your HTML -->
    
    <script src="https://cdn.jsdelivr.net/npm/gsap@3/dist/gsap.min.js"></script>
    <script src="../shared/capture-helper.js"></script>
    <script>
        // Implementation
    </script>
</body>
</html>
```

## Required Functions

```javascript
// 1. Parameter handling with URL support + type conversion
function getParams() { /* defaults + URLSearchParams + type conversion */ }

// 2. Apply parameters to DOM/styles
function applyParams(params) { /* update DOM based on params */ }

// 3. Create GSAP timeline
function createAnimation(params) { /* return gsap.timeline() */ }

// 4. Initialize on load
function initAnimation() { /* getParams() + applyParams() + createAnimation() */ }
window.addEventListener('load', initAnimation);
```

## Capture Interface

```javascript
window.__CAPTURE__ = window.CaptureHelper.createCustomCaptureInterface({
    duration: () => getParams().animationDurationSeconds,
    initAnimation: () => {
        const params = getParams();
        applyParams(params);
        return createAnimation(params);
    },
    setFrameLogic: async (frameIndex, fps, time, progress, timeline) => {
        timeline.progress(progress);
        // Add custom frame logic (particles, video sync, etc.)
    }
});
```

## Video Background Pattern

```javascript
setFrameLogic: async (frameIndex, fps, time, progress, timeline) => {
    timeline.progress(progress);
    
    const video = document.querySelector('video');
    video.currentTime = time;
    
    await new Promise(resolve => {
        const onSeeked = () => {
            video.removeEventListener('seeked', onSeeked);
            resolve();
        };
        video.addEventListener('seeked', onSeeked);
        setTimeout(resolve, 100); // Fallback
    });
}
```

## Testing

- **Preview**: Open `index.html?param=value`
- **Capture**: `node go_backend/scripts/capture-url.mjs http://localhost:3000/animation-name/`
