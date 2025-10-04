'use client';

import { useState } from 'react';
import Link from 'next/link';
import { Demo } from '@/lib/demos';
import { Check, Copy, ExternalLink, BookOpen } from 'lucide-react';

interface DemoCardsProps {
  demos: Demo[];
}

export function DemoCards({ demos }: DemoCardsProps) {
  const [selectedDemo, setSelectedDemo] = useState(0);
  const [copied, setCopied] = useState(false);

  const copyToClipboard = async () => {
    try {
      await navigator.clipboard.writeText(demos[selectedDemo].code);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      console.error('Failed to copy:', err);
    }
  };

  if (demos.length === 0) return null;

  return (
    <div className="container px-4 py-16">
      <div className="text-center mb-12">
        <h2 className="text-3xl font-bold mb-3">See It In Action</h2>
        <p className="text-fd-muted-foreground max-w-2xl mx-auto">
          Explore real-world examples of Mix SDK capabilities
        </p>
      </div>

      {/* Demo selector cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 mb-12 max-w-5xl mx-auto">
        {demos.map((demo, index) => (
          <button
            key={index}
            onClick={() => setSelectedDemo(index)}
            className={`p-4 rounded-lg border text-left transition-all ${
              selectedDemo === index
                ? 'border-fd-primary bg-fd-primary/5 shadow-md'
                : 'border-fd-border hover:border-fd-primary/50'
            }`}
          >
            <div className="flex items-center gap-3 mb-2">
              <span className="text-2xl">{demo.icon}</span>
              <h3 className="font-semibold">{demo.title}</h3>
            </div>
            <p className="text-sm text-fd-muted-foreground">{demo.description}</p>
          </button>
        ))}
      </div>

      {/* Selected demo showcase */}
      <div className="max-w-7xl mx-auto">
        <div className="grid grid-cols-1 lg:grid-cols-5 gap-8 items-start">
          {/* Code section - takes 2 columns */}
          <div className="lg:col-span-2 space-y-4">
            <div className="rounded-xl border bg-fd-card overflow-hidden shadow-lg">
              <div className="bg-fd-muted/50 px-4 py-2 border-b flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <div className="flex gap-1.5">
                    <div className="h-3 w-3 rounded-full bg-red-500/20"></div>
                    <div className="h-3 w-3 rounded-full bg-yellow-500/20"></div>
                    <div className="h-3 w-3 rounded-full bg-green-500/20"></div>
                  </div>
                  <span className="text-xs text-fd-muted-foreground ml-2">
                    {demos[selectedDemo].fileName}
                  </span>
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
              <pre className="p-6 overflow-x-auto text-sm leading-relaxed max-h-[600px] overflow-y-auto">
                <code className="language-python">{demos[selectedDemo].code}</code>
              </pre>
            </div>
          </div>

          {/* Video section - takes 3 columns */}
          <div className="lg:col-span-3 lg:sticky lg:top-8">
            <div className="rounded-2xl overflow-hidden border-2 border-fd-primary/20 shadow-2xl bg-gradient-to-br from-fd-card to-fd-muted/30 p-1">
              <div className="rounded-xl overflow-hidden bg-black aspect-video">
                <video
                  key={selectedDemo} // Force re-render when demo changes
                  src={demos[selectedDemo].videoSrc}
                  controls
                  className="w-full h-full object-contain"
                  poster={demos[selectedDemo].videoSrc}
                >
                  Your browser does not support the video tag.
                </video>
              </div>
            </div>
            {demos[selectedDemo].videoCaption && (
              <p className="text-sm text-fd-muted-foreground mt-6 text-center font-medium">
                {demos[selectedDemo].videoCaption}
              </p>
            )}
          </div>
        </div>
      </div>

      {/* Demo-specific link or general CTA */}
      <div className="max-w-7xl mx-auto mt-12">
        <div className="flex items-center justify-center gap-4 flex-wrap">
          {demos[selectedDemo].githubUrl && (
            <Link
              href={demos[selectedDemo].githubUrl!}
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-2 px-4 py-2 text-sm border rounded-lg hover:bg-fd-muted transition-colors"
            >
              <ExternalLink className="h-4 w-4" />
              Try on GitHub
            </Link>
          )}
          <Link
            href="https://github.com/recreate-run/mix-cookbooks"
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex items-center gap-2 px-4 py-2 text-sm border rounded-lg hover:bg-fd-muted transition-colors"
          >
            <BookOpen className="h-4 w-4" />
            Browse All Examples
          </Link>
        </div>
      </div>

      {/* Creative Freedom Message */}
      <div className="max-w-3xl mx-auto mt-20 text-center">
        <div className="inline-block px-3 py-1 rounded-full bg-fd-primary/10 text-fd-primary text-sm font-medium mb-4">
          Build Anything
        </div>
        <h3 className="text-2xl font-bold mb-4">
          These are just starting points
        </h3>
        <p className="text-fd-muted-foreground text-lg leading-relaxed">
          Mix is a flexible multimodal AI platform. Use your creativity to build automated video editing workflows,
          intelligent data analysis pipelines, content generation systems, or anything else you can imagine.
          The examples show what's possible—what you build is up to you.
        </p>
      </div>
    </div>
  );
}
