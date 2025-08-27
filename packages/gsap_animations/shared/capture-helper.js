/**
 * Shared GSAP Animation Capture Helper
 * 
 * Provides utilities for implementing window.__CAPTURE__ interface
 * consistently across all animations for video capture compatibility.
 */

/**
 * Creates a standardized window.__CAPTURE__ interface
 * @param {Object} config - Configuration object
 * @param {number|function} config.duration - Animation duration in seconds or function that returns duration
 * @param {function} config.initAnimation - Function to initialize/create the animation
 * @param {function|string} config.getTimeline - Function that returns timeline or timeline variable name 
 * @param {function} [config.onInit] - Optional callback after initialization
 * @param {function} [config.onSetFrame] - Optional callback when setting frame
 * @returns {Object} __CAPTURE__ interface object
 */
function createCaptureInterface(config) {
    return {
        get duration() {
            return typeof config.duration === 'function' ? config.duration() : config.duration;
        },
        
        timeline: null,
        
        init() {
            // Initialize the animation
            this.timeline = config.initAnimation();
            
            // Store reference for frame seeking
            if (this.timeline) {
                // Pause timeline for frame-by-frame control
                this.timeline.pause();
            }
            
            // Call optional callback
            if (config.onInit) {
                config.onInit(this.timeline);
            }
        },
        
        setFrame(frameIndex, fps) {
            if (!this.timeline) {
                console.warn('Timeline not initialized. Call init() first.');
                return;
            }
            
            // Calculate time position
            const time = frameIndex / fps;
            const progress = Math.min(time / this.duration, 1);
            
            // Seek to specific frame
            this.timeline.progress(progress);
            
            // Call optional callback
            if (config.onSetFrame) {
                config.onSetFrame(frameIndex, fps, time, progress, this.timeline);
            }
        },
        
        getTotalFrames(fps) {
            return Math.round(this.duration * fps);
        }
    };
}

/**
 * Helper for animations that use a global timeline variable
 * @param {Object} config - Configuration object
 * @param {number|function} config.duration - Animation duration
 * @param {function} config.createAnimation - Function that creates and returns timeline
 * @param {string} [config.timelineVar] - Global timeline variable name (optional)
 * @returns {Object} __CAPTURE__ interface
 */
function createTimelineCaptureInterface(config) {
    return createCaptureInterface({
        duration: config.duration,
        initAnimation: () => {
            const timeline = config.createAnimation();
            
            // If timeline variable name provided, store reference globally
            if (config.timelineVar && typeof window !== 'undefined') {
                window[config.timelineVar] = timeline;
            }
            
            return timeline;
        },
        onInit: config.onInit,
        onSetFrame: config.onSetFrame
    });
}

/**
 * Helper for animations that need special frame seeking logic
 * @param {Object} config - Configuration object  
 * @param {number|function} config.duration - Animation duration
 * @param {function} config.initAnimation - Function to initialize animation
 * @param {function} config.setFrameLogic - Custom frame seeking logic
 * @returns {Object} __CAPTURE__ interface
 */
function createCustomCaptureInterface(config) {
    const captureInterface = createCaptureInterface({
        duration: config.duration,
        initAnimation: config.initAnimation,
        onInit: config.onInit
    });
    
    // Override setFrame with custom logic
    captureInterface.setFrame = async function(frameIndex, fps) {
        if (!this.timeline) {
            console.warn('Timeline not initialized. Call init() first.');
            return;
        }
        
        const time = frameIndex / fps;
        const progress = Math.min(time / this.duration, 1);
        
        await config.setFrameLogic(frameIndex, fps, time, progress, this.timeline);
    };
    
    return captureInterface;
}

/**
 * Utility function to calculate total animation duration from multiple phases
 * @param {Array<number>} phases - Array of phase durations
 * @param {number} [repeatDelay=0] - Optional repeat delay
 * @param {number} [repeatCount=0] - Number of repeats (0 = no repeat, -1 = infinite)
 * @returns {number} Total duration
 */
function calculateTotalDuration(phases, repeatDelay = 0, repeatCount = 0) {
    const singleCycleDuration = phases.reduce((sum, duration) => sum + duration, 0);
    
    if (repeatCount === -1 || repeatCount === 0) {
        return singleCycleDuration;
    }
    
    return singleCycleDuration + (repeatCount * (singleCycleDuration + repeatDelay));
}

/**
 * Helper to ensure timeline is ready for capture (paused, at start position)
 * @param {Object} timeline - GSAP timeline
 */
function prepareTimelineForCapture(timeline) {
    if (timeline) {
        timeline.pause();
        timeline.progress(0);
        timeline.invalidate(); // Clear any cached values
    }
}

// Export for use in animations
if (typeof window !== 'undefined') {
    window.CaptureHelper = {
        createCaptureInterface,
        createTimelineCaptureInterface, 
        createCustomCaptureInterface,
        calculateTotalDuration,
        prepareTimelineForCapture
    };
}