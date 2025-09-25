import { createOpenAPI } from 'fumadocs-openapi/server';

export const openapi = createOpenAPI({
    input: ['./lib/mix-api-spec.yaml'],
  generateCodeSamples(endpoint) {
    return [
      // Disable all default language samples
      { lang: 'curl', label: 'cURL', source: false },
      { lang: 'javascript', label: 'JavaScript', source: false },
      { lang: 'go', label: 'Go', source: false },
      { lang: 'python', label: 'Python', source: false },
      { lang: 'java', label: 'Java', source: false },
      { lang: 'csharp', label: 'C#', source: false },
      
    ];
  },
});