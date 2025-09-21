"""
Blender sequencer and timeline utilities.

IMPORTANT INSTRUCTIONS:
1. In Blender, the video editor timeline is called "sequencer"
2. Video tracks should never have combined audio - audio must always be in separate tracks (tracks may be joined)
"""

from typing import List, Dict, Any, Optional, Tuple, Union
import os

__all__ = [
    'get_timeline_items', 'get_sequence_resize_info', 'add_image', 'add_video', 'add_audio',
    'add_transition', 'add_text', 'add_color', 'delete_timeline_item', 'fit_sequencer_view',
    'set_frame_range', 'get_frame_range', 'set_current_frame', 'get_current_frame',
    'modify_image', 'modify_video', 'modify_audio', 'duplicate_timeline_element',
    'blade_cut', 'detach_audio_from_video'
]


def get_timeline_items(channel: Optional[int] = None) -> List[Dict[str, Any]]:
    """
    Get timeline sequences, optionally filtered by channel.

    Args:
        channel: Optional channel number to filter by. If None, returns all sequences.

    Returns:
        List[Dict[str, Any]]: List of sequence dictionaries containing:
            - name: str - Sequence name
            - type: str - Sequence type (IMAGE, MOVIE, SOUND, etc.)
            - channel: int - Channel number
            - frame_start: float - Start frame
            - frame_end: float - End frame
            - duration: int - Duration in frames
            - filepath: Optional[str] - Full file path if available
            - original_resolution: Optional[Tuple[int, int]] - Original video resolution if applicable
            - transform: Dict[str, Any] - Transform properties including scale, offset, rotation
            - is_resized: bool - Whether the sequence has been resized from original

    Raises:
        ImportError: If bpy module is not available
        AttributeError: If sequence editor is not available
    """
    import bpy

    scene = bpy.context.scene

    if not scene.sequence_editor:
        raise AttributeError("No sequence editor found - timeline is empty or not initialized")

    sequences = scene.sequence_editor.sequences_all
    result = []

    for seq in sequences:
        if channel is None or seq.channel == channel:
            transform_data, original_res, is_resized = _get_sequence_transform_info(seq)
            
            sequence_data = {
                "name": seq.name,
                "type": seq.type,
                "channel": seq.channel,
                "frame_start": seq.frame_start,
                "frame_end": seq.frame_final_end,
                "duration": seq.frame_final_duration,
                "filepath": _get_sequence_filepath(seq),
                "original_resolution": original_res,
                "transform": transform_data,
                "is_resized": is_resized
            }
            result.append(sequence_data)

    return result


def _get_sequence_filepath(seq) -> Optional[str]:
    """
    Extract the file path from a sequence based on its type.

    Args:
        seq: Blender sequence object

    Returns:
        Optional[str]: Full file path if available, None otherwise
    """
    if seq.type == 'IMAGE':
        if hasattr(seq, 'directory') and hasattr(seq, 'elements') and seq.elements:
            directory = seq.directory
            filename = seq.elements[0].filename
            return os.path.join(directory, filename)
    elif seq.type == 'MOVIE':
        if hasattr(seq, 'filepath'):
            return seq.filepath
    elif seq.type == 'SOUND':
        if hasattr(seq, 'sound') and hasattr(seq.sound, 'filepath'):
            return seq.sound.filepath
    elif hasattr(seq, 'filepath'):
        return seq.filepath

    return None


def _get_sequence_transform_info(seq) -> Tuple[Dict[str, Any], Optional[Tuple[int, int]], bool]:
    """
    Extract transform information and determine if sequence has been resized.

    Args:
        seq: Blender sequence object

    Returns:
        Tuple containing:
            - Dict[str, Any]: Transform data with scale, offset, rotation
            - Optional[Tuple[int, int]]: Original resolution (width, height) if available
            - bool: Whether the sequence has been resized
    """
    transform_data = {
        "scale_x": 1.0,
        "scale_y": 1.0,
        "offset_x": 0.0,
        "offset_y": 0.0,
        "rotation": 0.0
    }
    
    original_resolution = None
    is_resized = False
    
    # Get transform properties if available
    if hasattr(seq, 'transform'):
        transform = seq.transform
        transform_data.update({
            "scale_x": getattr(transform, 'scale_x', 1.0),
            "scale_y": getattr(transform, 'scale_y', 1.0),
            "offset_x": getattr(transform, 'offset_x', 0.0),
            "offset_y": getattr(transform, 'offset_y', 0.0),
            "rotation": getattr(transform, 'rotation', 0.0)
        })
        
        # Check if scaling is not 1.0 (indicating resize)
        if transform_data["scale_x"] != 1.0 or transform_data["scale_y"] != 1.0:
            is_resized = True
    
    # Get original resolution for video/image sequences
    if seq.type == 'MOVIE' and hasattr(seq, 'elements') and seq.elements:
        try:
            # Get original video dimensions
            element = seq.elements[0]
            if hasattr(element, 'orig_width') and hasattr(element, 'orig_height'):
                original_resolution = (element.orig_width, element.orig_height)
        except (AttributeError, IndexError):
            pass
    elif seq.type == 'IMAGE' and hasattr(seq, 'elements') and seq.elements:
        try:
            # Get original image dimensions
            element = seq.elements[0]
            if hasattr(element, 'orig_width') and hasattr(element, 'orig_height'):
                original_resolution = (element.orig_width, element.orig_height)
        except (AttributeError, IndexError):
            pass
    
    return transform_data, original_resolution, is_resized


def get_sequence_resize_info(sequence_name: str) -> Dict[str, Any]:
    """
    Get detailed resize information for a specific sequence.

    Args:
        sequence_name: Name of the sequence to check

    Returns:
        Dict[str, Any]: Dictionary containing:
            - name: str - Sequence name
            - type: str - Sequence type
            - is_resized: bool - Whether the sequence has been resized
            - original_resolution: Optional[Tuple[int, int]] - Original dimensions
            - current_scale: Tuple[float, float] - Current scale factors (x, y)
            - effective_resolution: Optional[Tuple[int, int]] - Calculated current resolution
            - resize_method: str - How the resize was applied ('SCALE' or 'FIT_METHOD')
            - fit_method: Optional[str] - Fit method if used during import

    Raises:
        ImportError: If bpy module is not available
        ValueError: If sequence is not found
        AttributeError: If sequence editor is not available
    """
    import bpy

    scene = bpy.context.scene

    if not scene.sequence_editor:
        raise AttributeError("No sequence editor found - timeline is empty or not initialized")

    sequences = scene.sequence_editor.sequences_all
    
    # Find the sequence by name
    target_seq = None
    for seq in sequences:
        if seq.name == sequence_name:
            target_seq = seq
            break
    
    if target_seq is None:
        raise ValueError(f"Sequence '{sequence_name}' not found")

    transform_data, original_resolution, is_resized = _get_sequence_transform_info(target_seq)
    
    # Calculate effective resolution if we have original dimensions
    effective_resolution = None
    if original_resolution:
        effective_resolution = (
            int(original_resolution[0] * transform_data["scale_x"]),
            int(original_resolution[1] * transform_data["scale_y"])
        )
    
    # Determine resize method
    resize_method = "SCALE" if is_resized else "NONE"
    fit_method = getattr(target_seq, 'fit_method', None) if hasattr(target_seq, 'fit_method') else None
    
    return {
        "name": target_seq.name,
        "type": target_seq.type,
        "is_resized": is_resized,
        "original_resolution": original_resolution,
        "current_scale": (transform_data["scale_x"], transform_data["scale_y"]),
        "effective_resolution": effective_resolution,
        "resize_method": resize_method,
        "fit_method": fit_method,
        "transform": transform_data
    }


def add_image(
    filepath: str,
    channel: int,
    frame_start: int,
    name: Optional[str] = None,
    fit_method: str = 'ORIGINAL',
    frame_end: Optional[int] = None,
    position: Optional[Tuple[float, float]] = None,
    scale: Optional[float] = None
) -> Dict[str, Any]:
    """
    Add an image element to the timeline at a specific channel and position.

    Args:
        filepath: Path to the image file
        channel: Channel number to place the sequence
        frame_start: Starting frame position
        name: Optional sequence name (auto-generated from filename if None)
        fit_method: How to fit the image ('ORIGINAL', 'FIT', 'FILL', 'STRETCH')
        frame_end: Optional ending frame position. Images can be extended to any duration.
        position: Optional (x, y) offset in pixels from center
        scale: Optional uniform scale factor (1.0 = original size)

    Returns:
        Dict[str, Any]: Dictionary with sequence information matching get_timeline_items() format

    Raises:
        ImportError: If bpy module is not available
        FileNotFoundError: If filepath does not exist
        ValueError: If file is not a valid image format or parameters are invalid
        AttributeError: If sequence creation fails
    """
    import bpy

    # Validate file exists
    if not os.path.exists(filepath):
        raise FileNotFoundError(f"File not found: {filepath}")

    # Validate image file extension
    ext = os.path.splitext(filepath)[1].lower()
    image_extensions = {'.jpg', '.jpeg', '.png', '.tiff', '.tif', '.exr', '.hdr', '.bmp', '.tga'}
    if ext not in image_extensions:
        raise ValueError(f"Invalid image file extension: {ext}. Supported: {image_extensions}")

    scene = bpy.context.scene

    # Create sequence editor if it doesn't exist
    if not scene.sequence_editor:
        scene.sequence_editor_create()

    sequences = scene.sequence_editor.sequences

    # Generate name if not provided
    if name is None:
        name = os.path.splitext(os.path.basename(filepath))[0]

    # Create image sequence
    sequence = sequences.new_image(name=name, filepath=filepath, channel=channel, frame_start=frame_start, fit_method=fit_method)

    # Apply frame_end if specified (images can be extended to any duration)
    if frame_end is not None:
        requested_duration = frame_end - frame_start
        
        if requested_duration <= 0:
            raise ValueError(f"Frame end ({frame_end}) must be greater than frame start ({frame_start})")
        
        # Set the sequence end frame (extends image duration)
        sequence.frame_final_end = frame_end

    # Apply position and scale transforms if specified
    if position is not None or scale is not None:
        if hasattr(sequence, 'transform'):
            transform = sequence.transform
            if position is not None:
                transform.offset_x = position[0]
                transform.offset_y = position[1]
            if scale is not None:
                transform.scale_x = scale
                transform.scale_y = scale

    # Auto-fit sequencer view to show all sequences
    try:
        fit_sequencer_view()
    except:
        pass  # Silently ignore fit errors
    
    # Return sequence info in same format as get_timeline_items()
    return {
        "name": sequence.name,
        "type": sequence.type,
        "channel": sequence.channel,
        "frame_start": sequence.frame_start,
        "frame_end": sequence.frame_final_end,
        "duration": sequence.frame_final_duration,
        "filepath": _get_sequence_filepath(sequence)
    }


def add_video(
    filepath: str,
    channel: int,
    frame_start: int,
    name: Optional[str] = None,
    fit_method: str = 'ORIGINAL',
    frame_end: Optional[int] = None
) -> Dict[str, Any]:
    """
    Add a video element to the timeline at a specific channel and position.

    Args:
        filepath: Path to the video file
        channel: Channel number to place the sequence
        frame_start: Starting frame position
        name: Optional sequence name (auto-generated from filename if None)
        fit_method: How to fit the video ('ORIGINAL', 'FIT', 'FILL', 'STRETCH')
        frame_end: Optional ending frame position. Videos can only be trimmed, not extended.

    Returns:
        Dict[str, Any]: Dictionary with sequence information matching get_timeline_items() format

    Raises:
        ImportError: If bpy module is not available
        FileNotFoundError: If filepath does not exist
        ValueError: If file is not a valid video format or parameters are invalid
        AttributeError: If sequence creation fails
    """
    import bpy

    # Validate file exists
    if not os.path.exists(filepath):
        raise FileNotFoundError(f"File not found: {filepath}")

    # Validate video file extension
    ext = os.path.splitext(filepath)[1].lower()
    video_extensions = {'.mp4', '.mov', '.avi', '.mkv', '.webm', '.wmv', '.m4v', '.flv'}
    if ext not in video_extensions:
        raise ValueError(f"Invalid video file extension: {ext}. Supported: {video_extensions}")

    scene = bpy.context.scene

    # Create sequence editor if it doesn't exist
    if not scene.sequence_editor:
        scene.sequence_editor_create()

    sequences = scene.sequence_editor.sequences

    # Generate name if not provided
    if name is None:
        name = os.path.splitext(os.path.basename(filepath))[0]

    # Create video sequence
    sequence = sequences.new_movie(name=name, filepath=filepath, channel=channel, frame_start=frame_start, fit_method=fit_method)

    # Apply frame_end if specified (videos can only be trimmed, not extended)
    if frame_end is not None:
        original_duration = sequence.frame_final_duration
        requested_duration = frame_end - frame_start
        
        if requested_duration <= 0:
            raise ValueError(f"Frame end ({frame_end}) must be greater than frame start ({frame_start})")
        
        # Prevent extending beyond original duration
        if requested_duration > original_duration:
            raise ValueError(
                f"Cannot extend video beyond original duration. "
                f"Requested duration: {requested_duration} frames, "
                f"actual duration: {original_duration} frames"
            )
        
        # Set the sequence end frame (trims video)
        sequence.frame_final_end = frame_end

    # Auto-fit sequencer view to show all sequences
    try:
        fit_sequencer_view()
    except:
        pass  # Silently ignore fit errors
    
    # Return sequence info in same format as get_timeline_items()
    return {
        "name": sequence.name,
        "type": sequence.type,
        "channel": sequence.channel,
        "frame_start": sequence.frame_start,
        "frame_end": sequence.frame_final_end,
        "duration": sequence.frame_final_duration,
        "filepath": _get_sequence_filepath(sequence)
    }


def add_audio(
    filepath: str,
    channel: int,
    frame_start: int,
    name: Optional[str] = None,
    frame_end: Optional[int] = None
) -> Dict[str, Any]:
    """
    Add an audio element to the timeline at a specific channel and position.

    Args:
        filepath: Path to the audio file
        channel: Channel number to place the sequence
        frame_start: Starting frame position
        name: Optional sequence name (auto-generated from filename if None)
        frame_end: Optional ending frame position. Audio can only be trimmed, not extended.

    Returns:
        Dict[str, Any]: Dictionary with sequence information matching get_timeline_items() format

    Raises:
        ImportError: If bpy module is not available
        FileNotFoundError: If filepath does not exist
        ValueError: If file is not a valid audio format or parameters are invalid
        AttributeError: If sequence creation fails
    """
    import bpy

    # Validate file exists
    if not os.path.exists(filepath):
        raise FileNotFoundError(f"File not found: {filepath}")

    # Validate audio file extension
    ext = os.path.splitext(filepath)[1].lower()
    audio_extensions = {'.wav', '.mp3', '.flac', '.ogg', '.aac', '.m4a', '.wma'}
    if ext not in audio_extensions:
        raise ValueError(f"Invalid audio file extension: {ext}. Supported: {audio_extensions}")

    scene = bpy.context.scene

    # Create sequence editor if it doesn't exist
    if not scene.sequence_editor:
        scene.sequence_editor_create()

    sequences = scene.sequence_editor.sequences

    # Generate name if not provided
    if name is None:
        name = os.path.splitext(os.path.basename(filepath))[0]

    # Create audio sequence (no fit_method for audio)
    sequence = sequences.new_sound(name=name, filepath=filepath, channel=channel, frame_start=frame_start)

    # Apply frame_end if specified (audio can only be trimmed, not extended)
    if frame_end is not None:
        original_duration = sequence.frame_final_duration
        requested_duration = frame_end - frame_start
        
        if requested_duration <= 0:
            raise ValueError(f"Frame end ({frame_end}) must be greater than frame start ({frame_start})")
        
        # Prevent extending beyond original duration
        if requested_duration > original_duration:
            raise ValueError(
                f"Cannot extend audio beyond original duration. "
                f"Requested duration: {requested_duration} frames, "
                f"actual duration: {original_duration} frames"
            )
        
        # Set the sequence end frame (trims audio)
        sequence.frame_final_end = frame_end

    # Auto-fit sequencer view to show all sequences
    try:
        fit_sequencer_view()
    except:
        pass  # Silently ignore fit errors
    
    # Return sequence info in same format as get_timeline_items()
    return {
        "name": sequence.name,
        "type": sequence.type,
        "channel": sequence.channel,
        "frame_start": sequence.frame_start,
        "frame_end": sequence.frame_final_end,
        "duration": sequence.frame_final_duration,
        "filepath": _get_sequence_filepath(sequence)
    }



def add_transition(
    sequence1_name: str,
    sequence2_name: str,
    transition_type: str = 'CROSS',
    duration: int = 10,
    channel: Optional[int] = None
) -> Dict[str, Any]:
    """
    Add a transition effect between two sequences on the timeline.

    Args:
        sequence1_name: Name of the first sequence
        sequence2_name: Name of the second sequence
        transition_type: Type of transition ('CROSS', 'WIPE', 'GAMMA_CROSS')
        duration: Duration of transition in frames (default: 10)
        channel: Channel for transition effect (auto-detected if None)

    Returns:
        Dict[str, Any]: Dictionary with transition information matching get_timeline_items() format

    Raises:
        ImportError: If bpy module is not available
        ValueError: If sequences don't exist or aren't compatible for transitions
        AttributeError: If sequence editor is not available
    """
    import bpy

    scene = bpy.context.scene

    if not scene.sequence_editor:
        raise AttributeError("No sequence editor found - timeline is empty or not initialized")

    sequences = scene.sequence_editor.sequences

    # Find the sequences by name
    seq1 = None
    seq2 = None

    for seq in sequences:
        if seq.name == sequence1_name:
            seq1 = seq
        elif seq.name == sequence2_name:
            seq2 = seq

    if seq1 is None:
        raise ValueError(f"Sequence '{sequence1_name}' not found")
    if seq2 is None:
        raise ValueError(f"Sequence '{sequence2_name}' not found")

    # Determine transition channel
    if channel is None:
        channel = max(seq1.channel, seq2.channel) + 1

    # Validate transition type
    valid_transitions = {'CROSS', 'WIPE', 'GAMMA_CROSS'}
    if transition_type not in valid_transitions:
        raise ValueError(f"Invalid transition type: {transition_type}. Must be one of {valid_transitions}")

    # Calculate overlap region for transition
    overlap_start = max(seq1.frame_start, seq2.frame_start)
    overlap_end = min(seq1.frame_final_end, seq2.frame_final_end)

    if overlap_start >= overlap_end:
        raise ValueError(f"Sequences '{sequence1_name}' and '{sequence2_name}' do not overlap")

    # Adjust transition duration if it exceeds overlap
    max_duration = overlap_end - overlap_start
    if duration > max_duration:
        duration = max_duration

    # Calculate transition frame start
    transition_start = overlap_end - duration

    # Create transition effect
    transition_name = f"transition_{sequence1_name}_{sequence2_name}"

    if transition_type == 'CROSS':
        transition = sequences.new_effect(
            name=transition_name,
            type='CROSS',
            channel=channel,
            frame_start=transition_start,
            frame_end=transition_start + duration,
            seq1=seq1,
            seq2=seq2
        )
    elif transition_type == 'WIPE':
        transition = sequences.new_effect(
            name=transition_name,
            type='WIPE',
            channel=channel,
            frame_start=transition_start,
            frame_end=transition_start + duration,
            seq1=seq1,
            seq2=seq2
        )
    elif transition_type == 'GAMMA_CROSS':
        transition = sequences.new_effect(
            name=transition_name,
            type='GAMMA_CROSS',
            channel=channel,
            frame_start=transition_start,
            frame_end=transition_start + duration,
            seq1=seq1,
            seq2=seq2
        )

    # Auto-fit sequencer view to show all sequences
    try:
        fit_sequencer_view()
    except:
        pass  # Silently ignore fit errors
    
    # Return transition info in same format as get_timeline_items()
    return {
        "name": transition.name,
        "type": transition.type,
        "channel": transition.channel,
        "frame_start": transition.frame_start,
        "frame_end": transition.frame_final_end,
        "duration": transition.frame_final_duration,
        "filepath": None  # Transitions don't have filepaths
    }


def add_text(
    text: str,
    channel: int,
    frame_start: int,
    duration: int,
    name: Optional[str] = None,
    font_size: int = 50,
    color: Tuple[float, float, float, float] = (1.0, 1.0, 1.0, 1.0),
    location: Tuple[int, int] = (0, 0),
    use_background: bool = False,
    background_color: Tuple[float, float, float, float] = (0.0, 0.0, 0.0, 0.8)
) -> Dict[str, Any]:
    """
    Add a text element to the timeline at a specific channel and position.

    Args:
        text: The text content to display
        channel: Channel number to place the text sequence
        frame_start: Starting frame position
        duration: Duration in frames for how long text appears
        name: Optional sequence name (auto-generated if None)
        font_size: Font size in pixels (default: 50)
        color: Text color as RGBA tuple (values 0.0-1.0, default: white)
        location: Text position as (X, Y) tuple in pixels (default: center)
        use_background: Whether to show background behind text (default: False)
        background_color: Background color as RGBA tuple (values 0.0-1.0, default: semi-transparent black)

    Returns:
        Dict[str, Any]: Dictionary with sequence information matching get_timeline_items() format

    Raises:
        ImportError: If bpy module is not available
        ValueError: If parameters are invalid
        AttributeError: If sequence creation fails
    """
    import bpy

    scene = bpy.context.scene

    # Create sequence editor if it doesn't exist
    if not scene.sequence_editor:
        scene.sequence_editor_create()

    sequences = scene.sequence_editor.sequences

    # Generate name if not provided
    if name is None:
        name = f"text_{len([s for s in sequences if s.type == 'TEXT']) + 1:03d}"

    # Validate parameters
    if duration <= 0:
        raise ValueError(f"Duration must be positive, got: {duration}")
    if font_size <= 0:
        raise ValueError(f"Font size must be positive, got: {font_size}")
    if len(color) != 4 or not all(0.0 <= c <= 1.0 for c in color):
        raise ValueError(f"Color must be RGBA tuple with values 0.0-1.0, got: {color}")
    if len(background_color) != 4 or not all(0.0 <= c <= 1.0 for c in background_color):
        raise ValueError(f"Background color must be RGBA tuple with values 0.0-1.0, got: {background_color}")
    if len(location) != 2:
        raise ValueError(f"Location must be (X, Y) tuple, got: {location}")

    # Create text sequence using effect
    text_sequence = sequences.new_effect(
        name=name,
        type='TEXT',
        channel=channel,
        frame_start=frame_start,
        frame_end=frame_start + duration
    )

    # Configure text properties
    text_sequence.text = text
    text_sequence.font_size = font_size
    text_sequence.color = color
    text_sequence.location = location
    text_sequence.use_background = use_background
    if use_background:
        text_sequence.background_color = background_color

    # Auto-fit sequencer view to show all sequences
    try:
        fit_sequencer_view()
    except:
        pass  # Silently ignore fit errors
    
    # Return sequence info in same format as get_timeline_items()
    return {
        "name": text_sequence.name,
        "type": text_sequence.type,
        "channel": text_sequence.channel,
        "frame_start": text_sequence.frame_start,
        "frame_end": text_sequence.frame_final_end,
        "duration": text_sequence.frame_final_duration,
        "filepath": None  # Text sequences don't have filepaths
    }


def add_color(
    color: Tuple[float, float, float, float],
    channel: int,
    frame_start: int,
    duration: int,
    name: Optional[str] = None
) -> Dict[str, Any]:
    """
    Add a solid color element to the timeline at a specific channel and position.
    
    Args:
        color: Color as RGBA tuple (values 0.0-1.0)
        channel: Channel number to place the color sequence
        frame_start: Starting frame position
        duration: Duration in frames for how long color appears
        name: Optional sequence name (auto-generated if None)
        
    Returns:
        Dict[str, Any]: Dictionary with sequence information matching get_timeline_items() format
        
    Raises:
        ImportError: If bpy module is not available
        ValueError: If parameters are invalid
        AttributeError: If sequence creation fails
    """
    import bpy
    
    scene = bpy.context.scene
    
    # Create sequence editor if it doesn't exist
    if not scene.sequence_editor:
        scene.sequence_editor_create()
    
    sequences = scene.sequence_editor.sequences
    
    # Generate name if not provided
    if name is None:
        name = f"color_{len([s for s in sequences if s.type == 'COLOR']) + 1:03d}"
    
    # Validate parameters
    if duration <= 0:
        raise ValueError(f"Duration must be positive, got: {duration}")
    if len(color) != 4 or not all(0.0 <= c <= 1.0 for c in color):
        raise ValueError(f"Color must be RGBA tuple with values 0.0-1.0, got: {color}")
    
    # Create color sequence using effect
    color_sequence = sequences.new_effect(
        name=name,
        type='COLOR',
        channel=channel,
        frame_start=frame_start,
        frame_end=frame_start + duration
    )
    
    # Configure color properties
    color_sequence.color = color
    
    # Auto-fit sequencer view to show all sequences
    try:
        fit_sequencer_view()
    except:
        pass  # Silently ignore fit errors
    
    # Return sequence info in same format as get_timeline_items()
    return {
        "name": color_sequence.name,
        "type": color_sequence.type,
        "channel": color_sequence.channel,
        "frame_start": color_sequence.frame_start,
        "frame_end": color_sequence.frame_final_end,
        "duration": color_sequence.frame_final_duration,
        "filepath": None  # Color sequences don't have filepaths
    }


def delete_timeline_item(sequence_name: str) -> Dict[str, Any]:
    """
    Delete a sequence from the timeline by name.
    
    Args:
        sequence_name: Name of the sequence to delete
        
    Returns:
        Dict[str, Any]: Dictionary containing:
            - success: bool - Whether the operation succeeded
            - message: str - Success/error message
            - deleted_sequence: Optional[Dict[str, Any]] - Info about deleted sequence
            
    Raises:
        ImportError: If bpy module is not available
        ValueError: If sequence is not found
        AttributeError: If sequence editor is not available
    """
    import bpy
    
    scene = bpy.context.scene
    
    if not scene.sequence_editor:
        raise AttributeError("No sequence editor found - timeline is empty or not initialized")
    
    sequences = scene.sequence_editor.sequences
    
    # Find the sequence by name
    target_sequence = None
    for seq in sequences:
        if seq.name == sequence_name:
            target_sequence = seq
            break
    
    if target_sequence is None:
        raise ValueError(f"Sequence '{sequence_name}' not found")
    
    # Store sequence info before deletion
    deleted_sequence_info = {
        "name": target_sequence.name,
        "type": target_sequence.type,
        "channel": target_sequence.channel,
        "frame_start": target_sequence.frame_start,
        "frame_end": target_sequence.frame_final_end,
        "duration": target_sequence.frame_final_duration,
        "filepath": _get_sequence_filepath(target_sequence)
    }
    
    try:
        # Remove the sequence
        sequences.remove(target_sequence)
        
        # Auto-fit sequencer view to show remaining sequences
        try:
            fit_sequencer_view()
        except:
            pass  # Silently ignore fit errors
        
        return {
            "success": True,
            "message": f"Successfully deleted sequence '{sequence_name}'",
            "deleted_sequence": deleted_sequence_info
        }
        
    except Exception as e:
        return {
            "success": False,
            "message": f"Failed to delete sequence '{sequence_name}': {str(e)}",
            "deleted_sequence": None
        }


def fit_sequencer_view() -> Dict[str, Any]:
    """
    Fit the sequencer view to show all sequences in the timeline.
    
    This function finds the sequence editor area and fits the view to display
    all sequences, similar to pressing "View All" in the sequencer interface.
    
    Returns:
        Dict[str, Any]: Dictionary containing:
            - success: bool - Whether the operation succeeded
            - message: str - Success/error message
            - area_found: bool - Whether sequence editor area was found
            
    Raises:
        ImportError: If bpy module is not available
        AttributeError: If sequence editor is not available
    """
    import bpy
    
    # Find the sequence editor area
    sequencer_area = None
    sequencer_region = None
    
    for area in bpy.context.screen.areas:
        if area.type == 'SEQUENCE_EDITOR':
            sequencer_area = area
            for region in area.regions:
                if region.type == 'WINDOW':
                    sequencer_region = region
                    break
            break
    
    if sequencer_area is None:
        return {
            "success": False,
            "message": "Sequence editor area not found. Make sure the Video Editing workspace is active or a sequence editor area is visible.",
            "area_found": False
        }
    
    if sequencer_region is None:
        return {
            "success": False,
            "message": "Sequence editor window region not found.",
            "area_found": True
        }
    
    try:
        # Create context override for the sequencer area
        # Use modern Blender 3.2+ context override syntax
        ctx_override = {'area': sequencer_area, 'region': sequencer_region}
        
        # Fit the sequencer view to show all sequences
        with bpy.context.temp_override(**ctx_override):
            bpy.ops.sequencer.view_all()
        
        return {
            "success": True,
            "message": "Successfully fitted sequencer view to show all sequences",
            "area_found": True
        }
        
    except Exception as e:
        # Fallback to older context override method
        try:
            ctx = bpy.context.copy()
            ctx['area'] = sequencer_area
            ctx['region'] = sequencer_region
            bpy.ops.sequencer.view_all(ctx)
            
            return {
                "success": True,
                "message": "Successfully fitted sequencer view to show all sequences (fallback method)",
                "area_found": True
            }
        except Exception as e2:
            return {
                "success": False,
                "message": f"Failed to fit sequencer view: {str(e)} (fallback also failed: {str(e2)})",
                "area_found": True
            }


def set_frame_range(start_frame: int, end_frame: int) -> Dict[str, Any]:
    """
    Set the timeline frame range (start and end frames).
    
    Args:
        start_frame: Starting frame number for the timeline
        end_frame: Ending frame number for the timeline
        
    Returns:
        Dict[str, Any]: Dictionary containing:
            - success: bool - Whether the operation succeeded
            - message: str - Success/error message
            - frame_start: int - Set start frame
            - frame_end: int - Set end frame
            
    Raises:
        ImportError: If bpy module is not available
        ValueError: If frame range is invalid
    """
    import bpy
    
    # Validate frame range
    if start_frame < 0:
        raise ValueError(f"Start frame must be non-negative, got: {start_frame}")
    if end_frame < start_frame:
        raise ValueError(f"End frame ({end_frame}) must be >= start frame ({start_frame})")
    
    scene = bpy.context.scene
    
    try:
        # Set the frame range
        scene.frame_start = start_frame
        scene.frame_end = end_frame
        
        return {
            "success": True,
            "message": f"Successfully set frame range to {start_frame}-{end_frame}",
            "frame_start": start_frame,
            "frame_end": end_frame
        }
        
    except Exception as e:
        raise ValueError(f"Failed to set frame range: {str(e)}")


def get_frame_range() -> Dict[str, Any]:
    """
    Get the current timeline frame range and playhead position.
    
    Returns:
        Dict[str, Any]: Dictionary containing:
            - frame_start: int - Current start frame
            - frame_end: int - Current end frame
            - frame_current: int - Current playhead position
            - total_frames: int - Total number of frames in range
            
    Raises:
        ImportError: If bpy module is not available
    """
    import bpy
    
    scene = bpy.context.scene
    
    frame_start = scene.frame_start
    frame_end = scene.frame_end
    frame_current = scene.frame_current
    total_frames = frame_end - frame_start + 1
    
    return {
        "frame_start": frame_start,
        "frame_end": frame_end,
        "frame_current": frame_current,
        "total_frames": total_frames
    }


def set_current_frame(frame: int) -> Dict[str, Any]:
    """
    Set the current playhead position on the timeline.
    
    Args:
        frame: Frame number to set as current position
        
    Returns:
        Dict[str, Any]: Dictionary containing:
            - success: bool - Whether the operation succeeded
            - message: str - Success/error message
            - frame_current: int - Set current frame
            - in_range: bool - Whether frame is within timeline range
            
    Raises:
        ImportError: If bpy module is not available
        ValueError: If frame number is invalid
    """
    import bpy
    
    # Validate frame number
    if frame < 0:
        raise ValueError(f"Frame must be non-negative, got: {frame}")
    
    scene = bpy.context.scene
    
    try:
        # Set the current frame
        scene.frame_current = frame
        
        # Check if frame is within timeline range
        in_range = scene.frame_start <= frame <= scene.frame_end
        
        return {
            "success": True,
            "message": f"Successfully set current frame to {frame}",
            "frame_current": frame,
            "in_range": in_range
        }
        
    except Exception as e:
        raise ValueError(f"Failed to set current frame: {str(e)}")


def get_current_frame() -> int:
    """
    Get the current playhead position on the timeline.
    
    Returns:
        int: Current frame number
        
    Raises:
        ImportError: If bpy module is not available
    """
    import bpy
    
    return bpy.context.scene.frame_current


def _find_sequence_by_name(sequence_name: str):
    """
    Helper function to find a sequence by name.
    
    Args:
        sequence_name: Name of the sequence to find
        
    Returns:
        Blender sequence object
        
    Raises:
        ImportError: If bpy module is not available
        ValueError: If sequence is not found
        AttributeError: If sequence editor is not available
    """
    import bpy
    
    scene = bpy.context.scene
    
    if not scene.sequence_editor:
        raise AttributeError("No sequence editor found - timeline is empty or not initialized")
    
    sequences = scene.sequence_editor.sequences
    
    # Find the sequence by name
    target_sequence = None
    for seq in sequences:
        if seq.name == sequence_name:
            target_sequence = seq
            break
    
    if target_sequence is None:
        raise ValueError(f"Sequence '{sequence_name}' not found")
    
    return target_sequence


def _apply_sequence_trimming(target_sequence, trim_start: Optional[int], trim_end: Optional[int], allow_extension: bool = False):
    """
    Helper function to apply trimming to a sequence.
    
    Args:
        target_sequence: Blender sequence object
        trim_start: Optional new start frame for trimming
        trim_end: Optional new end frame for trimming
        allow_extension: Whether to allow extending beyond original duration
        
    Raises:
        ValueError: If trim parameters are invalid
    """
    if trim_start is not None or trim_end is not None:
        original_start = target_sequence.frame_start
        original_end = target_sequence.frame_final_end
        original_duration = target_sequence.frame_final_duration
        
        new_start = trim_start if trim_start is not None else original_start
        new_end = trim_end if trim_end is not None else original_end
        
        # Validate trim parameters
        if new_start < 0:
            raise ValueError(f"Trim start must be non-negative, got: {new_start}")
        if new_end <= new_start:
            raise ValueError(f"Trim end ({new_end}) must be greater than trim start ({new_start})")
        
        new_duration = new_end - new_start
        
        # Check against original duration if extension is not allowed
        if not allow_extension and new_duration > original_duration:
            raise ValueError(
                f"Cannot extend {target_sequence.type.lower()} beyond original duration. "
                f"Requested duration: {new_duration} frames, "
                f"original duration: {original_duration} frames"
            )
        
        # Apply trimming
        target_sequence.frame_start = new_start
        target_sequence.frame_final_end = new_end


def _apply_sequence_transforms(target_sequence, sequence_name: str, scale: Optional[Union[float, Tuple[float, float]]], position: Optional[Tuple[float, float]], rotation: Optional[float]):
    """
    Helper function to apply transform operations to a sequence.
    
    Args:
        target_sequence: Blender sequence object
        sequence_name: Name of the sequence (for error messages)
        scale: Optional scale factor
        position: Optional position offset
        rotation: Optional rotation angle
        
    Raises:
        ValueError: If transform parameters are invalid
        AttributeError: If sequence doesn't support transforms
    """
    from typing import Union
    
    if scale is not None or position is not None or rotation is not None:
        if not hasattr(target_sequence, 'transform'):
            raise AttributeError(f"Sequence '{sequence_name}' does not support transform operations")
        
        transform = target_sequence.transform
        
        # Apply scaling
        if scale is not None:
            if isinstance(scale, (int, float)):
                # Uniform scaling
                if scale <= 0:
                    raise ValueError(f"Scale must be positive, got: {scale}")
                transform.scale_x = scale
                transform.scale_y = scale
            elif isinstance(scale, (tuple, list)) and len(scale) == 2:
                # Non-uniform scaling
                scale_x, scale_y = scale
                if scale_x <= 0 or scale_y <= 0:
                    raise ValueError(f"Scale values must be positive, got: {scale}")
                transform.scale_x = scale_x
                transform.scale_y = scale_y
            else:
                raise ValueError(f"Scale must be a number or (x, y) tuple, got: {scale}")
        
        # Apply position offset
        if position is not None:
            if not isinstance(position, (tuple, list)) or len(position) != 2:
                raise ValueError(f"Position must be (x, y) tuple, got: {position}")
            transform.offset_x = position[0]
            transform.offset_y = position[1]
        
        # Apply rotation
        if rotation is not None:
            if not isinstance(rotation, (int, float)):
                raise ValueError(f"Rotation must be a number (radians), got: {rotation}")
            transform.rotation = rotation


def _get_updated_sequence_info(target_sequence) -> Dict[str, Any]:
    """
    Helper function to get updated sequence information.
    
    Args:
        target_sequence: Blender sequence object
        
    Returns:
        Dict[str, Any]: Updated sequence information matching get_timeline_items() format
    """
    # Auto-fit sequencer view to show all sequences
    try:
        fit_sequencer_view()
    except:
        pass  # Silently ignore fit errors
    
    # Return updated sequence info in same format as get_timeline_items()
    transform_data, original_res, is_resized = _get_sequence_transform_info(target_sequence)
    
    result = {
        "name": target_sequence.name,
        "type": target_sequence.type,
        "channel": target_sequence.channel,
        "frame_start": target_sequence.frame_start,
        "frame_end": target_sequence.frame_final_end,
        "duration": target_sequence.frame_final_duration,
        "filepath": _get_sequence_filepath(target_sequence),
        "original_resolution": original_res,
        "transform": transform_data,
        "is_resized": is_resized
    }
    
    # Add speed information for video sequences
    if target_sequence.type == 'MOVIE' and hasattr(target_sequence, 'speed_factor'):
        result["speed"] = target_sequence.speed_factor
    
    return result


def modify_image(
    sequence_name: str,
    trim_start: Optional[int] = None,
    trim_end: Optional[int] = None,
    scale: Optional[Union[float, Tuple[float, float]]] = None,
    position: Optional[Tuple[float, float]] = None,
    rotation: Optional[float] = None
) -> Dict[str, Any]:
    """
    Modify image sequences on the timeline (trim, zoom, reposition).
    
    This function only works with IMAGE sequence types. Other sequence types
    (MOVIE, SOUND, TEXT, COLOR, effects) are not supported and will raise an error.
    
    Args:
        sequence_name: Name of the sequence to modify
        trim_start: Optional new start frame for trimming
        trim_end: Optional new end frame for trimming
        scale: Optional scale factor - float for uniform scaling or (x, y) tuple for non-uniform
        position: Optional (x, y) offset in pixels from center
        rotation: Optional rotation angle in radians
        
    Returns:
        Dict[str, Any]: Updated sequence information matching get_timeline_items() format
        
    Raises:
        ImportError: If bpy module is not available
        ValueError: If sequence is not found, wrong type, or parameters are invalid
        AttributeError: If sequence editor is not available
    """
    from typing import Union
    
    # Find the sequence by name
    target_sequence = _find_sequence_by_name(sequence_name)
    
    # Check if sequence type is supported
    if target_sequence.type != 'IMAGE':
        raise ValueError(
            f"Sequence '{sequence_name}' has type '{target_sequence.type}'. "
            f"This function only supports IMAGE sequences."
        )
    
    # Apply trimming if specified (images can be extended)
    _apply_sequence_trimming(target_sequence, trim_start, trim_end, allow_extension=True)
    
    # Apply transform operations if specified
    _apply_sequence_transforms(target_sequence, sequence_name, scale, position, rotation)
    
    # Return updated sequence info
    return _get_updated_sequence_info(target_sequence)


def modify_video(
    sequence_name: str,
    trim_start: Optional[int] = None,
    trim_end: Optional[int] = None,
    scale: Optional[Union[float, Tuple[float, float]]] = None,
    position: Optional[Tuple[float, float]] = None,
    rotation: Optional[float] = None,
    speed: Optional[float] = None
) -> Dict[str, Any]:
    """
    Modify video sequences on the timeline (trim, zoom, reposition, speed).
    
    This function only works with MOVIE sequence types. Other sequence types
    (IMAGE, SOUND, TEXT, COLOR, effects) are not supported and will raise an error.
    
    Args:
        sequence_name: Name of the sequence to modify
        trim_start: Optional new start frame for trimming
        trim_end: Optional new end frame for trimming
        scale: Optional scale factor - float for uniform scaling or (x, y) tuple for non-uniform
        position: Optional (x, y) offset in pixels from center
        rotation: Optional rotation angle in radians
        speed: Optional playback speed multiplier (0.1-10.0, where 1.0 = normal speed)
        
    Returns:
        Dict[str, Any]: Updated sequence information matching get_timeline_items() format
        
    Raises:
        ImportError: If bpy module is not available
        ValueError: If sequence is not found, wrong type, or parameters are invalid
        AttributeError: If sequence editor is not available
    """
    from typing import Union
    
    # Find the sequence by name
    target_sequence = _find_sequence_by_name(sequence_name)
    
    # Check if sequence type is supported
    if target_sequence.type != 'MOVIE':
        raise ValueError(
            f"Sequence '{sequence_name}' has type '{target_sequence.type}'. "
            f"This function only supports MOVIE sequences."
        )
    
    # Apply trimming if specified (videos cannot be extended)
    _apply_sequence_trimming(target_sequence, trim_start, trim_end, allow_extension=False)
    
    # Apply transform operations if specified
    _apply_sequence_transforms(target_sequence, sequence_name, scale, position, rotation)
    
    # Apply speed modification if specified
    if speed is not None:
        if not isinstance(speed, (int, float)):
            raise ValueError(f"Speed must be a number, got: {speed}")
        if not (0.1 <= speed <= 10.0):
            raise ValueError(f"Speed must be between 0.1 and 10.0, got: {speed}")
        
        # Set speed factor if available
        if hasattr(target_sequence, 'speed_factor'):
            target_sequence.speed_factor = speed
        else:
            raise AttributeError(f"Sequence '{sequence_name}' does not support speed control")
    
    # Return updated sequence info
    return _get_updated_sequence_info(target_sequence)


def duplicate_timeline_element(
    sequence_name: str,
    new_channel: int,
    new_frame_start: int,
    new_name: Optional[str] = None
) -> Dict[str, Any]:
    """
    Duplicate an existing sequence to a new channel and position.
    
    Creates an exact copy of the specified sequence with all its properties
    (transform, color, volume, etc.) at the new location.
    
    Args:
        sequence_name: Name of the sequence to duplicate
        new_channel: Channel number for the duplicated sequence
        new_frame_start: Starting frame position for the duplicated sequence
        new_name: Optional name for the duplicate (auto-generated if None)
        
    Returns:
        Dict[str, Any]: Dictionary with duplicated sequence information matching get_timeline_items() format
        
    Raises:
        ImportError: If bpy module is not available
        ValueError: If sequence is not found or parameters are invalid
        AttributeError: If sequence editor is not available
    """
    import bpy
    
    scene = bpy.context.scene
    
    if not scene.sequence_editor:
        raise AttributeError("No sequence editor found - timeline is empty or not initialized")
    
    sequences = scene.sequence_editor.sequences
    
    # Find the original sequence by name
    original_sequence = None
    for seq in sequences:
        if seq.name == sequence_name:
            original_sequence = seq
            break
    
    if original_sequence is None:
        raise ValueError(f"Sequence '{sequence_name}' not found")
    
    # Validate parameters
    if new_channel < 1:
        raise ValueError(f"Channel must be >= 1, got: {new_channel}")
    if new_frame_start < 0:
        raise ValueError(f"Frame start must be non-negative, got: {new_frame_start}")
    
    # Generate new name if not provided
    if new_name is None:
        base_name = original_sequence.name
        # Remove existing _copy suffix if present
        if base_name.endswith('_copy'):
            base_name = base_name[:-5]
        
        # Find next available copy name
        counter = 1
        new_name = f"{base_name}_copy"
        while any(seq.name == new_name for seq in sequences):
            counter += 1
            new_name = f"{base_name}_copy{counter:02d}" if counter > 1 else f"{base_name}_copy"
    
    # Calculate duration for the duplicate
    original_duration = original_sequence.frame_final_duration
    new_frame_end = new_frame_start + original_duration
    
    # Create duplicate based on sequence type
    duplicate_sequence = None
    
    if original_sequence.type == 'IMAGE':
        filepath = _get_sequence_filepath(original_sequence)
        if filepath is None:
            raise ValueError(f"Cannot duplicate image sequence '{sequence_name}': filepath not found")
        
        fit_method = getattr(original_sequence, 'fit_method', 'ORIGINAL')
        duplicate_sequence = sequences.new_image(
            name=new_name,
            filepath=filepath,
            channel=new_channel,
            frame_start=new_frame_start,
            fit_method=fit_method
        )
        # Set duration to match original
        duplicate_sequence.frame_final_end = new_frame_end
        
    elif original_sequence.type == 'MOVIE':
        filepath = _get_sequence_filepath(original_sequence)
        if filepath is None:
            raise ValueError(f"Cannot duplicate movie sequence '{sequence_name}': filepath not found")
        
        fit_method = getattr(original_sequence, 'fit_method', 'ORIGINAL')
        duplicate_sequence = sequences.new_movie(
            name=new_name,
            filepath=filepath,
            channel=new_channel,
            frame_start=new_frame_start,
            fit_method=fit_method
        )
        # Trim to match original duration if needed
        if duplicate_sequence.frame_final_duration > original_duration:
            duplicate_sequence.frame_final_end = new_frame_end
            
    elif original_sequence.type == 'SOUND':
        filepath = _get_sequence_filepath(original_sequence)
        if filepath is None:
            raise ValueError(f"Cannot duplicate sound sequence '{sequence_name}': filepath not found")
        
        duplicate_sequence = sequences.new_sound(
            name=new_name,
            filepath=filepath,
            channel=new_channel,
            frame_start=new_frame_start
        )
        # Trim to match original duration if needed
        if duplicate_sequence.frame_final_duration > original_duration:
            duplicate_sequence.frame_final_end = new_frame_end
            
    elif original_sequence.type in ['TEXT', 'COLOR', 'CROSS', 'WIPE', 'GAMMA_CROSS']:
        # Create effect-based sequences
        if original_sequence.type == 'TEXT':
            duplicate_sequence = sequences.new_effect(
                name=new_name,
                type='TEXT',
                channel=new_channel,
                frame_start=new_frame_start,
                frame_end=new_frame_end
            )
        elif original_sequence.type == 'COLOR':
            duplicate_sequence = sequences.new_effect(
                name=new_name,
                type='COLOR',
                channel=new_channel,
                frame_start=new_frame_start,
                frame_end=new_frame_end
            )
        else:
            # For transition effects, we need the original input sequences
            # This is complex, so for now we'll create a basic effect
            duplicate_sequence = sequences.new_effect(
                name=new_name,
                type=original_sequence.type,
                channel=new_channel,
                frame_start=new_frame_start,
                frame_end=new_frame_end
            )
    else:
        raise ValueError(f"Unsupported sequence type for duplication: {original_sequence.type}")
    
    if duplicate_sequence is None:
        raise ValueError(f"Failed to create duplicate of sequence '{sequence_name}'")
    
    # Copy properties from original to duplicate
    try:
        # Copy transform properties if available
        if hasattr(original_sequence, 'transform') and hasattr(duplicate_sequence, 'transform'):
            orig_transform = original_sequence.transform
            dup_transform = duplicate_sequence.transform
            
            dup_transform.scale_x = orig_transform.scale_x
            dup_transform.scale_y = orig_transform.scale_y
            dup_transform.offset_x = orig_transform.offset_x
            dup_transform.offset_y = orig_transform.offset_y
            dup_transform.rotation = orig_transform.rotation
        
        # Copy audio properties if available
        if original_sequence.type == 'SOUND':
            if hasattr(original_sequence, 'volume'):
                duplicate_sequence.volume = original_sequence.volume
            if hasattr(original_sequence, 'pan'):
                duplicate_sequence.pan = original_sequence.pan
        
        # Copy text properties if available
        if original_sequence.type == 'TEXT':
            if hasattr(original_sequence, 'text'):
                duplicate_sequence.text = original_sequence.text
            if hasattr(original_sequence, 'font_size'):
                duplicate_sequence.font_size = original_sequence.font_size
            if hasattr(original_sequence, 'color'):
                duplicate_sequence.color = original_sequence.color
            if hasattr(original_sequence, 'location'):
                duplicate_sequence.location = original_sequence.location
            if hasattr(original_sequence, 'use_background'):
                duplicate_sequence.use_background = original_sequence.use_background
            if hasattr(original_sequence, 'background_color'):
                duplicate_sequence.background_color = original_sequence.background_color
        
        # Copy color properties if available
        if original_sequence.type == 'COLOR':
            if hasattr(original_sequence, 'color'):
                duplicate_sequence.color = original_sequence.color
                
    except Exception:
        # If property copying fails, continue with basic duplicate
        pass
    
    # Auto-fit sequencer view to show all sequences
    try:
        fit_sequencer_view()
    except:
        pass  # Silently ignore fit errors
    
    # Return sequence info in same format as get_timeline_items()
    transform_data, original_res, is_resized = _get_sequence_transform_info(duplicate_sequence)
    
    return {
        "name": duplicate_sequence.name,
        "type": duplicate_sequence.type,
        "channel": duplicate_sequence.channel,
        "frame_start": duplicate_sequence.frame_start,
        "frame_end": duplicate_sequence.frame_final_end,
        "duration": duplicate_sequence.frame_final_duration,
        "filepath": _get_sequence_filepath(duplicate_sequence),
        "original_resolution": original_res,
        "transform": transform_data,
        "is_resized": is_resized
    }


def modify_audio(
    sequence_name: str,
    trim_start: Optional[int] = None,
    trim_end: Optional[int] = None,
    volume: Optional[float] = None,
    pan: Optional[float] = None
) -> Dict[str, Any]:
    """
    Modify audio sequences on the timeline (trim, volume, pan).
    
    This function only works with SOUND sequence types. Other sequence types
    (IMAGE, MOVIE, TEXT, COLOR, effects) are not supported and will raise an error.
    
    Args:
        sequence_name: Name of the sequence to modify
        trim_start: Optional new start frame for trimming
        trim_end: Optional new end frame for trimming
        volume: Optional volume level (0.0-100.0, where 100.0 is full volume)
        pan: Optional stereo panning (-inf to +inf, 0.0 is center, only for mono sources)
        
    Returns:
        Dict[str, Any]: Updated sequence information matching get_timeline_items() format
        
    Raises:
        ImportError: If bpy module is not available
        ValueError: If sequence is not found, wrong type, or parameters are invalid
        AttributeError: If sequence editor is not available
    """
    import bpy
    
    scene = bpy.context.scene
    
    if not scene.sequence_editor:
        raise AttributeError("No sequence editor found - timeline is empty or not initialized")
    
    sequences = scene.sequence_editor.sequences
    
    # Find the sequence by name
    target_sequence = None
    for seq in sequences:
        if seq.name == sequence_name:
            target_sequence = seq
            break
    
    if target_sequence is None:
        raise ValueError(f"Sequence '{sequence_name}' not found")
    
    # Check if sequence type is supported
    if target_sequence.type != 'SOUND':
        raise ValueError(
            f"Sequence '{sequence_name}' has type '{target_sequence.type}'. "
            f"This function only supports SOUND sequences."
        )
    
    # Apply trimming if specified
    if trim_start is not None or trim_end is not None:
        original_start = target_sequence.frame_start
        original_end = target_sequence.frame_final_end
        original_duration = target_sequence.frame_final_duration
        
        new_start = trim_start if trim_start is not None else original_start
        new_end = trim_end if trim_end is not None else original_end
        
        # Validate trim parameters
        if new_start < 0:
            raise ValueError(f"Trim start must be non-negative, got: {new_start}")
        if new_end <= new_start:
            raise ValueError(f"Trim end ({new_end}) must be greater than trim start ({new_start})")
        
        new_duration = new_end - new_start
        
        # Audio can only be trimmed, not extended beyond original duration
        if new_duration > original_duration:
            raise ValueError(
                f"Cannot extend audio beyond original duration. "
                f"Requested duration: {new_duration} frames, "
                f"original duration: {original_duration} frames"
            )
        
        # Apply trimming
        target_sequence.frame_start = new_start
        target_sequence.frame_final_end = new_end
    
    # Apply audio modifications if specified
    if volume is not None:
        if not (0.0 <= volume <= 100.0):
            raise ValueError(f"Volume must be between 0.0 and 100.0, got: {volume}")
        target_sequence.volume = volume
    
    if pan is not None:
        if not isinstance(pan, (int, float)):
            raise ValueError(f"Pan must be a number, got: {pan}")
        target_sequence.pan = pan
    
    # Auto-fit sequencer view to show all sequences
    try:
        fit_sequencer_view()
    except:
        pass  # Silently ignore fit errors
    
    # Return updated sequence info in same format as get_timeline_items()
    return {
        "name": target_sequence.name,
        "type": target_sequence.type,
        "channel": target_sequence.channel,
        "frame_start": target_sequence.frame_start,
        "frame_end": target_sequence.frame_final_end,
        "duration": target_sequence.frame_final_duration,
        "filepath": _get_sequence_filepath(target_sequence),
        "volume": getattr(target_sequence, 'volume', None),
        "pan": getattr(target_sequence, 'pan', None)
    }


def blade_cut(
    sequence_name: str,
    cut_frame: int,
    left_name: Optional[str] = None,
    right_name: Optional[str] = None
) -> Dict[str, Any]:
    """
    Cut a video or audio sequence at a specific frame, creating two separate sequences.
    
    This function splits a MOVIE or SOUND sequence at the specified frame position,
    creating two new sequences: left part (start to cut frame) and right part 
    (cut frame to end). The original sequence is removed after successful cutting.
    
    Args:
        sequence_name: Name of the sequence to cut
        cut_frame: Frame position where to make the cut
        left_name: Optional name for the left part (auto-generated if None)
        right_name: Optional name for the right part (auto-generated if None)
        
    Returns:
        Dict[str, Any]: Dictionary containing:
            - success: bool - Whether the operation succeeded
            - message: str - Success/error message
            - original_sequence: Dict[str, Any] - Original sequence info
            - left_sequence: Optional[Dict[str, Any]] - Left part sequence info
            - right_sequence: Optional[Dict[str, Any]] - Right part sequence info
            
    Raises:
        ImportError: If bpy module is not available
        ValueError: If sequence is not found, wrong type, or cut frame is invalid
        AttributeError: If sequence editor is not available
    """
    import bpy
    
    scene = bpy.context.scene
    
    if not scene.sequence_editor:
        raise AttributeError("No sequence editor found - timeline is empty or not initialized")
    
    sequences = scene.sequence_editor.sequences
    
    # Find the sequence by name
    original_sequence = None
    for seq in sequences:
        if seq.name == sequence_name:
            original_sequence = seq
            break
    
    if original_sequence is None:
        raise ValueError(f"Sequence '{sequence_name}' not found")
    
    # Check if sequence type is supported for cutting
    if original_sequence.type not in ['MOVIE', 'SOUND']:
        raise ValueError(
            f"Sequence '{sequence_name}' has type '{original_sequence.type}'. "
            f"Blade cutting only supports MOVIE and SOUND sequences."
        )
    
    # Validate cut frame position
    seq_start = int(original_sequence.frame_start)
    seq_end = int(original_sequence.frame_final_end)
    cut_frame = int(cut_frame)
    
    if cut_frame <= seq_start:
        raise ValueError(f"Cut frame ({cut_frame}) must be after sequence start ({seq_start})")
    if cut_frame >= seq_end:
        raise ValueError(f"Cut frame ({cut_frame}) must be before sequence end ({seq_end})")
    
    # Store original sequence info before modification
    original_info = {
        "name": original_sequence.name,
        "type": original_sequence.type,
        "channel": original_sequence.channel,
        "frame_start": original_sequence.frame_start,
        "frame_end": original_sequence.frame_final_end,
        "duration": original_sequence.frame_final_duration,
        "filepath": _get_sequence_filepath(original_sequence)
    }
    
    # Generate names for left and right parts if not provided
    if left_name is None:
        left_name = f"{sequence_name}_L"
        counter = 1
        while any(seq.name == left_name for seq in sequences):
            counter += 1
            left_name = f"{sequence_name}_L{counter:02d}" if counter > 1 else f"{sequence_name}_L"
    
    if right_name is None:
        right_name = f"{sequence_name}_R"  
        counter = 1
        while any(seq.name == right_name for seq in sequences):
            counter += 1
            right_name = f"{sequence_name}_R{counter:02d}" if counter > 1 else f"{sequence_name}_R"
    
    try:
        # Get original sequence properties
        original_channel = int(original_sequence.channel)
        original_filepath = _get_sequence_filepath(original_sequence)
        
        if original_filepath is None:
            raise ValueError(f"Cannot cut sequence '{sequence_name}': filepath not found")
        
        # Create left part (start to cut_frame)
        left_sequence = None
        if original_sequence.type == 'MOVIE':
            fit_method = getattr(original_sequence, 'fit_method', 'ORIGINAL')
            left_sequence = sequences.new_movie(
                name=left_name,
                filepath=original_filepath,
                channel=original_channel,
                frame_start=seq_start,
                fit_method=fit_method
            )
            # Trim to cut frame
            left_sequence.frame_final_end = cut_frame
            
        elif original_sequence.type == 'SOUND':
            left_sequence = sequences.new_sound(
                name=left_name,
                filepath=original_filepath,
                channel=original_channel,
                frame_start=seq_start
            )
            # Trim to cut frame
            left_sequence.frame_final_end = cut_frame
        
        # Create right part (cut_frame to end)
        right_sequence = None
        if original_sequence.type == 'MOVIE':
            fit_method = getattr(original_sequence, 'fit_method', 'ORIGINAL')
            right_sequence = sequences.new_movie(
                name=right_name,
                filepath=original_filepath,
                channel=original_channel,
                frame_start=cut_frame,
                fit_method=fit_method
            )
            # Trim to original end
            original_duration = seq_end - seq_start
            right_duration = seq_end - cut_frame
            if right_sequence.frame_final_duration > right_duration:
                right_sequence.frame_final_end = seq_end
                
        elif original_sequence.type == 'SOUND':
            right_sequence = sequences.new_sound(
                name=right_name,
                filepath=original_filepath,
                channel=original_channel,
                frame_start=cut_frame
            )
            # Trim to original end
            right_duration = seq_end - cut_frame
            if right_sequence.frame_final_duration > right_duration:
                right_sequence.frame_final_end = seq_end
        
        # Copy properties from original to both parts
        if left_sequence and right_sequence:
            # Copy transform properties if available
            if (hasattr(original_sequence, 'transform') and 
                hasattr(left_sequence, 'transform') and 
                hasattr(right_sequence, 'transform')):
                
                orig_transform = original_sequence.transform
                
                # Copy to left sequence
                left_transform = left_sequence.transform
                left_transform.scale_x = orig_transform.scale_x
                left_transform.scale_y = orig_transform.scale_y
                left_transform.offset_x = orig_transform.offset_x
                left_transform.offset_y = orig_transform.offset_y
                left_transform.rotation = orig_transform.rotation
                
                # Copy to right sequence
                right_transform = right_sequence.transform
                right_transform.scale_x = orig_transform.scale_x
                right_transform.scale_y = orig_transform.scale_y
                right_transform.offset_x = orig_transform.offset_x
                right_transform.offset_y = orig_transform.offset_y
                right_transform.rotation = orig_transform.rotation
            
            # Copy audio properties if available
            if original_sequence.type == 'SOUND':
                if hasattr(original_sequence, 'volume'):
                    left_sequence.volume = original_sequence.volume
                    right_sequence.volume = original_sequence.volume
                if hasattr(original_sequence, 'pan'):
                    left_sequence.pan = original_sequence.pan
                    right_sequence.pan = original_sequence.pan
            
            # Copy video speed properties if available
            if (original_sequence.type == 'MOVIE' and 
                hasattr(original_sequence, 'speed_factor')):
                if hasattr(left_sequence, 'speed_factor'):
                    left_sequence.speed_factor = original_sequence.speed_factor
                if hasattr(right_sequence, 'speed_factor'):
                    right_sequence.speed_factor = original_sequence.speed_factor
        
        # Remove the original sequence
        sequences.remove(original_sequence)
        
        # Auto-fit sequencer view to show all sequences
        try:
            fit_sequencer_view()
        except:
            pass  # Silently ignore fit errors
        
        # Get info for the created sequences
        left_info = None
        right_info = None
        
        if left_sequence:
            left_info = {
                "name": left_sequence.name,
                "type": left_sequence.type,
                "channel": left_sequence.channel,
                "frame_start": left_sequence.frame_start,
                "frame_end": left_sequence.frame_final_end,
                "duration": left_sequence.frame_final_duration,
                "filepath": _get_sequence_filepath(left_sequence)
            }
        
        if right_sequence:
            right_info = {
                "name": right_sequence.name,
                "type": right_sequence.type,
                "channel": right_sequence.channel,
                "frame_start": right_sequence.frame_start,
                "frame_end": right_sequence.frame_final_end,
                "duration": right_sequence.frame_final_duration,
                "filepath": _get_sequence_filepath(right_sequence)
            }
        
        return {
            "success": True,
            "message": f"Successfully cut sequence '{sequence_name}' at frame {cut_frame} into '{left_name}' and '{right_name}'",
            "original_sequence": original_info,
            "left_sequence": left_info,
            "right_sequence": right_info
        }
        
    except Exception as e:
        return {
            "success": False,
            "message": f"Failed to cut sequence '{sequence_name}': {str(e)}",
            "original_sequence": original_info,
            "left_sequence": None,
            "right_sequence": None
        }


def detach_audio_from_video(
    video_sequence_name: str,
    audio_channel: int,
    audio_name: Optional[str] = None
) -> Dict[str, Any]:
    """
    Detach audio from a video sequence and create a separate audio sequence.
    
    This function extracts the audio track from a video sequence and creates
    a new audio sequence on the specified channel, allowing independent editing
    of video and audio components.
    
    Args:
        video_sequence_name: Name of the video sequence to detach audio from
        audio_channel: Channel number to place the detached audio sequence
        audio_name: Optional name for the audio sequence (auto-generated if None)
        
    Returns:
        Dict[str, Any]: Dictionary containing:
            - success: bool - Whether the operation succeeded
            - message: str - Success/error message
            - video_sequence: Dict[str, Any] - Original video sequence info
            - audio_sequence: Optional[Dict[str, Any]] - Created audio sequence info (None if no audio)
            
    Raises:
        ImportError: If bpy module is not available
        ValueError: If sequence is not found, wrong type, or parameters are invalid
        AttributeError: If sequence editor is not available
    """
    import bpy
    
    scene = bpy.context.scene
    
    if not scene.sequence_editor:
        raise AttributeError("No sequence editor found - timeline is empty or not initialized")
    
    sequences = scene.sequence_editor.sequences
    
    # Find the video sequence by name
    video_sequence = None
    for seq in sequences:
        if seq.name == video_sequence_name:
            video_sequence = seq
            break
    
    if video_sequence is None:
        raise ValueError(f"Sequence '{video_sequence_name}' not found")
    
    # Check if sequence type is supported
    if video_sequence.type != 'MOVIE':
        raise ValueError(
            f"Sequence '{video_sequence_name}' has type '{video_sequence.type}'. "
            f"This function only supports MOVIE sequences."
        )
    
    # Validate audio channel
    if audio_channel < 1:
        raise ValueError(f"Audio channel must be >= 1, got: {audio_channel}")
    
    # Get video file path
    video_filepath = _get_sequence_filepath(video_sequence)
    if video_filepath is None:
        raise ValueError(f"Cannot detach audio from '{video_sequence_name}': video filepath not found")
    
    # Generate audio sequence name if not provided
    if audio_name is None:
        base_name = video_sequence.name
        audio_name = f"{base_name}_audio"
        
        # Find next available audio name if conflict exists
        counter = 1
        while any(seq.name == audio_name for seq in sequences):
            counter += 1
            audio_name = f"{base_name}_audio{counter:02d}" if counter > 1 else f"{base_name}_audio"
    
    try:
        # Create audio sequence from the same video file
        audio_sequence = sequences.new_sound(
            name=audio_name,
            filepath=video_filepath,
            channel=audio_channel,
            frame_start=video_sequence.frame_start
        )
        
        # Match the audio duration to the video sequence duration
        video_duration = video_sequence.frame_final_duration
        audio_duration = audio_sequence.frame_final_duration
        
        # Trim audio to match video duration if needed
        if audio_duration > video_duration:
            audio_sequence.frame_final_end = video_sequence.frame_final_end
        elif audio_duration < video_duration:
            # If audio is shorter than video, it means video has no audio or partial audio
            # Keep the audio as is, but note this in the response
            pass
        
        # Auto-fit sequencer view to show all sequences
        try:
            fit_sequencer_view()
        except:
            pass  # Silently ignore fit errors
        
        # Get video sequence info
        video_info = {
            "name": video_sequence.name,
            "type": video_sequence.type,
            "channel": video_sequence.channel,
            "frame_start": video_sequence.frame_start,
            "frame_end": video_sequence.frame_final_end,
            "duration": video_sequence.frame_final_duration,
            "filepath": video_filepath
        }
        
        # Get audio sequence info
        audio_info = {
            "name": audio_sequence.name,
            "type": audio_sequence.type,
            "channel": audio_sequence.channel,
            "frame_start": audio_sequence.frame_start,
            "frame_end": audio_sequence.frame_final_end,
            "duration": audio_sequence.frame_final_duration,
            "filepath": _get_sequence_filepath(audio_sequence)
        }
        
        return {
            "success": True,
            "message": f"Successfully detached audio from '{video_sequence_name}' to '{audio_name}'",
            "video_sequence": video_info,
            "audio_sequence": audio_info
        }
        
    except Exception as e:
        # Check if the error is due to no audio track in the video
        error_msg = str(e).lower()
        if "sound" in error_msg or "audio" in error_msg or "track" in error_msg:
            return {
                "success": False,
                "message": f"Video '{video_sequence_name}' contains no audio track to detach",
                "video_sequence": {
                    "name": video_sequence.name,
                    "type": video_sequence.type,
                    "channel": video_sequence.channel,
                    "frame_start": video_sequence.frame_start,
                    "frame_end": video_sequence.frame_final_end,
                    "duration": video_sequence.frame_final_duration,
                    "filepath": video_filepath
                },
                "audio_sequence": None
            }
        else:
            raise ValueError(f"Failed to detach audio from '{video_sequence_name}': {str(e)}")
