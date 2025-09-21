"""
Blender workspace utilities.
"""

from typing import List


def get_current_workspace() -> str:
    """
    Get the name of the current Blender workspace.
    
    Returns:
        str: The name of the current workspace
        
    Raises:
        ImportError: If bpy module is not available
        AttributeError: If workspace context is not available
    """
    import bpy
    
    if not hasattr(bpy.context, 'workspace'):
        raise AttributeError("Workspace context not available")
    
    return bpy.context.workspace.name
