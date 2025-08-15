import { source } from '@/lib/source';
import {
  PageRoot,
  PageArticle,
  PageTOC,
  PageTOCItems,
  PageTOCPopover,
  PageTOCPopoverContent,
  PageTOCPopoverItems,
  PageTOCPopoverTrigger,
  PageTOCTitle,
} from 'fumadocs-ui/layouts/docs/page';
import { notFound } from 'next/navigation';
import { createRelativeLink } from 'fumadocs-ui/mdx';
import { getMDXComponents } from '@/mdx-components';

export default async function Page(props: {
  params: Promise<{ slug?: string[] }>;
}) {
  const params = await props.params;
  const page = source.getPage(params.slug);
  if (!page) notFound();

  const MDXContent = page.data.body;
  const toc = page.data.toc;

  return (
    <PageRoot
      toc={{
        toc,
        single: false,
      }}
    >
      {toc.length > 0 && (
        <PageTOCPopover>
          <PageTOCPopoverTrigger />
          <PageTOCPopoverContent>
            <PageTOCPopoverItems />
          </PageTOCPopoverContent>
        </PageTOCPopover>
      )}
      <PageArticle>
        <h1 className="text-3xl font-semibold">{page.data.title}</h1>
        <p className="text-lg text-fd-muted-foreground">
          {page.data.description}
        </p>
        <div className="prose flex-1 text-fd-foreground/80">
          <MDXContent
            components={getMDXComponents({
              // this allows you to link to other pages with relative file paths
              a: createRelativeLink(source, page),
            })}
          />
        </div>
      </PageArticle>
      {toc.length > 0 && (
        <PageTOC>
          <PageTOCTitle />
          <PageTOCItems />
        </PageTOC>
      )}
    </PageRoot>
  );
}

export async function generateStaticParams() {
  return source.generateParams();
}

export async function generateMetadata(props: {
  params: Promise<{ slug?: string[] }>;
}) {
  const params = await props.params;
  const page = source.getPage(params.slug);
  if (!page) notFound();

  return {
    title: page.data.title,
    description: page.data.description,
  };
}
