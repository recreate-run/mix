/**
 * HOW TO ADD MORE DEMOS TO HOMEPAGE
 *
 * Edit: /lib/demos.ts
 *
 * Simply add a new demo object to the homepageDemos array:
 */

/*
// In /lib/demos.ts

export const homepageDemos: Demo[] = [
  {
    icon: "📊",
    title: "Portfolio Analysis",
    description: "Upload CSV, get AI-powered insights",
    fileName: "portfolio_analysis.py",
    videoSrc: "/videos/financial_portfolio_demo.mp4",
    videoCaption: "Watch Mix upload data, analyze trends, and generate interactive charts",
    code: `from mix_python_sdk import Mix

with Mix(server_url="http://localhost:8088") as mix:
    # your code here`
  },

  // Add your new demo here:
  {
    icon: "🎥",  // Emoji icon
    title: "Video Analysis",  // Demo title
    description: "Find and analyze video content",  // Short description
    fileName: "video_search.py",  // Filename shown in code block
    videoSrc: "/videos/your-video-demo.mp4",  // Video path (put video in docs/public/videos/)
    videoCaption: "AI finds, analyzes, and extracts highlights from videos",  // Optional caption
    code: `from mix_python_sdk import Mix

with Mix(server_url="http://localhost:8088") as mix:
    session = mix.sessions.create(title="Video Search")
    mix.messages.send(
        session_id=session.id,
        content="Find top 3 AI videos and extract 10 sec highlights"
    )`
  },

  // Add more demos here - they automatically show on homepage!
];

*/

/**
 * That's it! No need to touch page.tsx
 *
 * The page.tsx automatically maps over the homepageDemos array and renders each demo.
 * Demos are displayed with:
 * - Code block on left with syntax highlighting
 * - Video on right with 16:9 aspect ratio
 * - Responsive layout (stacks on mobile)
 * - Proper spacing between multiple demos
 */
