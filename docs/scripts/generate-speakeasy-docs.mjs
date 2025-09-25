import { generateFiles } from 'fumadocs-openapi';
import fs from 'fs/promises';
import path from 'path';

async function main() {
  try {
    console.log('Fetching OpenAPI spec from Speakeasy...');
    
    // Fetch the OpenAPI spec from Speakeasy (returns YAML format)
    const response = await fetch('https://spec.speakeasy.com/recreate/mix/mix-rest-api-with-code-samples', {
      headers: {
        'Accept': 'application/json'
      }
    });
    
    if (!response.ok) {
      throw new Error(`Failed to fetch OpenAPI spec: ${response.status}`);
    }
    
    const yamlContent = await response.text();
    console.log('OpenAPI spec fetched successfully');
    
    // Save the spec to a permanent file for fumadocs-openapi to process
    const specPath = path.join(process.cwd(), 'lib', 'mix-api-spec.yaml');
    await fs.writeFile(specPath, yamlContent);
    
    // Generate files using Fumadocs OpenAPI
    console.log('Generating API documentation files...');
    
    await generateFiles({
      input: ['./lib/mix-api-spec.yaml'],
      output: './content/docs/api',
      // Generate one page per operation, grouped by tag
      per: 'operation',
      groupBy: 'tag',
      // Generate index files with cards linking to generated pages
      index: {
        url: {
          baseUrl: '/docs',
          contentDir: './content/docs',
        },
        items: [
          {
            path: 'index.mdx',
            description: 'Complete API reference for Mix',
          },
        ],
      },
      // Note: imports are causing character-by-character generation issues
      // We'll handle imports manually in the index file if needed
      // Add auto-generated comment
      addGeneratedComment: '<!-- This file is auto-generated. Do not edit manually. -->',
      // Include descriptions from OpenAPI spec
      includeDescription: true,
      // Custom frontmatter for better organization
      frontmatter: (title, description, operation) => ({
        title,
        description,
        full: true,
        // Add method and route for proper HTTP method badge styling
        ...(operation && {
          method: operation.method?.toUpperCase(),
          route: operation.path,
        }),
      }),
    });
    
    // Create meta.json for the API section
    const apiDir = path.join(process.cwd(), 'content', 'docs', 'api');
    const metaContent = {
      title: "API Reference",
      description: "Complete API reference for Mix - session management, messaging, and system operations",
      icon: "Terminal",
      root: true
    };
    
    await fs.writeFile(
      path.join(apiDir, 'meta.json'),
      JSON.stringify(metaContent, null, 2)
    );
    
    console.log('API documentation generated successfully!');
    
  } catch (error) {
    console.error('Error generating API docs:', error);
    process.exit(1);
  }
}

main();