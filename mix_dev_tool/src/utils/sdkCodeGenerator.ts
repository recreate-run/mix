/**
 * Utility to generate SDK code snippets for TypeScript and Python
 * Shows users how to reproduce their message using the Mix SDK
 */

import type { Attachment } from "@/stores/attachmentSlice";

interface CodeGeneratorParams {
	sessionId: string;
	message: string;
	attachments?: Attachment[];
	serverUrl?: string;
}

/**
 * Generate TypeScript SDK code for sending a message
 */
export function generateTypeScriptCode({
	sessionId,
	message,
	attachments = [],
	serverUrl = "http://localhost:8088",
}: CodeGeneratorParams): string {
	const hasAttachments = attachments.length > 0;

	// Escape special characters in message for template literal
	const escapedMessage = message.replace(/`/g, "\\`").replace(/\$/g, "\\$");

	let code = `import { Mix } from 'mix-typescript-sdk';

const mix = new Mix({
  serverURL: '${serverUrl}'
});

// Create or use existing session
const session = await mix.sessions.create({
  title: 'My Session'
});
`;

	// Add file upload code if there are attachments
	if (hasAttachments) {
		code += `
// Upload files
`;
		for (const attachment of attachments) {
			code += `const file${attachments.indexOf(attachment)} = await mix.sessions.files.upload({
  sessionId: session.id,
  file: yourFile // File object or path
});
`;
		}
		code += "\n";
	}

	// Add message sending code
	code += `// Send message
const stream = await mix.sessions.sendMessage({
  sessionId: '${sessionId}',
  requestBody: {
    message: \`${escapedMessage}\`,
    stream: true
  }
});

// Handle streaming events
for await (const event of stream) {
  switch (event.type) {
    case 'content':
      console.log(event.content);
      break;
    case 'tool':
      console.log('Tool:', event.toolName);
      break;
    case 'complete':
      console.log('Complete!');
      break;
  }
}`;

	return code;
}

/**
 * Generate Python SDK code for sending a message
 */
export function generatePythonCode({
	sessionId,
	message,
	attachments = [],
	serverUrl = "http://localhost:8088",
}: CodeGeneratorParams): string {
	const hasAttachments = attachments.length > 0;

	// Escape special characters in message for Python string
	const escapedMessage = message
		.replace(/\\/g, "\\\\")
		.replace(/'/g, "\\'")
		.replace(/"/g, '\\"');

	let code = `import asyncio
from mix_python_sdk import Mix

async def main():
    async with Mix(server_url='${serverUrl}') as mix:
        # Create or use existing session
        session = mix.sessions.create(title='My Session')
`;

	// Add file upload code if there are attachments
	if (hasAttachments) {
		code += `
        # Upload files
`;
		for (const attachment of attachments) {
			code += `        file${attachments.indexOf(attachment)} = mix.sessions.files.upload(
            session_id=session.id,
            file=your_file  # File path or file object
        )
`;
		}
		code += "\n";
	}

	// Add message sending code
	code += `        # Send message
        stream = mix.sessions.send_message(
            session_id='${sessionId}',
            message='${escapedMessage}',
            stream=True
        )

        # Handle streaming events
        async for event in stream:
            if event.type == 'content':
                print(event.content, end='', flush=True)
            elif event.type == 'tool':
                print(f'Tool: {event.tool_name}')
            elif event.type == 'complete':
                print('Complete!')

if __name__ == '__main__':
    asyncio.run(main())`;

	return code;
}
