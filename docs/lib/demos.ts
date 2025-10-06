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
  videoSrc?: string;
  youtubeId?: string;
  videoCaption?: string;
  githubUrl?: string;
}

export const homepageDemos: Demo[] = [
  {
    title: "Portfolio Analysis",
    description: "Upload CSV, get AI-powered insights",
    fileName: "portfolio_analysis.py",
    youtubeId: "hLkipswyJJ4",
    videoCaption: "Watch Mix upload data, analyze trends, and generate interactive charts",
    githubUrl: "https://github.com/recreate-run/mix-cookbooks/blob/main/python_examples/demos/portfolio_analysis.py",
    code: `from mix_python_sdk import Mix

async with Mix(server_url="http://localhost:8088") as mix:
    mix.preferences.update_preferences(
        preferred_provider="anthropic",
        main_agent_model="claude-sonnet-4-5"
    )

    session = mix.sessions.create(title="Portfolio Analysis Q4 2024")

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
    await stream_message(
        mix, session.id,
        f"Analyze @{file_info.url} and find top winners/losers. Show 3 plots."
    )`
  },

  {
    title: "Multimodal Web Search",
    description: "Search, analyze, and extract video highlights",
    fileName: "web_search_multimodal.py",
    youtubeId: "N--FfjtUjwY",
    videoCaption: "AI searches for videos, finds key moments, and downloads highlights",
    githubUrl: "https://github.com/recreate-run/mix-cookbooks/blob/main/python_examples/demos/web_search_multimodal.py",
    code: `from mix_python_sdk import Mix

async with Mix(server_url="http://localhost:8088") as mix:
    mix.preferences.update_preferences(
        preferred_provider="anthropic",
        main_agent_model="claude-sonnet-4-5"
    )

    session = mix.sessions.create(title="Web Search Demo")

    # AI searches, analyzes videos, downloads highlights
    await stream_message(
        mix, session.id,
        "Find top 3 karpathy LLM videos, extract key 10-sec clips, download and show them."
    )`
  },

  {
    title: "TikTok Video Creation",
    description: "Generate viral-style videos with AI",
    fileName: "tiktok_video.py",
    youtubeId: "RP3eNYRhfyM",
    videoCaption: "AI finds trending content, creates short videos with animations",
    githubUrl: "https://github.com/recreate-run/mix-cookbooks/blob/main/python_examples/demos/tiktok_video.py",
    code: `from mix_python_sdk import Mix

async with Mix(server_url="http://localhost:8088") as mix:
    mix.preferences.update_preferences(
        preferred_provider="anthropic",
        main_agent_model="claude-sonnet-4-5"
    )

    session = mix.sessions.create(title="TikTok Video Demo")

    # AI finds content, creates video, adds animations
    await stream_message(
        mix, session.id,
        "Find top cat video and create 5-sec TikTok clip with title animation. Export and show."
    )`
  },

  // Add more demos here - they will automatically show on homepage
];
