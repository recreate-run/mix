/**
 * Homepage Demo Showcase Configuration
 *
 * Add new demos here - they will automatically appear on the homepage
 */

export interface Demo {
  title: string;
  description: string;
  fileName: string;
  code: string;
  videoSrc: string;
  videoCaption?: string;
  githubUrl?: string;
}

export const homepageDemos: Demo[] = [
  {
    title: "Portfolio Analysis",
    description: "Upload CSV, get AI-powered insights",
    fileName: "portfolio_analysis.py",
    videoSrc: "/videos/financial_portfolio_demo.mp4",
    videoCaption: "Watch Mix upload data, analyze trends, and generate interactive charts",
    githubUrl: "https://github.com/recreate-run/mix-cookbooks/blob/main/python_examples/demos/portfolio_analysis.py",
    code: `from mix_python_sdk import Mix

with Mix(server_url="http://localhost:8088") as mix:
    # Configure AI provider
    mix.preferences.update_preferences(
        preferred_provider="anthropic",
        main_agent_model="claude-sonnet-4-5"
    )

    # Create analysis session
    session = mix.sessions.create(
        title="Portfolio Analysis Q4 2024"
    )

    # Upload portfolio CSV
    with open("portfolio.csv", "rb") as f:
        file_info = mix.files.upload_session_file(
            id=session.id,
            file={
                "file_name": "portfolio.csv",
                "content": f,
                "content_type": "text/csv"
            }
        )

    # AI analyzes and creates visualizations
    mix.streaming.send_streaming_message(
        id=session.id,
        content=f"Analyze @{file_info.url} and find top winners/losers in Q4. Show 3 plots."
    )`
  },

  {
    title: "Multimodal Web Search",
    description: "Search, analyze, and extract video highlights",
    fileName: "web_search_multimodal.py",
    videoSrc: "/videos/multimodal_web_search.mp4", // Replace with actual video when available
    videoCaption: "AI searches for videos, finds key moments, and downloads highlights",
    githubUrl: "https://github.com/recreate-run/mix-cookbooks/blob/main/python_examples/demos/web_search_multimodal.py",
    code: `from mix_python_sdk import Mix

with Mix(server_url="http://localhost:8088") as mix:
    # Configure AI provider
    mix.preferences.update_preferences(
        preferred_provider="anthropic",
        main_agent_model="claude-sonnet-4-5"
    )

    # Create search session
    session = mix.sessions.create(
        title="Web Search Multimodal"
    )

    # AI searches, analyzes, and extracts video highlights
    mix.streaming.send_streaming_message(
        id=session.id,
        content="Find top 3 Karpathy LLM videos, find the most important 10 second section from each, then download and show them."
    )`
  },

  {
    title: "TikTok Video Creation",
    description: "Generate viral-style videos with AI",
    fileName: "tiktok_video.py",
    videoSrc: "/videos/launch_devtools.mp4", // Replace with actual video when available
    videoCaption: "AI finds trending content, creates short videos with animations",
    githubUrl: "https://github.com/recreate-run/mix-cookbooks/blob/main/python_examples/demos/tiktok_video.py",
    code: `from mix_python_sdk import Mix

with Mix(server_url="http://localhost:8088") as mix:
    # Configure AI provider
    mix.preferences.update_preferences(
        preferred_provider="anthropic",
        main_agent_model="claude-sonnet-4-5"
    )

    # Create video generation session
    session = mix.sessions.create(
        title="TikTok Video Creation"
    )

    # AI finds content, creates video, adds animations
    mix.streaming.send_streaming_message(
        id=session.id,
        content="Find the top cat video and create a 5 sec tiktok video from it. Add a title animation. Export it and show."
    )`
  },

  // Add more demos here - they will automatically show on homepage
];
