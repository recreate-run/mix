// URL-based Animation Capture with Playwright for Go Backend
// Usage: node capture-url.mjs <URL> --fps=30 --width=360 --height=640 --duration=3 --output=video.mp4
import { chromium } from "playwright";
import { spawn } from "child_process";
import { parseArgs } from "util";
import { mkdir, rm, writeFile } from "fs/promises";
import { existsSync } from "fs";
import path from "path";

// Parse command line arguments
const { values: args, positionals } = parseArgs({
  options: {
    fps: { type: "string", default: "30" },
    width: { type: "string", default: "360" },
    height: { type: "string", default: "640" },
    duration: { type: "string", default: "3" },
    output: { type: "string", default: "animation_output.mp4" },
    tempDir: { type: "string", default: "./temp_frames" }
  },
  allowPositionals: true
});

// Validate required arguments
if (positionals.length === 0) {
  console.error("Error: URL is required as first argument");
  process.exit(1);
}

const URL = positionals[0];
const WIDTH = parseInt(args.width);
const HEIGHT = parseInt(args.height);
const FPS = parseInt(args.fps);
const DURATION = parseFloat(args.duration);
const OUTPUT = args.output;
const FRAMES_DIR = path.resolve(args.tempDir);

// Validate numeric arguments
if (isNaN(WIDTH) || WIDTH <= 0) {
  console.error("Error: Invalid width value");
  process.exit(1);
}
if (isNaN(HEIGHT) || HEIGHT <= 0) {
  console.error("Error: Invalid height value");
  process.exit(1);
}
if (isNaN(FPS) || FPS <= 0) {
  console.error("Error: Invalid fps value");
  process.exit(1);
}
if (isNaN(DURATION) || DURATION <= 0) {
  console.error("Error: Invalid duration value");
  process.exit(1);
}

// Setup temporary frames directory
async function setupFramesDirectory() {
  if (existsSync(FRAMES_DIR)) {
    await rm(FRAMES_DIR, { recursive: true });
  }
  await mkdir(FRAMES_DIR, { recursive: true });
}

// Cleanup temporary frames directory
async function cleanupFramesDirectory() {
  if (existsSync(FRAMES_DIR)) {
    await rm(FRAMES_DIR, { recursive: true });
  }
}

// Capture frames from URL
async function captureFrames() {
  console.log(`🌐 Loading URL: ${URL}`);
  
  const browser = await chromium.launch({
    channel: 'chrome',
    headless: true,
    args: [
      "--hide-scrollbars",
      "--disable-gpu", 
      "--no-sandbox",
      "--disable-dev-shm-usage",
      "--force-color-profile=srgb"
    ]
  });

  const context = await browser.newContext({ 
    viewport: { width: WIDTH, height: HEIGHT } 
  });
  const page = await context.newPage();
  
  try {
    // Load the page with increased timeout
    await page.goto(URL, { waitUntil: "networkidle", timeout: 30000 });
    
    // Wait for DOM and scripts to fully load
    await page.waitForLoadState('domcontentloaded');
    await page.waitForLoadState('networkidle');
    
    // Give extra time for async initialization
    await page.waitForTimeout(2000);
    
    // Wait for capture interface to be available
    await page.waitForFunction(() => {
      if (typeof window !== 'undefined') {
        console.log('DOM ready:', document.readyState);
        console.log('CaptureHelper available:', !!window.CaptureHelper);
        console.log('__CAPTURE__ available:', !!window.__CAPTURE__);
        if (window.__CAPTURE__) {
          console.log('__CAPTURE__ methods:', Object.keys(window.__CAPTURE__));
        }
      }
      return !!window.__CAPTURE__;
    }, { timeout: 15000 });
    
    // Get animation duration from the page
    const pageDuration = await page.evaluate(() => window.__CAPTURE__.duration || 3);
    const actualDuration = DURATION || pageDuration;
    
    console.log(`🎬 Animation duration: ${actualDuration}s`);
    
    // Initialize the animation
    await page.evaluate(() => window.__CAPTURE__.init());
    
    const totalFrames = Math.round(actualDuration * FPS);
    console.log(`🎞️  Total frames to capture: ${totalFrames}`);

    const captureStartTime = Date.now();

    // Capture frames
    for (let i = 0; i < totalFrames; i++) {
      // Set the exact frame position
      await page.evaluate(async ({ frameIndex, fps }) => {
        await window.__CAPTURE__.setFrame(frameIndex, fps);
      }, { frameIndex: i, fps: FPS });

      // Ensure all layout and paint operations are complete
      await page.evaluate(() => 
        new Promise(resolve =>
          requestAnimationFrame(() => requestAnimationFrame(resolve))
        )
      );

      // Capture the frame
      const buffer = await page.screenshot({
        type: "png"
      });

      // Save frame to disk with zero-padded filename
      const frameFilename = path.join(FRAMES_DIR, `frame_${i.toString().padStart(6, '0')}.png`);
      await writeFile(frameFilename, buffer);

      // Progress reporting (less verbose for backend usage)
      if ((i + 1) % Math.max(1, Math.floor(totalFrames / 10)) === 0) {
        const progress = ((i + 1) / totalFrames * 100).toFixed(1);
        console.log(`🎥 Captured frame ${i + 1}/${totalFrames} (${progress}%)`);
      }
    }

    const captureTime = (Date.now() - captureStartTime) / 1000;
    const captureSpeed = totalFrames / captureTime;
    
    console.log(`✅ Frame capture completed in ${captureTime.toFixed(2)}s (${captureSpeed.toFixed(1)} fps)`);

  } catch (error) {
    console.error(`❌ Capture error:`, error.message);
    throw error;
  } finally {
    await browser.close();
  }
}

// Assemble video from frames using FFmpeg
async function assembleVideo() {
  console.log(`🎞️  Assembling video: ${OUTPUT}`);
  
  return new Promise((resolve, reject) => {
    const ffmpeg = spawn("ffmpeg", [
      "-y",
      "-framerate", String(FPS),
      "-i", path.join(FRAMES_DIR, "frame_%06d.png"),
      "-c:v", "libx264",
      "-pix_fmt", "yuv420p",
      "-crf", "18",
      "-preset", "veryfast",
      OUTPUT
    ], { 
      stdio: ["pipe", "pipe", "pipe"]
    });

    let stderr = "";
    ffmpeg.stderr.on("data", (data) => {
      stderr += data.toString();
    });

    ffmpeg.on("exit", code => {
      if (code === 0) {
        console.log(`📁 Output saved: ${OUTPUT}`);
        resolve();
      } else {
        console.error("FFmpeg stderr:", stderr);
        reject(new Error(`FFmpeg failed with exit code ${code}`));
      }
    });

    ffmpeg.on("error", reject);
  });
}

async function main() {
  const overallStartTime = Date.now();
  
  console.log(`🎬 Starting URL-based animation capture...`);
  console.log(`🌐 URL: ${URL}`);
  console.log(`📐 Resolution: ${WIDTH}x${HEIGHT} @ ${FPS}fps`);

  try {
    // Setup
    const setupStartTime = Date.now();
    await setupFramesDirectory();
    const setupTime = (Date.now() - setupStartTime) / 1000;
    console.log(`🔧 Setup time: ${setupTime.toFixed(2)}s`);
    
    // Capture frames
    await captureFrames();

    // Assemble final video
    const assemblyStartTime = Date.now();
    await assembleVideo();
    const assemblyTime = (Date.now() - assemblyStartTime) / 1000;

    const totalTime = (Date.now() - overallStartTime) / 1000;
    const renderSpeed = DURATION / totalTime;
    
    console.log(`📊 PERFORMANCE SUMMARY:`);
    console.log(`🔧 Setup: ${setupTime.toFixed(2)}s`);
    console.log(`🎥 Frame capture: ${(totalTime - setupTime - assemblyTime).toFixed(2)}s`);
    console.log(`🎞️  Video assembly: ${assemblyTime.toFixed(2)}s`);
    console.log(`⏱️  Total time: ${totalTime.toFixed(2)}s`);
    console.log(`🚀 Speed: ${renderSpeed.toFixed(1)}x faster than real-time`);

  } catch (error) {
    console.error("❌ Error during capture:", error.message);
    process.exit(1);
  } finally {
    // Cleanup
    await cleanupFramesDirectory();
    console.log(`🧹 Cleaned up temporary frames`);
  }
}

main().catch(err => {
  console.error("💥 Fatal error:", err.message);
  process.exit(1);
});