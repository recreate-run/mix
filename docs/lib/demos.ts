/**
 * Homepage Demo Showcase Configuration
 *
 * Add new demos here - they will automatically appear on the homepage
 */

export interface CodeSnippet {
    language: string;
    fileName: string;
    code: string;
    githubUrl: string;
}

export interface Demo {
    title: string;
    description: string;
    codeSnippets: CodeSnippet[];
    videoSrc?: string;
    youtubeId?: string;
    videoCaption?: string;
}

export const homepageDemos: Demo[] = [
    {
        title: "Portfolio Analysis",
        description: "Upload CSV, get AI-powered insights",
        youtubeId: "H-Ee2OwACRo",
        videoCaption: "Watch Mix upload data, analyze trends, and generate interactive charts",
        codeSnippets: [
            {
                language: "python",
                fileName: "portfolio_analysis.py",
                githubUrl: "https://github.com/recreate-run/mix-cookbooks/blob/main/python_examples/demos/portfolio_analysis.py",
                code: `from mix_python_sdk import Mix

async with Mix(server_url="http://localhost:8088") as mix:
    # Configure preferences
    mix.preferences.update_preferences(
        preferred_provider="anthropic",
        main_agent_model="claude-sonnet-4-5"
    )

    # Create session and upload CSV
    session = mix.sessions.create(
        title="Portfolio Analysis Q4 2024"
    )

    file_info = mix.files.upload_session_file(
        id=session.id,
        file=open("portfolio.csv", "rb")
    )

    # AI analyzes and creates visualizations
    await stream_message(
        mix, session.id,
        f"Analyze @{file_info.url} and find top winners/losers."
    )`
            },
            {
                language: "typescript",
                fileName: "portfolio_analysis.ts",
                githubUrl: "https://github.com/recreate-run/mix-cookbooks/blob/main/typescript_examples/demos/portfolio_analysis.ts",
                code: `import { Mix } from "mix-typescript-sdk";
import { sendWithCallbacks } from "./helpers";

const mix = new Mix({ serverURL: "http://localhost:8088" });

// Configure preferences
await mix.preferences.update({
  preferred_provider: "anthropic",
  main_agent_model: "claude-sonnet-4-5",
});

// Create session and upload CSV
const session = await mix.sessions.create({
  title: "Portfolio Analysis Q4 2024",
});

const csvFile = new File([csvContent], "portfolio.csv");
const fileInfo = await mix.files.upload({
  id: session.id,
  requestBody: { file: csvFile },
});

// AI analyzes and creates visualizations
await sendWithCallbacks(
  mix, session.id,
  \`Analyze @\${fileInfo.url} and find top winners/losers.\`
);`
            }
        ]
    },

    {
        title: "Multimodal Web Search",
        description: "Search, analyze, and extract video highlights",
        youtubeId: "rWsg4EMWX0c",
        videoCaption: "AI searches for videos, finds key moments, and downloads highlights",
        codeSnippets: [
            {
                language: "python",
                fileName: "web_search_multimodal.py",
                githubUrl: "https://github.com/recreate-run/mix-cookbooks/blob/main/python_examples/demos/web_search_multimodal.py",
                code: `from mix_python_sdk import Mix

async with Mix(server_url="http://localhost:8088") as mix:
    # Configure preferences
    mix.preferences.update_preferences(
        preferred_provider="anthropic",
        main_agent_model="claude-sonnet-4-5"
    )

    session = mix.sessions.create(
        title="Web Search Demo"
    )

    # AI searches, analyzes, downloads highlights
    await stream_message(
        mix, session.id,
        "Find top 3 karpathy LLM videos, extract key "
        "10-sec clips, download and show them."
    )`
            },
            {
                language: "typescript",
                fileName: "web_search_multimodal.ts",
                githubUrl: "https://github.com/recreate-run/mix-cookbooks/blob/main/typescript_examples/demos/web_search_multimodal.ts",
                code: `import { Mix } from "mix-typescript-sdk";
import { sendWithCallbacks } from "./helpers";

const mix = new Mix({ serverURL: "http://localhost:8088" });

// Configure preferences
await mix.preferences.update({
  preferred_provider: "anthropic",
  main_agent_model: "claude-sonnet-4-5",
});

const session = await mix.sessions.create({
  title: "Web Search Demo",
});

// AI searches, analyzes, downloads highlights
await sendWithCallbacks(
  mix, session.id,
  "Find top 3 karpathy LLM videos, extract key " +
  "10-sec clips, download and show them."
);`
            }
        ]
    },

    {
        title: "TikTok Video Creation",
        description: "Generate viral-style videos with AI",
        youtubeId: "7KDE7obwjBU",
        videoCaption: "AI finds trending content, creates short videos with animations",
        codeSnippets: [
            {
                language: "python",
                fileName: "tiktok_video.py",
                githubUrl: "https://github.com/recreate-run/mix-cookbooks/blob/main/python_examples/demos/tiktok_video.py",
                code: `from mix_python_sdk import Mix

async with Mix(server_url="http://localhost:8088") as mix:
    # Configure preferences
    mix.preferences.update_preferences(
        preferred_provider="anthropic",
        main_agent_model="claude-sonnet-4-5"
    )

    session = mix.sessions.create(
        title="TikTok Video Demo"
    )

    # AI finds content, creates video, adds animations
    await stream_message(
        mix, session.id,
        "Find top cat video and create 5-sec TikTok clip "
        "with title animation. Export and show."
    )`
            },
            {
                language: "typescript",
                fileName: "tiktok_video.ts",
                githubUrl: "https://github.com/recreate-run/mix-cookbooks/blob/main/typescript_examples/demos/tiktok_video.ts",
                code: `import { Mix } from "mix-typescript-sdk";
import { sendWithCallbacks } from "./helpers";

const mix = new Mix({ serverURL: "http://localhost:8088" });

// Configure preferences
await mix.preferences.update({
  preferred_provider: "anthropic",
  main_agent_model: "claude-sonnet-4-5",
});

const session = await mix.sessions.create({
  title: "TikTok Video Demo",
});

// AI finds content, creates video, adds animations
await sendWithCallbacks(
  mix, session.id,
  "Find top cat video and create 5-sec TikTok clip " +
  "with title animation. Export and show."
);`
            }
        ]
    },

    // Add more demos here - they will automatically show on homepage

    // To add a new language to existing demos:
    // 1. Add a new CodeSnippet object to the codeSnippets array
    // 2. Set language to the language name (e.g., "javascript", "go", "rust")
    // 3. The dropdown will automatically include the new language
];
