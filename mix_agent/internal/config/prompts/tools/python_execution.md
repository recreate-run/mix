Executes Python code in a secure, isolated sandbox environment with dependency management and safety controls.

WHEN TO USE THIS TOOL:
- Use when you need to run Python code for calculations, data processing, or testing
- Perfect for mathematical computations, data analysis, and algorithm verification
- Helpful for validating code logic before implementing in the main application

HOW TO USE:
- Provide Python code as the input parameter
- The tool creates an ephemeral virtual environment for each execution
- Code runs in complete isolation with no persistence between executions
- Results include stdout, stderr, and return code for comprehensive feedback

SECURITY FEATURES:
- Complete isolation through UV's ephemeral virtual environments
- No state persistence between executions prevents interference
- Restricted system access and blocked dangerous operations
- Timeout protection prevents infinite loops (30 second default, 2 minutes maximum)
- Output truncation prevents memory exhaustion
- Code validation blocks potentially unsafe patterns

AVAILABLE PACKAGES:
- numpy: Pre-installed for mathematical computations
- All Python standard library modules
- Additional packages can be installed during execution (when needed)

CRITICAL REQUIREMENTS:
1. CODE SAFETY: Avoid using subprocess, exec, eval, or similar dangerous functions
2. TIMEOUT LIMITS: Keep execution time under 30 seconds for optimal performance
3. OUTPUT SIZE: Large outputs will be truncated at 30,000 characters
4. NO PERSISTENCE: Files and variables do not persist between executions

LIMITATIONS:
- No network access during execution
- No file system persistence between runs
- Cannot access external resources or make API calls
- Memory and CPU usage is limited by the sandbox environment

EXAMPLES:

<example>
# Basic calculation
result = 2 + 2
print(f"2 + 2 = {result}")
</example>

<example>
# NumPy operations
import numpy as np
data = np.array([1, 2, 3, 4, 5])
mean = np.mean(data)
std = np.std(data)
print(f"Mean: {mean}, Std: {std}")
</example>

ERROR HANDLING:
- Syntax errors, runtime exceptions, and import errors are captured in stderr
- Return code indicates success (0) or failure (non-zero)
- Timeout errors are reported with appropriate error messages
- All errors are safely contained within the sandbox environment

TIPS:
- Use this tool for complex calculations that would be difficult to do manually
- Test algorithms and data transformations before implementing in your codebase
- Combine with other tools to validate results or process data for further use
- Keep code simple and focused for best performance and clarity