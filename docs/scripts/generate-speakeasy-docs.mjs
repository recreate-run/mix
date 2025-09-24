import { generateFiles } from 'fumadocs-openapi';
import { rmSync, existsSync, mkdirSync } from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

// Get the directory of this script file
const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const projectRoot = path.join(__dirname, '..');

console.log('🚀 Generating API docs from Speakeasy registry...');

const outputPath = path.join(projectRoot, 'content/docs/api');
const speakeasyUrl = 'https://spec.speakeasy.com/recreate/mix/mix-rest-api';

console.log('📍 Input URL:', speakeasyUrl);
console.log('📁 Output path:', outputPath);

// Ensure parent directories exist
mkdirSync(path.dirname(outputPath), { recursive: true });

// Clean existing generated API docs
try {
  rmSync(outputPath, { recursive: true, force: true });
  console.log('🧹 Cleaned existing API docs');
} catch (error) {
  console.log('ℹ️ No existing API docs to clean');
}

// Generate API documentation from Speakeasy's enhanced OpenAPI spec
try {
  console.log('📥 Fetching enhanced spec with code samples...');
  
  await generateFiles({
    input: [speakeasyUrl],
    output: outputPath,
    per: 'tag',  // Group by API tags (Authentication, Sessions, etc.)
    frontmatter: (title, description) => ({
      title: title,
      description: description,
      full: true,  // Use full width for API docs
    }),
  });
  
  console.log('✅ Generation completed successfully!');
  
  if (existsSync(outputPath)) {
    console.log('📚 API documentation created at:', outputPath);
    
    // List generated files
    const { readdirSync } = await import('fs');
    const files = readdirSync(outputPath, { recursive: true });
    console.log('📄 Generated files:');
    files.forEach(file => console.log(`   - ${file}`));
  }
  
} catch (error) {
  console.error('❌ Generation failed:', error.message);
  process.exit(1);
}