import { NextRequest, NextResponse } from 'next/server';
import { readFile } from 'fs/promises';
import { join } from 'path';

export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ path: string[] }> }
) {
  try {
    const { path } = await params;
    let filePath = decodeURIComponent(path.join('/'));

    // If path already includes extension (.mdx or .md), use it directly
    // Otherwise, try adding .mdx then .md
    const hasExtension = filePath.endsWith('.mdx') || filePath.endsWith('.md');

    if (hasExtension) {
      const fullPath = join(process.cwd(), 'content/docs', filePath);
      const markdown = await readFile(fullPath, 'utf-8');
      return new NextResponse(markdown, {
        headers: {
          'Content-Type': 'text/plain',
        },
      });
    }

    // Try with .mdx extension first, then .md
    try {
      const fullPath = join(process.cwd(), 'content/docs', `${filePath}.mdx`);
      const markdown = await readFile(fullPath, 'utf-8');
      return new NextResponse(markdown, {
        headers: {
          'Content-Type': 'text/plain',
        },
      });
    } catch {
      const fullPath = join(process.cwd(), 'content/docs', `${filePath}.md`);
      const markdown = await readFile(fullPath, 'utf-8');
      return new NextResponse(markdown, {
        headers: {
          'Content-Type': 'text/plain',
        },
      });
    }
  } catch (error) {
    console.error('Error reading markdown file:', error);
    return new NextResponse(JSON.stringify({ error: 'File not found', path: (await params).path.join('/') }), {
      status: 404,
      headers: {
        'Content-Type': 'application/json',
      },
    });
  }
}
