#!/usr/bin/env python3
"""
Script to automatically update __init__.py with all public functions from module files.
"""

import ast
import os
from pathlib import Path
from typing import Dict, List, Set


def extract_public_functions(file_path: Path) -> Set[str]:
    """Extract public functions from a Python file."""
    try:
        with open(file_path, 'r') as f:
            content = f.read()
        
        tree = ast.parse(content)
        
        # First check for __all__ definition
        for node in ast.walk(tree):
            if isinstance(node, ast.Assign):
                for target in node.targets:
                    if isinstance(target, ast.Name) and target.id == '__all__':
                        if isinstance(node.value, ast.List):
                            return {
                                elt.s if hasattr(elt, 's') else elt.value 
                                for elt in node.value.elts
                                if hasattr(elt, 's') or hasattr(elt, 'value')
                            }
        
        # Fallback: extract public function definitions (not starting with _)
        functions = set()
        for node in ast.walk(tree):
            if isinstance(node, ast.FunctionDef) and not node.name.startswith('_'):
                functions.add(node.name)
        
        return functions
    
    except Exception as e:
        print(f"Error parsing {file_path}: {e}")
        return set()


def generate_init_file(module_dir: Path) -> str:
    """Generate __init__.py content for a module directory."""
    imports = []
    
    # Find all Python files (excluding __init__.py)
    for py_file in module_dir.glob("*.py"):
        if py_file.name == "__init__.py":
            continue
        
        module_name = py_file.stem
        functions = extract_public_functions(py_file)
        
        if functions:
            # Sort functions for consistent output
            sorted_functions = sorted(functions)
            
            # Format imports nicely
            import_lines = [f"from .{module_name} import ("]
            for i, func in enumerate(sorted_functions):
                if i == len(sorted_functions) - 1:
                    import_lines.append(f"    {func}")
                else:
                    import_lines.append(f"    {func},")
            import_lines.append(")")
            
            imports.append("\n".join(import_lines))
    
    # Generate full __init__.py content
    content = '"""\nBlender integration tools.\n"""\n\n'
    content += "\n".join(imports)
    
    return content


def main():
    """Main function to update __init__.py."""
    script_dir = Path(__file__).parent
    
    # Generate new content
    new_content = generate_init_file(script_dir)
    
    # Write to __init__.py
    init_file = script_dir / "__init__.py"
    with open(init_file, 'w') as f:
        f.write(new_content)
    
    print(f"Updated {init_file}")


if __name__ == "__main__":
    main()