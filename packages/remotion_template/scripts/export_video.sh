#!/bin/bash

set -e

# Parse command line arguments
CONFIG_FILE=""
OUTPUT_NAME=""

while [[ $# -gt 0 ]]; do
    case $1 in
        --config) CONFIG_FILE="$2"; shift 2 ;;
        --output) OUTPUT_NAME="$2"; shift 2 ;;
        *) echo "Usage: $0 [--config <config_file>] --output <output_name>"; exit 1 ;;
    esac
done

# Validate output name
if [[ -z "$OUTPUT_NAME" ]]; then
    echo "Usage: $0 [--config <config_file>] --output <output_name>"
    echo "Config can be provided via --config file or stdin"
    exit 1
fi

# Read config from file or stdin
if [[ -n "$CONFIG_FILE" ]]; then
    if [[ ! -f "$CONFIG_FILE" ]]; then
        echo "❌ Error: Config file not found: $CONFIG_FILE"
        exit 1
    fi
    echo "🎬 Exporting $OUTPUT_NAME from $CONFIG_FILE"
    CONFIG_JSON=$(cat "$CONFIG_FILE")
else
    echo "🎬 Exporting $OUTPUT_NAME from stdin"
    CONFIG_JSON=$(cat)
fi

# Ensure output directory exists
mkdir -p output

# Validate JSON format first
echo "$CONFIG_JSON" | node -e "
    try {
        JSON.parse(require('fs').readFileSync(0, 'utf8'));
    } catch (error) {
        console.error('❌ Error: Invalid JSON in config file');
        console.error('Details:', error.message);
        process.exit(1);
    }
" || exit 1

# Extract format from config to determine dimensions
FORMAT=$(echo "$CONFIG_JSON" | node -e "
    const config = JSON.parse(require('fs').readFileSync(0, 'utf8'));
    const format = config?.composition?.format;
    if (format === 'vertical' || format === 'horizontal') {
        console.log(format);
    } else {
        console.error('⚠️  Warning: Invalid or missing format in config, defaulting to horizontal');
        console.log('horizontal');
    }
")

# Validate format value was extracted
if [[ -z "$FORMAT" ]]; then
    echo "❌ Error: Could not determine video format from config file"
    echo "Please ensure $CONFIG_FILE contains: {\"composition\": {\"format\": \"vertical\" | \"horizontal\"}}"
    exit 1
fi

# Set dimensions based on format
if [[ "$FORMAT" == "vertical" ]]; then
    WIDTH=1080
    HEIGHT=1920
else
    WIDTH=1920
    HEIGHT=1080
fi

echo "📐 Using format: $FORMAT (${WIDTH}x${HEIGHT})"

# Always use DynamicComposition with config as props
INPUT_PROPS="{\"config\": $CONFIG_JSON}"

# Render video with explicit dimensions
npx remotion render "DynamicComposition" "output/$OUTPUT_NAME.mp4" \
    --props="$INPUT_PROPS" \
    --width="$WIDTH" \
    --height="$HEIGHT"

echo "✅ Export completed: output/$OUTPUT_NAME.mp4"