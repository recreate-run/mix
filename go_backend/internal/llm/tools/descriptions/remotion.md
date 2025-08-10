# export_video

Export Remotion videos using single-step bash script execution within the session workspace.

## Usage notes

- Execute single make command that handles config writing, video rendering, and file organization
- Do not use this tool unless the user explicitly asks you to export a remotion video

```bash
# Single make command execution:
cd $<workdir>/remotion_project && make export_video CONFIG_JSON='<config_json>' OUTPUT='<filename>'
# Video output: $<workdir>/output/<filename>_<timestamp>.mp4
```

## Parameters

- `config` (object, required): Remotion video configuration with composition settings and elements
- `filename` (string, optional): Custom output filename (defaults to timestamp-based naming)


## Notes

- Config files use timestamp naming: `config_1641234567.json`
- All paths are relative to `$<workdir>`
- Videos export as MP4 with H.264 codec, CRF 18 quality
- Default resolution: 1920x1080 at 30fps