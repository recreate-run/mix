import defaultMdxComponents from 'fumadocs-ui/mdx';
import { Mermaid } from '@/components/mdx/mermaid';
import { APIPage } from 'fumadocs-openapi/ui';
import { openapi } from '@/lib/openapi';
import type { MDXComponents } from 'mdx/types';

// use this function to get MDX components, you will need it for rendering MDX
export function getMDXComponents(components?: MDXComponents): MDXComponents {
  return {
    ...defaultMdxComponents,
    Mermaid,
    APIPage: (props: any) => (
      <APIPage {...openapi.getAPIPageProps(props)}
      />
    ),
    ...components,
  };
}
