'use client';

import { useEffect, useState } from 'react';
import { Suspense } from 'react';
import { CodeBlock } from 'fumadocs-ui/components/codeblock';
import { DynamicCodeBlock } from 'fumadocs-ui/components/dynamic-codeblock';

import * as yaml from 'js-yaml';

interface SpeakeaySample {
  title: string;
  description: string;
  code: string;
  language: string;
  path?: string;
  method?: string;
  operation?: string;
}

interface SpeakeaySamplesProps {
  language: 'typescript' | 'python';
  endpoint?: string;
  operation?: string;
}

// Parse YAML structure from Speakeasy using proper YAML parser
function parseYaml(yamlStr: string): any {
  try {
    const samples: SpeakeaySample[] = [];

    // Parse the YAML properly
    const data = yaml.load(yamlStr) as any;

    if (!data?.actions) {
      return samples;
    }

    for (const action of data.actions) {
      const targetMatch = action.target?.match(/\$\["paths"\]\["([^"]+)"\]\["([^"]+)"\]/);
      if (!targetMatch) continue;

      const path = targetMatch[1];
      const method = targetMatch[2];

      const codeSamples = action.update?.['x-codeSamples'];
      if (!codeSamples) continue;

      for (const sample of codeSamples) {
        if (sample.lang !== 'typescript' && sample.lang !== 'python') continue;

        const code = sample.source;
        if (!code || code.length < 10) continue;

        samples.push({
          title: `${method.toUpperCase()} ${path}`,
          description: `${method.toUpperCase()} ${path}`,
          code: code.trim(),
          language: sample.lang,
          path,
          method: method.toUpperCase(),
          operation: `${method}:${path}`
        });
      }
    }

    return samples;
  } catch (e) {
    console.error('Error parsing YAML:', e);
    return [];
  }
}

export function SpeakeaySamples({ language, endpoint, operation }: SpeakeaySamplesProps) {
  const [samples, setSamples] = useState<SpeakeaySample[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchSamples = async () => {
      try {
        setLoading(true);
        setError(null);

        // Create a URL for the specific language
        const url = `https://spec.speakeasy.com/recreate/mix/mix-rest-api-${language}-code-samples`;

        const response = await fetch(url);

        if (!response.ok) {
          throw new Error(`Failed to fetch samples: ${response.status}`);
        }

        // Get the raw text content since it's YAML
        const yamlContent = await response.text();

        // Parse the YAML content
        const processedSamples = parseYaml(yamlContent);

        // Filter by endpoint or operation if specified
        let filteredSamples = [...processedSamples];

        if (endpoint) {
          filteredSamples = filteredSamples.filter(
            sample => sample.path?.includes(endpoint)
          );
        }

        if (operation) {
          filteredSamples = filteredSamples.filter(
            sample => sample.operation === operation
          );
        }

        setSamples(filteredSamples);

      } catch (err) {
        console.error('Error fetching samples:', err);
        setError(err instanceof Error ? err.message : 'Unknown error');
      } finally {
        setLoading(false);
      }
    };

    fetchSamples();
  }, [language, endpoint, operation]);

  if (loading) {
    return <div className="text-muted-foreground">Loading code samples...</div>;
  }

  if (error) {
    return (
      <div className="border border-destructive/50 bg-destructive/10 rounded-md p-4">
        <p className="text-destructive text-sm">Error loading samples: {error}</p>
      </div>
    );
  }

  if (samples.length === 0) {
    return <div className="text-muted-foreground">No code samples found for the specified filters.</div>;
  }

  return (
    <div className="space-y-8">
      {samples.map((sample, index) => {
        // Create a clean ID from the path and method for navigation
        const headingId = sample.path && sample.method
          ? `${sample.method.toLowerCase()}-${sample.path.replace(/[^a-zA-Z0-9]/g, '-').toLowerCase()}`
          : `sample-${index}`;

        return (
          <div key={index}>
            {sample.path && sample.method && (
              <h3 id={headingId}>
                <span className={`inline-block px-2 py-1 rounded text-xs font-mono mr-2 ${sample.method === 'GET' ? 'bg-green-100 text-green-800' :
                    sample.method === 'POST' ? 'bg-blue-100 text-blue-800' :
                      sample.method === 'PUT' ? 'bg-yellow-100 text-yellow-800' :
                        sample.method === 'DELETE' ? 'bg-red-100 text-red-800' :
                          'bg-gray-100 text-gray-800'
                  }`}>
                  {sample.method}
                </span>
                <code>{sample.path}</code>
              </h3>
            )}


            <DynamicCodeBlock
              lang={sample.language}
              code={sample.code}
              options={{
                themes: {
                  light: 'github-light',
                  dark: 'github-dark',
                },
                components: {
                  // override components (e.g. `pre` and `code`)
                },
                // other Shiki options
              }}
            />

          </div>
        );
      })}
    </div>
  );
}

export default function SpeakeaySamplesWrapper(props: SpeakeaySamplesProps) {
  return (
    <Suspense fallback={<div>Loading code samples...</div>}>
      <SpeakeaySamples {...props} />
    </Suspense>
  );
}