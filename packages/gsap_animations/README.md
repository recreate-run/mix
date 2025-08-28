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

## Video Background Pattern (Capture/Export)

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

## Video-Animation Synchronization (Live Playback)

**Core Principle**: Let GSAP's timeline control video timing, not parameter durations.

### ✅ Correct Pattern

```javascript
function createTimeline(params) {
    const tl = gsap.timeline({ repeat: -1, repeatDelay: 1 });
    
    // Start video when timeline begins
    tl.call(() => {
        video.currentTime = 0;
        video.play().catch(console.error);
    }, [], 0);
    
    // Add all your text/visual animations...
    tl.to(element, { opacity: 1, duration: 0.5 }, 0)
      .to(chars, { y: 0, stagger: 0.1 })
      .to(element, { scale: 1.2, duration: 0.3 });
    
    // Stop video when animation actually completes
    tl.call(() => {
        video.pause();
    });
    
    return tl;
}
```

### ❌ Avoid These Patterns

```javascript
// DON'T: Hard-coded parameter timing
tl.call(() => video.pause(), [], params.totalAnimationDurationSeconds);

// DON'T: Event listeners for duration control  
video.addEventListener('timeupdate', () => {
    if (video.currentTime >= params.duration) video.pause();
});
```

### Why This Works

- **Automatic duration calculation**: GSAP computes real animation duration including stagger effects
- **Perfect synchronization**: Video stops exactly when animation completes
- **No timing bugs**: Eliminates mismatches between parameter duration and actual animation duration
- **Capture compatibility**: Works seamlessly with frame-accurate video export

## Testing

- **Preview**: Open `index.html?param=value`
- **Capture**: `node go_backend/scripts/capture-url.mjs http://localhost:3000/animation-name/`
