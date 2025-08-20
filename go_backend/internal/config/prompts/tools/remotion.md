# export_video

Export Remotion videos using the HTTP API for flexible and efficient video generation.

## Usage notes

- Use HTTP requests to export videos with full configuration control
- Do not use this tool unless the user explicitly asks you to export a remotion video
- The endpoint is session-agnostic - specify output path directly
- Use the bash tool to make these CURL requests

## HTTP Endpoint

**POST** `/api/video/export`

Request Body (JSON)

<request><--json_body--></request>

Response (Success - HTTP 200)

<response>{
  "success": true,
  "outputPath": "/path/to/output/video.mp4",
  "message": "Video exported successfully"
}</response>

Response (Error - HTTP 4xx/5xx)

<response>{
  "error": "error_code",
  "message": "Detailed error message"
}</response>

## Parameters

- `config` (object, required): Remotion video configuration
  - `composition` (object): Video settings (duration, fps, format)
  - `elements` (array): Video elements (text, images, animations)
- `outputPath` (string, required): Full path where video should be saved
- `sessionId` (string, optional): Associate export with specific session

## CURL Examples

### Video Export

<bash>curl -X POST http://localhost:8088/api/video/export \
  -H "Content-Type: application/json" \
  -d '<--json_body-->'</bash>

### Check for Success

<bash># Store response and check status
response=$(curl -s -X POST <http://localhost:8088/api/video/export> \
  -H "Content-Type: application/json" \
  -d '<--json_body-->')

echo "$response" | jq .success  # Should return true
echo "$response" | jq .outputPath  # Should return the output path</bash>

## Error Handling

Common error codes:

- `missing_config`: Required config parameter missing
- `missing_output_path`: Required outputPath parameter missing
- `invalid_json`: Malformed JSON in request body
- `invalid_config`: Config format is invalid
- `export_failed`: Video export process failed

## Notes

- Default resolution: Vertical (1080x1920) or Horizontal (1920x1080) based on config format
- Videos export as MP4 with H.264 codec, high quality settings
- All video processing uses the shared, up-to-date Remotion codebase
- Temporary config files are automatically cleaned up after export
- Session ID is optional - useful for organizing exports by session
