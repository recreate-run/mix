# Remotion Video Creation Tools
These tools enable AI-powered animated video creation using Remotion within session workspaces.

## Instructions

- All video creation happens within the current session workspace directory
- Videos are exported to `{session_workspace}/output/` directory  
- Use absolute file paths when working with session directories
- The export_video tool uses bash commands to render videos in pre-existing Remotion projects
- Each session has its own isolated Remotion project already set up in `{session_workspace}/remotion_project/`
- Videos are rendered as MP4 files with timestamp-based naming
- Remotion project is automatically cloned and configured during session creation

## Tools

### export_video
Export video using Remotion CLI through bash commands within the session workspace.

**Purpose**: Write video configuration and render final MP4 video using pre-existing Remotion project.

**Implementation**: Use bash commands to execute the video export process:

```bash
# Get current session workspace directory (provided by session context)
WORKSPACE_DIR="$PWD"  # Current working directory should be session workspace
REMOTION_PROJECT_DIR="$WORKSPACE_DIR/remotion_project"
OUTPUT_DIR="$WORKSPACE_DIR/output"
TIMESTAMP=$(date +%s)
OUTPUT_FILE="video_$TIMESTAMP.mp4"

# Ensure output directory exists
mkdir -p "$OUTPUT_DIR"

# Write video configuration to temporary file
CONFIG_FILE="$REMOTION_PROJECT_DIR/temp_config.json"
echo "Writing video configuration..."

# Note: The actual config JSON should be provided by the tool execution context
# This is a placeholder - the actual implementation would write the config passed to the tool
cat > "$CONFIG_FILE" << 'EOF_CONFIG'
{
  "config": {
    "composition": {
      "durationInFrames": 150,
      "fps": 30,
      "width": 1920,
      "height": 1080
    },
    "elements": []
  }
}
EOF_CONFIG

# Export the video
echo "Rendering video..."
cd "$REMOTION_PROJECT_DIR"

npx remotion render DynamicComposition "$OUTPUT_DIR/$OUTPUT_FILE" --props="$CONFIG_FILE"

cd "$WORKSPACE_DIR"

# Clean up temporary config file
rm "$CONFIG_FILE"

echo "Video exported successfully: $OUTPUT_DIR/$OUTPUT_FILE"
```

**Parameters**:
- `config` (object): The same Remotion video configuration from create_video_config
- `filename` (string, optional): Custom filename for the exported video (defaults to timestamp-based name)

**Notes**:
- The Remotion project is pre-configured during session creation with all dependencies installed
- Videos are exported to the session's output/ directory
- Configuration is written to a temporary JSON file and passed to Remotion CLI
- All operations are contained within the session workspace for security
- Each session has an isolated Remotion project cloned from the template repository

## Example Workflow

1. **Create video configuration**:
   ```
   Use create_video_config with composition settings and animated elements
   ```

2. **Preview in frontend**:
   ```
   Frontend displays live preview using @remotion/player with the configuration
   ```

3. **Export final video**:
   ```
   Use export_video tool which executes bash commands to render high-quality MP4
   ```

## Video Formats

- Output: MP4 with H.264 codec
- Quality: High quality (CRF 18)
- Resolution: Configurable (default 1920x1080)
- Frame rate: Configurable (default 30fps)