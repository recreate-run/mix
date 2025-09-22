# Gemini Multimodal Analyzer Tool Implementation Plan

## Overview

Create a new Gemini-based multimodal analyzer tool that leverages Google's Gemini provider for AI-powered analysis of images, audio, and video files. This tool will be completely independent of the existing multimodal analyzer CLI tool and will be the first in the codebase to internally use an LLM provider for processing, making it a hybrid tool combining file system operations with AI analysis.

## Current State Analysis

### Existing Infrastructure

- **Separate CLI Tool**: An existing multimodal analyzer CLI tool exists but this new tool will be completely independent
- **No Agent Integration**: No existing Go implementation for multimodal analysis within the agent tool system
- **Provider Support**: Gemini provider exists with full multimodal support in `/mix_agent/internal/llm/provider/gemini.go`
- **Tool Registration**: Tools are registered in `/mix_agent/internal/llm/agent/tools.go` at lines 25-40

### Key Reference Points

- **Agent Interface**: `/mix_agent/internal/llm/agent/agent.go` lines 62-74 define Service interface
- **Provider Creation**: `/mix_agent/internal/llm/agent/agent.go` lines 1058-1145 show provider creation patterns
- **Tool Patterns**: Existing tools in `/mix_agent/internal/llm/tools/` directory follow consistent patterns
- **Binary Content**: Message system supports attachments via BinaryContent struct

## Implementation Plan

### Phase 1: Tool Structure and Interface

#### 1.1 Create Core Tool File

**Location**: `/mix_agent/internal/llm/tools/gemini_multimodal_analyzer.go`

**Structure**:

- Implement BaseTool interface following patterns from existing tools
- Add dependencies: permissions service, provider creation capability
- Include configuration for Gemini-specific options
- Support for file path validation and MIME type detection
- **Independent Implementation**: No code sharing or dependencies on existing multimodal CLI tool

#### 1.2 Tool Information Definition

**Reference**: Similar to existing tools' Info() method patterns

**Requirements**:

- Create new tool description specific to Gemini integration
- Define JSON schema parameters for agent tool interface
- Support for file paths, analysis types (image/audio/video), prompts
- Required fields: file path or directory, analysis type, prompt
- **No CLI Interface**: Tool designed purely for agent integration, not CLI usage

#### 1.3 Permission Integration

**Reference**: `/mix_agent/internal/llm/agent/agent.go` lines 591-608 show permission handling patterns

**Implementation**:

- Request file read permissions for target media files
- Validate user consent for AI analysis of potentially sensitive content
- Handle permission denied scenarios gracefully

### Phase 2: Provider Integration

#### 2.1 Gemini Provider Instantiation

**Reference**: `/mix_agent/internal/llm/agent/agent.go` lines 1058-1145 for provider creation patterns

**Approach**:

- Create dedicated Gemini provider instance within tool
- Use existing createAgentProvider patterns but with Gemini-specific configuration
- Implement proper API key management through existing credential system
- Configure for multimodal content processing

#### 2.2 Content Preparation

**Reference**: Message attachment handling patterns from agent conversations

**Implementation**:

- Convert file paths to BinaryContent structures with appropriate MIME types
- Handle file size limitations and preprocessing (as described in documentation)
- Support batch processing for multiple files
- Implement image optimization for files >500KB as documented

#### 2.3 Provider Communication

**Pattern**: Follow agent message processing from `/mix_agent/internal/llm/agent/agent.go` lines 425-643

**Design**:

- Construct appropriate message format for Gemini API
- Include custom prompts or use default analysis prompts
- Handle streaming responses if supported
- Process provider responses and extract analysis results

### Phase 3: File System Integration

#### 3.1 File Discovery and Validation

**Requirements from Documentation**:

- Support single file analysis
- Support directory-based batch processing
- Implement recursive directory scanning
- Handle multiple file types (images, audio, video)

#### 3.2 MIME Type Detection

**Integration Point**: Leverage existing file handling patterns from view tool

**Implementation**:

- Automatic MIME type detection for proper provider formatting
- File extension validation against supported formats
- Error handling for unsupported or corrupted files

#### 3.3 Batch Processing Logic

**Capability**: Process multiple files with progress tracking and concurrency control

**Design**:

- Implement concurrent file processing with configurable limits
- Progress reporting through tool response metadata
- Error aggregation for failed analyses

### Phase 4: Response Formatting

#### 4.1 Output Format Support

**Documentation Requirements**: Support JSON format only

**Implementation**:

- Structured JSON response objects for programmatic access
- Metadata inclusion (file info, processing time, model used)
- Consistent format for both single file and batch processing

#### 4.2 Error Handling

**Pattern**: Follow tool error patterns from existing implementations

**Coverage**:

- File access errors
- Provider communication failures
- Analysis processing errors
- Partial batch processing failures with detailed reporting

### Phase 5: Tool Registration and Testing

#### 5.1 Agent Integration

**Location**: `/mix_agent/internal/llm/agent/tools.go` lines 25-40

**Changes Required**:

- Add Gemini multimodal analyzer tool to CoderAgentTools array
- Ensure proper dependency injection (permissions service)
- Maintain tool ordering and categorization

#### 5.2 Plan Mode Compatibility

**Reference**: `/mix_agent/internal/llm/agent/agent.go` lines 1033-1049 for plan mode tool filtering

**Consideration**:

- Determine if tool should be available in plan mode
- Likely should NOT be available in plan mode due to AI processing requirements
- Update isToolAllowedInPlanMode function if needed

## Technical Considerations

### Authentication and API Keys

- Leverage existing credential management system from `/mix_agent/internal/llm/agent/agent.go` lines 1092-1105
- Use database-stored API keys only, no environment fallbacks
- Handle OAuth scenarios for Gemini API access

### Performance Optimization

- Implement file caching for repeated analyses
- Use appropriate Gemini model selection based on content type
- Optimize concurrent processing to avoid rate limits

### Security and Privacy

- Ensure no sensitive file content is logged
- Respect user permission boundaries
- Handle temporary file cleanup for preprocessed content

### Error Recovery

- Graceful degradation for network issues
- Retry logic for transient provider failures
- Clear error messages for user actionable issues

## Tool Parameter Schema

### JSON Schema Definition

```json
{
  "type": "object",
  "properties": {
    "file_path": {
      "type": "string",
      "description": "Path to single file for analysis"
    },
    "directory_path": {
      "type": "string",
      "description": "Path to directory for batch processing"
    },
    "analysis_type": {
      "type": "string",
      "enum": ["image", "audio", "video"],
      "description": "Type of media analysis to perform"
    },
    "prompt": {
      "type": "string",
      "description": "Analysis prompt for the media content"
    },
    "recursive": {
      "type": "boolean",
      "default": false,
      "description": "Process directories recursively"
    },
    "word_count": {
      "type": "integer",
      "minimum": 50,
      "maximum": 1000,
      "default": 200,
      "description": "Target word count for analysis"
    },
    "audio_mode": {
      "type": "string",
      "enum": ["transcript", "description"],
      "description": "Audio analysis mode (required for audio type)"
    },
    "video_mode": {
      "type": "string",
      "enum": ["description"],
      "description": "Video analysis mode (required for video type)"
    }
  },
  "required": ["analysis_type", "prompt"],
  "oneOf": [
    {"required": ["file_path"]},
    {"required": ["directory_path"]}
  ]
}
```

### Parameter Validation Rules

- **Mutually Exclusive**: Cannot specify both `file_path` and `directory_path`
- **Type-Specific Requirements**: `audio_mode` required when `analysis_type` is "audio"
- **Type-Specific Requirements**: `video_mode` required when `analysis_type` is "video"
- **Path Validation**: All paths must be absolute and accessible
- **File Extension Validation**: File extensions must match specified `analysis_type`

### Default Prompts by Analysis Type

- **Image**: "Analyze this image and describe what you see in detail"
- **Audio (transcript)**: "Provide an accurate transcription of this audio"
- **Audio (description)**: "Describe the audio content, including speakers, topics, and key points"
- **Video**: "Analyze this video content, describing both visual and audio elements"

## Implementation Priority

**High Priority**:

- Core tool implementation with single file analysis
- Basic Gemini provider integration
- Permission system integration

**Medium Priority**:

- Batch processing capabilities
- Comprehensive error handling
- Performance optimization for concurrent processing

**Low Priority**:

- Advanced model selection options
- Performance optimizations
- Extended file format support

This implementation will establish the foundation for AI-powered multimodal analysis within the Mix agent system while following established architectural patterns and maintaining security best practices.
