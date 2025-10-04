'use client';

import { ReactNode, useState } from 'react';
import Link from 'next/link';
import { Check, Copy, ExternalLink, BookOpen } from 'lucide-react';

interface DemoShowcaseProps {
  icon: string;
  title: string;
  description: string;
  fileName: string;
  code: string;
  videoSrc: string;
  videoCaption?: string;
  githubUrl?: string;
}

export function DemoShowcase({
  icon,
  title,
  description,
  fileName,
  code,
  videoSrc,
  videoCaption,
  githubUrl
}: DemoShowcaseProps) {
  const [copied, setCopied] = useState(false);

  const copyToClipboard = async () => {
    try {
      await navigator.clipboard.writeText(code);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      console.error('Failed to copy:', err);
    }
  };

  return (
    <div className="grid grid-cols-1 lg:grid-cols-2 gap-12 items-start max-w-7xl mx-auto">
      <div className="space-y-4 flex flex-col">
        <div className="flex items-center gap-3 mb-6">
          <div className="h-10 w-10 rounded-lg bg-fd-primary/10 flex items-center justify-center">
            <span className="text-xl">{icon}</span>
          </div>
          <div>
            <h3 className="text-xl font-semibold">{title}</h3>
            <p className="text-sm text-fd-muted-foreground">{description}</p>
          </div>
        </div>

        <div className="rounded-xl border bg-fd-card overflow-hidden shadow-lg">
          <div className="bg-fd-muted/50 px-4 py-2 border-b flex items-center justify-between">
            <div className="flex items-center gap-2">
              <div className="flex gap-1.5">
                <div className="h-3 w-3 rounded-full bg-red-500/20"></div>
                <div className="h-3 w-3 rounded-full bg-yellow-500/20"></div>
                <div className="h-3 w-3 rounded-full bg-green-500/20"></div>
              </div>
              <span className="text-xs text-fd-muted-foreground ml-2">{fileName}</span>
            </div>
            <button
              onClick={copyToClipboard}
              className="p-1.5 rounded-md hover:bg-fd-muted transition-colors text-fd-muted-foreground hover:text-fd-foreground"
              title="Copy code"
            >
              {copied ? (
                <Check className="h-4 w-4 text-green-500" />
              ) : (
                <Copy className="h-4 w-4" />
              )}
            </button>
          </div>
          <pre className="p-6 overflow-x-auto text-sm leading-relaxed">
            <code className="language-python">{code}</code>
          </pre>
        </div>
      </div>

      <div className="lg:sticky lg:top-8 flex flex-col">
        <div className="rounded-xl overflow-hidden border shadow-2xl bg-fd-card">
          <video
            src={videoSrc}
            controls
            className="w-full h-auto"
            poster={videoSrc}
          >
            Your browser does not support the video tag.
          </video>
        </div>
        {videoCaption && (
          <p className="text-sm text-fd-muted-foreground mt-4 text-center">
            {videoCaption}
          </p>
        )}
        {githubUrl && (
          <div className="mt-6 text-center">
            <Link
              href={githubUrl}
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-2 px-4 py-2 text-sm border rounded-lg hover:bg-fd-muted transition-colors"
            >
              <ExternalLink className="h-4 w-4" />
              View on GitHub
            </Link>
          </div>
        )}
      </div>
    </div>
  );
}

interface DemoSectionProps {
  title?: string;
  subtitle?: string;
  children: ReactNode;
}

export function DemoSection({
  title = "See It In Action",
  subtitle = "Watch real-world examples of Mix in action",
  children
}: DemoSectionProps) {
  return (
    <div className="container px-4 py-16 bg-gradient-to-b from-fd-background to-fd-muted/30">
      <div className="text-center mb-12">
        <h2 className="text-3xl font-bold mb-3">{title}</h2>
        <p className="text-fd-muted-foreground max-w-2xl mx-auto">
          {subtitle}
        </p>
      </div>
      <div className="space-y-24">
        {children}
      </div>

      {/* Try Examples CTA */}
      <div className="max-w-4xl mx-auto mt-24">
        <div className="relative rounded-2xl border bg-gradient-to-br from-fd-primary/5 to-fd-primary/10 p-8 overflow-hidden">
          <div className="relative z-10">
            <div className="flex items-center gap-3 mb-4">
              <BookOpen className="h-6 w-6 text-fd-primary" />
              <h3 className="text-2xl font-bold">Try These Examples</h3>
            </div>
            <p className="text-fd-muted-foreground mb-6 max-w-2xl">
              Explore more real-world examples and build your own AI workflows. Our cookbooks include portfolio analysis, video processing, web search, and more.
            </p>
            <Link
              href="https://github.com/recreate-run/mix-cookbooks"
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-2 px-6 py-3 bg-fd-primary text-fd-primary-foreground rounded-lg hover:bg-fd-primary/90 transition-colors font-medium"
            >
              View Cookbook Examples
              <ExternalLink className="h-4 w-4" />
            </Link>
          </div>
          <div className="absolute top-0 right-0 -mt-4 -mr-4 h-32 w-32 rounded-full bg-fd-primary/10 blur-3xl"></div>
          <div className="absolute bottom-0 left-0 -mb-4 -ml-4 h-32 w-32 rounded-full bg-fd-primary/10 blur-3xl"></div>
        </div>
      </div>
    </div>
  );
}
