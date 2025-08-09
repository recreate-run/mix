# Remotion Video Integration - Session-Based Hybrid Approach

## Overview

This document outlines the integration of Remotion for generating animated video content in our chat application. Using a hybrid approach, videos are previewed client-side with the Remotion Player and exported server-side for high-quality output. Each session gets its own isolated Remotion project within its workspace directory.

## Architecture

```
┌─────────────────────┐   Video Config    ┌──────────────────────┐
│   Frontend (Tauri)  │ ─────────────────→ │   Backend (Go)       │
│                     │                    │                      │
│ • Video Preview     │                    │ • Session Isolation  │
│ • Remotion Player   │                    │ • Per-Session Setup  │
│ • Dynamic Config    │                    │ • Remotion CLI       │
│                     │                    │ • Workspace Management│
└─────────────────────┘                    └──────────────────────┘
                                                       │
                                                       ▼
                                            ┌──────────────────────┐
                                            │ Session Workspace    │
                                            │                      │
                                            │ • remotion_project/  │
                                            │ • input/ (existing)  │
                                            │ • output/            │
                                            │ • MIX.md (existing)  │
                                            └──────────────────────┘
```

### Frontend Responsibilities
- Display live video animation preview using `@remotion/player`
- Generate JSON configuration from AI responses
- Handle video customization and preview controls
- Request video exports from backend with session context

### Backend Responsibilities
- Clone Remotion template repository during session creation using bash scripts
- Initialize Remotion projects automatically when sessions are created
- Execute Remotion CLI within session workspace context via bash commands
- Manage exported video files within session isolation
- Use AI tools from `go_backend/internal/llm/tools/descriptions/remotion.md`

## Implementation Details

### 1. Frontend Integration

#### Package Installation
```bash
cd tauri_app
npm install @remotion/player
```

#### Message Type Extensions
```typescript
// Add to existing Message type
type RemotionVideoConfig = {
  composition: {
    durationInFrames: number;
    fps: number;
    width: number;
    height: number;
  };
  elements: VideoElement[];
};

type VideoElement = {
  type: 'text' | 'shape';
  content: string;
  from: number;
  durationInFrames: number;
  style?: {
    fontSize?: number;
    color?: string;
    backgroundColor?: string;
  };
  animation?: {
    type: 'fadeIn' | 'fadeOut' | 'slideIn' | 'slideOut';
    duration: number;
  };
  position: {
    x: number;
    y: number;
  };
};

type Message = {
  // ... existing fields
  remotionVideoConfig?: RemotionVideoConfig;
};
```

#### Dynamic Video Composition Component
```typescript
// components/remotion/DynamicVideoComposition.tsx
import { AbsoluteFill, Sequence, useCurrentFrame, useVideoConfig } from 'remotion';

export const DynamicVideoComposition: React.FC<{ config: RemotionVideoConfig }> = ({ config }) => {
  return (
    <AbsoluteFill style={{ backgroundColor: '#000' }}>
      {config.elements.map((element, index) => (
        <Sequence
          key={index}
          from={element.from}
          durationInFrames={element.durationInFrames}
        >
          <ElementRenderer element={element} />
        </Sequence>
      ))}
    </AbsoluteFill>
  );
};
```

#### Chat Integration
```typescript
// In ConversationDisplay component
const RemotionVideoPreview: React.FC<{ 
  config: RemotionVideoConfig; 
  sessionId: string;
}> = ({ config, sessionId }) => {
  const [isExporting, setIsExporting] = useState(false);
  
  const handleExport = async () => {
    // Export is handled by AI using export_video tool from remotion.md
    // User can ask AI to "export this video" and the AI will use bash commands
    // to render the video within the session workspace
    alert('Ask the AI to "export this video" to render it using Remotion CLI');
  };

  return (
    <div className="remotion-video-preview">
      <Player
        component={DynamicVideoComposition}
        inputProps={{ config }}
        durationInFrames={config.composition.durationInFrames}
        fps={config.composition.fps}
        compositionWidth={config.composition.width}
        compositionHeight={config.composition.height}
        controls
        style={{ width: '100%', maxWidth: '600px' }}
      />
      <div className="video-controls">
        <button onClick={handleExport} disabled={isExporting}>
          {isExporting ? 'Exporting...' : 'Export Video'}
        </button>
      </div>
    </div>
  );
};
```

### 2. Backend Integration

#### Directory Structure
```
go_backend/
├── internal/
│   └── llm/
│       └── tools/
│           └── descriptions/
│               └── remotion.md         # AI tool descriptions for video creation

# Per-session workspace structure:
{workspace_directory}/
├── input/                    # Existing session structure
│   ├── images/
│   ├── videos/
│   ├── audios/
│   └── text/
├── remotion_project/         # New: created during session creation
│   ├── package.json          # Cloned from template repository
│   ├── remotion.config.ts
│   ├── node_modules/         # npm install per session
│   └── src/
│       ├── DynamicComposition.tsx
│       └── components/
├── output/                   # New: rendered videos output
│   └── *.mp4
└── MIX.md                    # Existing session file
```

#### Bash-Based Implementation

Instead of complex Go backend tools and API endpoints, Remotion video creation is handled through AI tools defined in `go_backend/internal/llm/tools/descriptions/remotion.md`. These tools use bash commands to:

1. **Setup Remotion Projects**: Clone template repository during session creation using bash scripts
2. **Export Videos**: Execute `npx remotion render` commands within session workspaces
3. **Session Isolation**: All operations occur within session workspace boundaries
4. **File Management**: Videos exported to `{session_workspace}/output/` directory

**Key Advantages**:
- **Simplicity**: No complex Go backend code or API endpoints required
- **Direct Control**: AI can execute bash commands directly for video operations
- **Session Safety**: All operations confined to session workspace directories  
- **Standard Toolchain**: Uses Remotion CLI as intended without abstraction layers

### 3. Project Setup Approach

#### Template Repository Approach
Instead of creating projects from scratch, the implementation uses a pre-configured template repository:

```bash
# Clone template repository to session workspace
git clone https://github.com/sarath-menon/remotion-template-dynamic.git remotion_project
```

This approach provides:
- **Consistent Setup**: Pre-configured project with all necessary files
- **Version Control**: Template updates can be managed via Git
- **Instant Availability**: No need to run CLI setup during session creation
- **Custom Configuration**: Already includes dynamic composition and high-quality settings

#### Pre-Configured Features
The template repository includes:
1. **High-Quality Config**: `remotion.config.ts` configured with CRF 18 for broadcast quality
2. **Dynamic Composition**: Pre-built `DynamicVideoComposition` component with animation support
3. **JSON Props Support**: Ready for dynamic video generation from AI-provided configurations
4. **Dependencies**: All required packages pre-defined in package.json

## AI Integration

### AI Tools Integration

Remotion video creation is handled through AI tools defined in `go_backend/internal/llm/tools/descriptions/remotion.md` and included in the system prompt. This file contains two main tools:

#### 1. create_video_config
- **Purpose**: Generate JSON configuration for animated video content
- **Output**: Remotion configuration that can be previewed in frontend  
- **Parameters**: composition settings, animated elements, styling, animations
- **Usage**: "Create a video showing text animation with fade-in effect"

#### 2. export_video  
- **Purpose**: Export high-quality MP4 using Remotion CLI via bash commands
- **Implementation**: Uses pre-existing Remotion project in session workspace to render video
- **Output**: MP4 file in `{session_workspace}/output/` directory
- **Usage**: "Export this video to MP4"

### How It Works

1. **User Request**: "Create an animated title video"
2. **AI Response**: Uses `create_video_config` tool to generate Remotion JSON configuration 
3. **Frontend Preview**: Displays live preview using `@remotion/player`
4. **User Export Request**: "Export this video" 
5. **AI Export**: Uses `export_video` tool with bash commands to render MP4 to output/ directory in session workspace

**Key Benefits**:
- **No Backend Complexity**: No Go tools or API endpoints required
- **Direct Control**: AI executes bash commands directly within session workspace
- **Session Isolation**: All video files contained within session boundaries
- **Standard Toolchain**: Uses Remotion CLI without abstraction layers

## Pros and Cons

### Pros ✅

#### Technical Advantages
- **High Performance**: Client-side preview is instant, server-side export is high-quality
- **Full Remotion Features**: Access to all Remotion capabilities (effects, transitions, etc.)
- **Proven Toolchain**: Uses Remotion CLI as intended, battle-tested rendering
- **Codec Support**: Full support for H.264, H.265, WebM, and all formats
- **Quality Control**: Fine-grained control over bitrate, CRF, and encoding settings
- **Scalable**: Server can handle multiple concurrent exports

#### User Experience
- **Live Preview**: Immediate feedback with full playback controls
- **Professional Output**: Broadcast-quality title animations
- **Progress Tracking**: Can implement real-time render progress updates
- **Flexible Export**: Multiple formats and quality settings
- **Offline Capable**: Preview works without network (after initial load)

#### Development Benefits
- **Code Reuse**: Same component logic on frontend and backend
- **Type Safety**: Shared TypeScript types between preview and export
- **Maintainable**: Clean separation of concerns
- **Extensible**: Easy to add new element types and animations
- **Debuggable**: Standard React debugging for preview, CLI logs for export

### Cons ❌

#### Infrastructure Requirements
- **Backend Complexity**: Requires Node.js and Remotion setup on server
- **Resource Usage**: Title rendering is CPU/memory intensive
- **Storage Management**: Need to manage temporary files and cleanup
- **Dependencies**: Additional npm packages and potential version conflicts

#### Performance Considerations
- **Export Time**: Server-side rendering takes time (30-120 seconds typically)
- **Concurrent Limits**: Server can handle limited simultaneous exports
- **Memory Usage**: Large compositions can consume significant RAM
- **Disk Space**: Temporary files and output title animations require storage

#### Operational Challenges
- **Process Management**: Need to handle hanging/failed render processes
- **Error Handling**: Complex error scenarios (timeout, out of memory, etc.)
- **Security**: File system access and process execution concerns
- **Monitoring**: Need to track render queue and system resources

## Alternative Approaches Considered

### Pure Client-Side Title Rendering
- **Pros**: No backend complexity, instant title exports
- **Cons**: Limited typography quality, browser font rendering issues, limited animation capabilities

### Pre-built Title Templates
- **Pros**: Faster rendering, consistent branding
- **Cons**: Less customization, limited to predefined styles

### Cloud Title Rendering Service
- **Pros**: Scalable, professional typography engines
- **Cons**: External dependency, latency, cost per title generation

## Implementation Timeline

### Phase 1: Basic Video Integration (1-2 weeks)
- [ ] Install Remotion packages in frontend
- [x] Create template Remotion project structure (in separate repository)
- [x] Create DynamicVideoComposition component (in template)
- [ ] Implement JSON-based element rendering in frontend
- [ ] Add video preview to chat messages
- [ ] Create core animation library (text, shapes, basic effects)

### Phase 2: Session-Based Export Functionality (1 week)
- [x] Implement bash script for per-session setup
- [x] Create template repository cloning mechanism
- [x] Implement session-aware Remotion project initialization
- [ ] Add export button and progress UI for videos
- [ ] Handle session-specific video file downloads
- [x] Create output/ directory management

### Phase 3: AI Video Tool Integration (1 week)
- [x] Add create_video_config and export_video tools to AI system
- [x] Update tool descriptions for video generation scenarios
- [ ] Test AI-generated video scenarios end-to-end
- [ ] Refine element types and animations based on testing
- [ ] Test session isolation and cleanup in production scenarios

### Phase 4: Polish & Basic Optimization (1 week)
- [ ] Error handling and recovery mechanisms
- [ ] Session cleanup and storage management
- [ ] Basic performance optimization
- [ ] Documentation and testing

## Security Considerations

### Session Isolation
- Each session gets isolated Remotion project within its workspace
- File operations are confined to session working directory
- Leverage existing session directory permission system
- Exports are contained within session-specific output/ folder

### File System Security
- Validate all file paths are within session workspace boundaries
- Use session working directory as security boundary
- Follow existing input/ directory patterns for media access
- Automatic cleanup through session lifecycle management
- Prevent directory traversal beyond session workspace

### Process Security
- Run Remotion CLI with limited permissions within session context
- Implement timeout mechanisms for hanging render processes
- Monitor resource usage per session and implement limits
- Isolate npm dependencies per session project

### Input Validation
- Sanitize all JSON configurations before writing to session
- Validate element properties and values against schema
- Prevent code injection in dynamic props and file paths
- Validate session ID exists and user has access
- Restrict file access to session workspace and input/ directories

## Monitoring and Maintenance

### Key Metrics
- Export success/failure rates
- Basic error logging

### Maintenance Tasks
- Cleanup of output/ directories
- Update template repository dependencies

## Conclusion

The session-based hybrid approach provides the best balance of immediate video preview feedback with professional-quality output, while maintaining strong isolation between user sessions. By creating Remotion projects within each session's workspace directory, we leverage the existing robust session management system and ensure clean separation of user content.

**Key Benefits of Session-Based Approach:**
- **Strong Isolation**: Each session has its own Remotion environment
- **Leverages Existing Infrastructure**: Uses established session workspace patterns
- **Clean Organization**: Videos and output files are contained within session context
- **Security**: Builds on proven session directory permission system
- **Extensibility**: Easy to add session-specific customizations and media access

The implementation complexity is justified by the significant advantages in video quality, session isolation, and extensibility for various animation types. The phased approach allows for incremental development and testing of each video generation component while maintaining the robustness of the existing session management system.