"""
Blender video export and rendering utilities.
"""

from typing import Dict, Any, Optional, Tuple
import os


def export_video(
    output_path: str,
    frame_start: Optional[int] = None,
    frame_end: Optional[int] = None,
    resolution: Tuple[int, int] = (1920, 1080),
    fps: int = 24,
    video_format: str = 'MPEG4',
    codec: str = 'H264',
    quality: str = 'HIGH'
) -> Dict[str, Any]:
    """
    Export timeline sequences to a video file.
    
    Args:
        output_path: Path for the output video file
        frame_start: Starting frame (auto-detected from sequences if None)
        frame_end: Ending frame (auto-detected from sequences if None)
        resolution: Video resolution as (width, height) tuple, default: (1920, 1080)
        fps: Frames per second, default: 24
        video_format: Video container format ('MPEG4', 'AVI', 'QUICKTIME', 'WEBM'), default: 'MPEG4'
        codec: Video codec ('H264', 'XVID', 'THEORA', 'VP9'), default: 'H264'
        quality: Quality preset ('LOW', 'MEDIUM', 'HIGH', 'LOSSLESS'), default: 'HIGH'
        
    Returns:
        Dict[str, Any]: Dictionary with export information:
            - output_path: str - Full path to exported video
            - frame_start: int - Start frame of export
            - frame_end: int - End frame of export
            - duration: int - Duration in frames
            - resolution: Tuple[int, int] - Video resolution
            - fps: int - Frames per second
            - file_size: int - File size in bytes (if successful)
            - success: bool - Whether export completed successfully
            
    Raises:
        ImportError: If bpy module is not available
        ValueError: If parameters are invalid
        FileNotFoundError: If output directory doesn't exist
        AttributeError: If sequence editor is not available
    """
    import bpy
    
    scene = bpy.context.scene
    
    # Validate output path
    output_dir = os.path.dirname(output_path)
    if not os.path.exists(output_dir):
        raise FileNotFoundError(f"Output directory does not exist: {output_dir}")
    
    # Validate parameters
    if len(resolution) != 2 or resolution[0] <= 0 or resolution[1] <= 0:
        raise ValueError(f"Resolution must be positive (width, height) tuple, got: {resolution}")
    if fps <= 0:
        raise ValueError(f"FPS must be positive, got: {fps}")
    
    # Validate format and codec combinations
    valid_formats = {'MPEG4', 'AVI', 'QUICKTIME', 'WEBM'}
    if video_format not in valid_formats:
        raise ValueError(f"Invalid video format: {video_format}. Must be one of {valid_formats}")
    
    valid_codecs = {'H264', 'XVID', 'THEORA', 'VP9'}
    if codec not in valid_codecs:
        raise ValueError(f"Invalid codec: {codec}. Must be one of {valid_codecs}")
    
    valid_qualities = {'LOW', 'MEDIUM', 'HIGH', 'LOSSLESS'}
    if quality not in valid_qualities:
        raise ValueError(f"Invalid quality: {quality}. Must be one of {valid_qualities}")
    
    # Auto-detect frame range from sequences if not provided
    if frame_start is None or frame_end is None:
        if not scene.sequence_editor or not scene.sequence_editor.sequences_all:
            raise AttributeError("No sequences found - cannot auto-detect frame range")
        
        sequences = scene.sequence_editor.sequences_all
        auto_start = min(seq.frame_start for seq in sequences)
        auto_end = max(seq.frame_final_end for seq in sequences)
        
        if frame_start is None:
            frame_start = int(auto_start)
        if frame_end is None:
            frame_end = int(auto_end)
    
    # Validate frame range
    if frame_start >= frame_end:
        raise ValueError(f"Frame start ({frame_start}) must be less than frame end ({frame_end})")
    
    # Configure render settings
    render = scene.render
    
    # Set output path and format
    render.filepath = output_path
    render.image_settings.file_format = video_format
    
    # Set resolution and frame rate
    render.resolution_x = resolution[0]
    render.resolution_y = resolution[1]
    render.fps = fps
    
    # Set frame range
    scene.frame_start = frame_start
    scene.frame_end = frame_end
    
    # Configure FFmpeg settings for video output
    if video_format in ['MPEG4', 'AVI', 'QUICKTIME', 'WEBM']:
        render.image_settings.file_format = 'FFMPEG'
        ffmpeg = render.ffmpeg
        
        # Set container format
        if video_format == 'MPEG4':
            ffmpeg.format = 'MPEG4'
        elif video_format == 'AVI':
            ffmpeg.format = 'AVI'
        elif video_format == 'QUICKTIME':
            ffmpeg.format = 'QUICKTIME'
        elif video_format == 'WEBM':
            ffmpeg.format = 'WEBM'
        
        # Set codec
        if codec == 'H264':
            ffmpeg.codec = 'H264'
        elif codec == 'XVID':
            ffmpeg.codec = 'XVID'
        elif codec == 'THEORA':
            ffmpeg.codec = 'THEORA'
        elif codec == 'VP9':
            ffmpeg.codec = 'VP9'
        
        # Set quality settings
        if quality == 'LOW':
            ffmpeg.constant_rate_factor = 'HIGH'
            ffmpeg.gopsize = 18
        elif quality == 'MEDIUM':
            ffmpeg.constant_rate_factor = 'MEDIUM'
            ffmpeg.gopsize = 12
        elif quality == 'HIGH':
            ffmpeg.constant_rate_factor = 'LOW'
            ffmpeg.gopsize = 6
        elif quality == 'LOSSLESS':
            ffmpeg.constant_rate_factor = 'LOSSLESS'
            ffmpeg.gopsize = 1
    
    # Start the render
    try:
        bpy.ops.render.render(animation=True, write_still=True)
        success = True
    except Exception as e:
        raise AttributeError(f"Render failed: {str(e)}")
    
    # Get file size if export was successful
    file_size = 0
    if success and os.path.exists(output_path):
        file_size = os.path.getsize(output_path)
    
    # Return export information
    return {
        "output_path": output_path,
        "frame_start": frame_start,
        "frame_end": frame_end,
        "duration": frame_end - frame_start + 1,
        "resolution": resolution,
        "fps": fps,
        "file_size": file_size,
        "success": success
    }


def capture_preview_frame(
    output_path: str,
    frame: Optional[int] = None,
    resolution: Tuple[int, int] = (1920, 1080),
    format: str = 'PNG',
    quality: int = 90
) -> Dict[str, Any]:
    """
    Capture a screenshot of the current preview in the video editor.
    
    Args:
        output_path: Path for the output image file
        frame: Frame number to capture (uses current frame if None)
        resolution: Image resolution as (width, height) tuple, default: (1920, 1080)
        format: Image format ('PNG', 'JPEG', 'TIFF', 'BMP', 'TARGA'), default: 'PNG'
        quality: Image quality for JPEG format (1-100), default: 90
        
    Returns:
        Dict[str, Any]: Dictionary with capture information:
            - output_path: str - Full path to captured image
            - frame: int - Frame that was captured
            - resolution: Tuple[int, int] - Image resolution
            - format: str - Image format used
            - file_size: int - File size in bytes (if successful)
            - success: bool - Whether capture completed successfully
            
    Raises:
        ImportError: If bpy module is not available
        ValueError: If parameters are invalid
        FileNotFoundError: If output directory doesn't exist
        AttributeError: If sequence editor is not available
    """
    import bpy
    
    scene = bpy.context.scene
    
    # Validate output path
    output_dir = os.path.dirname(output_path)
    if not os.path.exists(output_dir):
        raise FileNotFoundError(f"Output directory does not exist: {output_dir}")
    
    # Validate parameters
    if len(resolution) != 2 or resolution[0] <= 0 or resolution[1] <= 0:
        raise ValueError(f"Resolution must be positive (width, height) tuple, got: {resolution}")
    
    # Validate image format
    valid_formats = {'PNG', 'JPEG', 'TIFF', 'BMP', 'TARGA'}
    if format not in valid_formats:
        raise ValueError(f"Invalid image format: {format}. Must be one of {valid_formats}")
    
    # Validate quality for JPEG
    if format == 'JPEG' and (quality < 1 or quality > 100):
        raise ValueError(f"JPEG quality must be between 1-100, got: {quality}")
    
    # Set frame if specified, otherwise use current frame
    if frame is not None:
        if frame < 0:
            raise ValueError(f"Frame number must be non-negative, got: {frame}")
        scene.frame_set(frame)
        capture_frame = frame
    else:
        capture_frame = scene.frame_current
    
    # Store original render settings to restore later
    original_filepath = scene.render.filepath
    original_format = scene.render.image_settings.file_format
    original_color_mode = scene.render.image_settings.color_mode
    original_quality = scene.render.image_settings.quality
    original_res_x = scene.render.resolution_x
    original_res_y = scene.render.resolution_y
    
    try:
        # Configure render settings for screenshot
        render = scene.render
        
        # Set output path and format
        render.filepath = output_path
        render.image_settings.file_format = format
        
        # Set resolution
        render.resolution_x = resolution[0]
        render.resolution_y = resolution[1]
        
        # Configure format-specific settings
        if format == 'PNG':
            render.image_settings.color_mode = 'RGBA'
            render.image_settings.compression = 15  # PNG compression level
        elif format == 'JPEG':
            render.image_settings.color_mode = 'RGB'
            render.image_settings.quality = quality
        elif format in ['TIFF', 'BMP', 'TARGA']:
            render.image_settings.color_mode = 'RGBA'
        
        # Ensure we're in the sequence editor context for preview capture
        if not scene.sequence_editor:
            raise AttributeError("Sequence editor is not available - cannot capture preview")
        
        # Set context to sequence editor for preview rendering
        with bpy.context.temp_override(
            window=bpy.context.window,
            screen=bpy.context.screen,
            area=[area for area in bpy.context.screen.areas if area.type == 'SEQUENCE_EDITOR'][0] if any(area.type == 'SEQUENCE_EDITOR' for area in bpy.context.screen.areas) else bpy.context.area,
            region=[region for region in bpy.context.area.regions if region.type == 'WINDOW'][0] if bpy.context.area and any(region.type == 'WINDOW' for region in bpy.context.area.regions) else bpy.context.region
        ):
            # Render single frame (preview capture)
            bpy.ops.render.render(write_still=True)
        
        success = True
        
    except Exception as e:
        raise AttributeError(f"Preview capture failed: {str(e)}")
    
    finally:
        # Restore original render settings
        scene.render.filepath = original_filepath
        scene.render.image_settings.file_format = original_format
        scene.render.image_settings.color_mode = original_color_mode
        scene.render.image_settings.quality = original_quality
        scene.render.resolution_x = original_res_x
        scene.render.resolution_y = original_res_y
    
    # Get file size if capture was successful
    file_size = 0
    if success and os.path.exists(output_path):
        file_size = os.path.getsize(output_path)
    
    # Return capture information
    return {
        "output_path": output_path,
        "frame": capture_frame,
        "resolution": resolution,
        "format": format,
        "file_size": file_size,
        "success": success
    }