# Home Page Redesign - Mix Documentation

**Goal:** Transform the text-only home page into a visual, compelling landing page that shows Mix in action within 10 seconds.

---

## 🎯 Current vs. Proposed

### Current Home Page (Text-only)
```
- Title + description
- 2 "Choose Your Path" cards
- 4 Documentation cards
- Why Mix? (bullets)
- Need Help? (links)
```

### Proposed Home Page (Visual-first)
```
1. HERO SECTION - 30sec demo GIF
2. SEE IT IN ACTION - 3 demo GIFs
3. KEY FEATURES - Icons + screenshots
4. CHOOSE YOUR PATH - Visual cards
5. WHY MIX - Comparison visual
6. QUICK CODE SNIPPET - Live example
7. GET STARTED CTA
```

---

## 📹 GIFs/Videos to Record

### Priority 1: HERO DEMO (30 seconds)
**This is the MOST IMPORTANT asset - it's the first thing users see**

#### What to Show:
A complete end-to-end workflow that demonstrates Mix's power

#### Recording Script:

**Scene 1: DevTools Launch (0-5s)**
1. Terminal: `make dev`
2. DevTools window opens
3. Zoom to chat interface

**Scene 2: User Input (5-10s)**
1. Show cursor typing in chat:
   ```
   Find the top cat video and create a 5 sec tiktok video from it
   ```
2. Press Enter
3. Show "thinking" animation starting

**Scene 3: Agent Working (10-25s)**
1. Show streaming response:
   - "🔍 Searching for cat videos..."
   - Tool execution: WebSearch
   - "🎬 Creating video from best result..."
   - Tool execution: ShowMedia (video creation)
   - Progress indicators

**Scene 4: Final Result (25-30s)**
1. Show completed video playing in UI
2. Fade to end card:
   ```
   Built with Mix
   github.com/recreate-run/mix
   ```

#### Recording Tips:
- **Resolution:** 1920x1080 or 1440x900
- **Frame Rate:** 30fps
- **Tool:** QuickTime (Mac), OBS (any platform), or Loom
- **Cursor:** Make cursor movements visible and smooth
- **Speed:** Real-time, no speedup (shows actual performance)
- **Audio:** Optional background music (subtle)

#### Export Settings:
- **Format:** MP4 first (convert to GIF later)
- **Max Size:** 10MB for GIF, 20MB for MP4
- **Converter:** Use ezgif.com or Gifski for MP4→GIF

---

### Priority 2: FEATURE DEMOS (3 GIFs, 10 seconds each)

#### GIF 1: TikTok Video Creation
**File:** `tiktok-demo.gif`

**What to record:**
1. Open DevTools
2. Type: "Create a 5-second video about golden retrievers"
3. Show agent searching web
4. Show video being created
5. Final video plays

**Key moments to capture:**
- User prompt
- Web search tool execution
- Video creation tool
- Final output

---

#### GIF 2: Multimodal Search
**File:** `search-demo.gif`

**What to record:**
1. Type: "Find videos about kerala puttu and show me"
2. Agent uses WebSearch
3. Shows video thumbnails/results
4. Displays in UI

**Key moments:**
- Search query
- Tool execution (Brave Search)
- Results appearing
- Media display

---

#### GIF 3: Session Management
**File:** `session-demo.gif`

**What to record:**
1. Show Sessions sidebar
2. Create new session (click + or /clear)
3. Switch between sessions
4. Show conversation history persisting
5. Fork a session

**Key moments:**
- Multiple sessions visible
- Switching context
- History preservation
- Fork action

---

### Priority 3: UI SCREENSHOTS (Static images)

#### Screenshot 1: DevTools Interface
**File:** `devtools-ui.png`

**What to capture:**
- Full DevTools window
- Chat interface with a conversation
- Sessions sidebar visible
- Settings icon visible
- Tools panel showing

**Annotations to add:**
1. Arrow pointing to chat input: "Natural language prompts"
2. Arrow to sessions: "Manage conversations"
3. Arrow to settings: "Configure AI providers"
4. Arrow to tools: "Monitor agent actions"

**Tool for annotations:** Skitch, Cleanshot X, or Shottr

---

#### Screenshot 2: Settings/Authentication
**File:** `settings-auth.png`

**What to capture:**
- Settings dialog open
- Anthropic provider section visible
- OAuth and API Key options shown
- Model selection visible

**Annotations:**
1. "One-click OAuth" → Sign in with Claude Code button
2. "Or use API Key" → API key input
3. "Choose model" → Model dropdown

---

#### Screenshot 3: Tool Execution in Action
**File:** `tool-execution.png`

**What to capture:**
- Chat showing agent using tools
- Tool execution cards visible
- Input/output visible
- Status indicators (thinking, executing, complete)

---

### Priority 4: Code Demo Screenshot
**File:** `python-sdk-demo.png`

**What to capture:**
1. **Left side:** Python code
   ```python
   from mix_python_sdk import Mix

   client = Mix(server_url="http://localhost:8088")
   session = client.sessions.create(title="Video Creator")

   response = client.messages.send(
       session_id=session.id,
       content="Create a 5-second video about cats"
   )
   ```

2. **Right side:** Terminal output or result
   - Show the session being created
   - Show the response
   - Or show the actual video created

**Tool:** Split-screen screenshot or code editor + terminal

---

## 🎨 Home Page Layout Design

### Section 1: HERO SECTION

```markdown
---
title: Mix - Agentic AI Backbone for Multimodal Apps
description: Open-source AI orchestration platform for video, images, and text
---

<div className="hero-section" style="text-align: center; padding: 4rem 0;">

  # Build AI Workflows That Actually Work

  <p className="hero-subtitle">
    Open-source AI orchestration for multimodal workflows. HTTP API, DevTools GUI, Python & TypeScript SDKs.
  </p>

  <div className="hero-demo">
    <!-- 30-SECOND DEMO GIF HERE -->
    <img src="/images/hero-demo.gif" alt="Mix in action" />
  </div>

  <div className="hero-cta">
    <Button href="/docs/mix/quickstart">Get Started</Button>
    <Button href="https://github.com/recreate-run/mix" variant="outline">
      View on GitHub ↗
    </Button>
  </div>

  <p className="hero-meta">
    🚀 5-minute setup • 🔓 Apache 2.0 • ⭐ [GitHub Stars Count]
  </p>

</div>
```

**What to record for this:** The 30-second hero demo (Priority 1 above)

---

### Section 2: SEE IT IN ACTION

```markdown
## See What You Can Build

<div className="demo-grid">
  <Card>
    <h3>🎬 Create TikTok Videos</h3>
    <img src="/images/tiktok-demo.gif" alt="TikTok video creation" />
    <p>Turn prompts into polished videos automatically</p>
  </Card>

  <Card>
    <h3>🔍 Multimodal Search</h3>
    <img src="/images/search-demo.gif" alt="Multimodal search" />
    <p>Search, analyze, and display video content</p>
  </Card>

  <Card>
    <h3>📊 Session Management</h3>
    <img src="/images/session-demo.gif" alt="Session management" />
    <p>Organize and switch between AI workflows</p>
  </Card>
</div>
```

**What to record:** The 3 feature demos (Priority 2 above)

---

### Section 3: KEY FEATURES

```markdown
## Why Choose Mix?

<div className="features-grid">

  <Feature icon="🎯">
    ### HTTP-First API
    RESTful API makes debugging easier than stdio-based tools
    <img src="/images/api-example.png" alt="API call" className="feature-img" />
  </Feature>

  <Feature icon="🎨">
    ### DevTools GUI
    Interactive playground for testing and debugging
    <img src="/images/devtools-ui.png" alt="DevTools" className="feature-img" />
  </Feature>

  <Feature icon="🔓">
    ### Zero Lock-in
    All data stored as plain text and native files
    <Mermaid chart="
      graph LR
        A[Your Data] --> B[SQLite DB]
        A --> C[Native Files]
        A --> D[Plain Text]
    "/>
  </Feature>

  <Feature icon="🤖">
    ### Multi-Provider
    Anthropic • OpenAI • Google • OpenRouter
    <div className="provider-logos">
      [Provider logos or badges]
    </div>
  </Feature>

  <Feature icon="📦">
    ### Built for Multimodal
    Video, images, and text workflows out of the box
  </Feature>

  <Feature icon="🛠️">
    ### Python & TypeScript SDKs
    First-class SDK support, not an afterthought
  </Feature>

</div>
```

**What to record/create:**
- Screenshot of API call (Postman or curl in terminal)
- Screenshot of DevTools UI (Priority 3, Screenshot 1)
- Simple diagram for data storage (optional)

---

### Section 4: CHOOSE YOUR PATH

```markdown
## Get Started Your Way

<Tabs defaultValue="devtools">
  <TabsList>
    <TabsTrigger value="devtools">🎨 DevTools</TabsTrigger>
    <TabsTrigger value="python">🐍 Python SDK</TabsTrigger>
    <TabsTrigger value="typescript">⚡ TypeScript SDK</TabsTrigger>
  </TabsList>

  <TabsContent value="devtools">
    <div className="path-content">
      <div className="path-visual">
        <img src="/images/devtools-ui.png" alt="DevTools interface" />
      </div>
      <div className="path-description">
        <h3>Interactive GUI Playground</h3>
        <p>Perfect for: Testing workflows, debugging, exploration</p>

        **Quick start:**
        ```bash
        git clone https://github.com/recreate-run/mix.git
        cd mix
        make dev
        ```

        <Button href="/docs/mix/quickstart#quickstart-with-devtools">
          Start with DevTools →
        </Button>
      </div>
    </div>
  </TabsContent>

  <TabsContent value="python">
    <div className="path-content">
      <div className="path-visual">
        <img src="/images/python-sdk-demo.png" alt="Python SDK code" />
      </div>
      <div className="path-description">
        <h3>Build with Python</h3>
        <p>Perfect for: Production apps, automation, data workflows</p>

        **Quick start:**
        ```python
        from mix_python_sdk import Mix

        client = Mix(server_url="http://localhost:8088")
        session = client.sessions.create(title="My Workflow")

        response = client.messages.send(
            session_id=session.id,
            content="Analyze this video"
        )
        ```

        <Button href="/docs/mix/python-sdk/installation">
          Start with Python →
        </Button>
      </div>
    </div>
  </TabsContent>

  <TabsContent value="typescript">
    <div className="path-content">
      <div className="path-visual">
        ```typescript
        import { Mix } from 'mix-typescript-sdk';

        const mix = new Mix({
          serverURL: 'http://localhost:8088'
        });

        const session = await mix.sessions.create({
          title: 'My Workflow'
        });
        ```
      </div>
      <div className="path-description">
        <h3>Build with TypeScript</h3>
        <p>Perfect for: Web apps, React/Next.js, real-time UIs</p>

        <Button href="/docs/mix/typescript-sdk/quickstart">
          Start with TypeScript →
        </Button>
      </div>
    </div>
  </TabsContent>
</Tabs>
```

**What to record:**
- Screenshot of Python code + output (Priority 4 above)
- Screenshot of TypeScript code + output (similar to Python)

---

### Section 5: WHY MIX?

```markdown
## Mix vs. Alternatives

<Callout type="info">
Mix combines the power of agentic SDKs with the flexibility of HTTP APIs and the usability of DevTools.
</Callout>

<ComparisonTable>
  <thead>
    <tr>
      <th>Feature</th>
      <th>Mix</th>
      <th>Claude Code SDK</th>
      <th>LangChain</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td>Open Source</td>
      <td>✅ Apache 2.0</td>
      <td>❌</td>
      <td>✅</td>
    </tr>
    <tr>
      <td>API Type</td>
      <td>✅ HTTP REST</td>
      <td>stdio</td>
      <td>~</td>
    </tr>
    <tr>
      <td>DevTools GUI</td>
      <td>✅ Built-in</td>
      <td>❌</td>
      <td>❌</td>
    </tr>
    <tr>
      <td>Multimodal</td>
      <td>✅ Video/Image/Text</td>
      <td>Code-focused</td>
      <td>~</td>
    </tr>
    <tr>
      <td>Self-Host</td>
      <td>✅</td>
      <td>❌</td>
      <td>✅</td>
    </tr>
  </tbody>
</ComparisonTable>

**When to choose Mix:**
- 🎬 Building multimodal workflows (video/image processing)
- 🔌 Need HTTP API for multiple clients
- 🐛 Want visual debugging tools
- 🏢 Require full control over infrastructure
```

**No recording needed** - This is a simple comparison table

---

### Section 6: QUICK CODE EXAMPLE

```markdown
## See It in Code

<Tabs defaultValue="python">
  <TabsList>
    <TabsTrigger value="python">Python</TabsTrigger>
    <TabsTrigger value="typescript">TypeScript</TabsTrigger>
    <TabsTrigger value="curl">cURL</TabsTrigger>
  </TabsList>

  <TabsContent value="python">
    ```python
    from mix_python_sdk import Mix

    # Create client
    client = Mix(server_url="http://localhost:8088")

    # Create session
    session = client.sessions.create(title="Video Creator")

    # Send message
    response = client.messages.send(
        session_id=session.id,
        content="Create a 5-second video about golden retrievers"
    )

    print(f"Response: {response.text}")
    # Output: "I've created a video showcasing golden retrievers..."
    ```
  </TabsContent>

  <TabsContent value="typescript">
    ```typescript
    import { Mix } from 'mix-typescript-sdk';

    const mix = new Mix({ serverURL: 'http://localhost:8088' });

    const session = await mix.sessions.create({
      title: 'Video Creator'
    });

    const response = await mix.messages.send({
      sessionId: session.id,
      content: 'Create a 5-second video about golden retrievers'
    });

    console.log(response.text);
    ```
  </TabsContent>

  <TabsContent value="curl">
    ```bash
    # Create session
    SESSION_ID=$(curl -X POST http://localhost:8088/api/v1/sessions \
      -H "Content-Type: application/json" \
      -d '{"title":"Video Creator"}' | jq -r '.id')

    # Send message
    curl -X POST http://localhost:8088/api/v1/sessions/$SESSION_ID/messages \
      -H "Content-Type: application/json" \
      -d '{"text":"Create a 5-second video about golden retrievers"}'
    ```
  </TabsContent>
</Tabs>
```

---

### Section 7: CALL TO ACTION

```markdown
## Ready to Build?

<div className="cta-section">
  <div className="cta-primary">
    <h3>Get started in 5 minutes</h3>
    <p>Choose your preferred way to start building with Mix</p>

    <div className="cta-buttons">
      <Button href="/docs/mix/quickstart" size="lg">
        🚀 Quickstart Guide
      </Button>
      <Button href="/docs/mix/examples" variant="outline" size="lg">
        📚 View Examples
      </Button>
    </div>
  </div>

  <div className="cta-secondary">
    <div className="cta-links">
      <a href="https://github.com/recreate-run/mix">⭐ Star on GitHub</a>
      <a href="https://github.com/recreate-run/mix/discussions">💬 Join Discussions</a>
      <a href="/docs/mix/troubleshooting">🛟 Get Help</a>
    </div>
  </div>
</div>

---

<div className="footer-badges">
  ![Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)
  ![GitHub Stars](https://img.shields.io/github/stars/recreate-run/mix)
  ![Built with Go](https://img.shields.io/badge/Built%20with-Go-00ADD8?logo=go)
  ![Tauri](https://img.shields.io/badge/Desktop-Tauri-24C8DB?logo=tauri)
</div>
```

---

## 📋 Recording Checklist

### Before Recording:
- [ ] Clean up DevTools (close unnecessary windows)
- [ ] Prepare prompts in advance (copy-paste ready)
- [ ] Set up authentication (OAuth or API key working)
- [ ] Test workflow once to ensure it works
- [ ] Close notifications/distractions
- [ ] Set screen resolution (1920x1080 or 1440x900)

### GIFs to Record:
- [ ] **Priority 1:** 30-second hero demo (end-to-end workflow)
- [ ] **Priority 2a:** TikTok video creation (10s)
- [ ] **Priority 2b:** Multimodal search (10s)
- [ ] **Priority 2c:** Session management (10s)

### Screenshots to Take:
- [ ] DevTools UI (full interface, annotated)
- [ ] Settings/Auth dialog (OAuth + API key visible)
- [ ] Tool execution in action
- [ ] Python SDK code + output
- [ ] TypeScript SDK code + output

### Post-Recording:
- [ ] Convert MP4 to GIF (optimize size <5MB)
- [ ] Add annotations to screenshots
- [ ] Save to `/docs/public/images/`
- [ ] Test GIFs load properly in docs
- [ ] Check file sizes (optimize if needed)

---

## 🎬 Recording Tips

### For Smooth GIFs:
1. **Slow down:** Move cursor deliberately, not frantically
2. **Pause:** Hold on important moments for 1-2 seconds
3. **Clear actions:** One action at a time, don't rush
4. **Zoom in:** Make sure text is readable (14pt+ font)

### For Clean Screenshots:
1. **Hide personal info:** No email addresses, API keys
2. **Use sample data:** "Example User" not your real name
3. **Consistent theme:** All screenshots same color theme
4. **High contrast:** Ensure readability

### Tools Recommendation:
- **Mac:** QuickTime (recording), Gifski (MP4→GIF), Cleanshot X (screenshots)
- **Windows:** OBS (recording), ScreenToGif (direct GIF), ShareX (screenshots)
- **Linux:** Peek (GIF), Kazam (video), Flameshot (screenshots)

---

## 🚀 Next Steps

1. **Record the hero demo first** - This is the most important
2. **Take DevTools screenshot** - Shows the interface
3. **Record 3 feature GIFs** - Shows capabilities
4. **Create the home page MDX** - Integrate all assets
5. **Test and iterate** - Get feedback, refine

Once these assets are ready, we'll:
1. Update `/docs/content/docs/index.mdx` with new layout
2. Add all GIFs/screenshots to `/docs/public/images/`
3. Test the page load speed
4. Get feedback before launch

---

**Questions to answer before recording:**
1. Which demo workflow best showcases Mix? (Cat video TikTok is great!)
2. What's your authentication setup? (OAuth vs API key - show both?)
3. Any specific features to highlight? (Tool execution, streaming, etc.)

Let me know when you're ready to start recording and I can help with specific prompts or troubleshooting!
