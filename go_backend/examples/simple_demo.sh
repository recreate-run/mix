#!/bin/bash

# Simple Agentic Loop Demonstration
# This script shows how the AI agent iteratively calls tools to complete tasks

echo "=== Agentic Loop Demo ==="
echo "This demonstrates how the AI agent automatically:"
echo "1. Analyzes the task"
echo "2. Selects and calls appropriate tools"
echo "3. Processes tool results"
echo "4. Continues until the task is complete"
echo ""

# Demo 1: File creation task (should trigger Write tool)
echo "🔄 Demo 1: Creating a simple file"
echo "Task: 'Create a hello.txt file with Hello World message'"
echo ""

./mix -p "Create a hello.txt file with the message 'Hello World from the agentic loop!'"

echo ""
echo "✅ Demo 1 complete. Check if hello.txt was created:"
if [ -f "hello.txt" ]; then
    echo "📄 File created successfully:"
    cat hello.txt
else
    echo "❌ File was not created"
fi

echo ""
echo "=== End of Demo ==="