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
    <div className="container px-4 pb-8">
      {/* Selected demo showcase */}
      <div className="max-w-7xl mx-auto">
        <div className="grid grid-cols-1 lg:grid-cols-5 gap-8 items-start">
          {/* Code section - takes 2 columns */}
          <div className="lg:col-span-2 space-y-4">
            <div className="rounded-2xl border-2 border-fd-border/50 bg-gradient-to-br from-fd-card to-fd-muted/20 overflow-hidden shadow-xl">
              <div className="bg-gradient-to-r from-fd-muted/80 to-fd-muted/60 backdrop-blur-sm px-5 py-3 border-b border-fd-border/50 flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <div className="flex gap-2">
                    <div className="h-3 w-3 rounded-full bg-red-500/30 ring-1 ring-red-500/20"></div>
                    <div className="h-3 w-3 rounded-full bg-yellow-500/30 ring-1 ring-yellow-500/20"></div>
                    <div className="h-3 w-3 rounded-full bg-green-500/30 ring-1 ring-green-500/20"></div>
                  </div>
                  <div className="h-4 w-px bg-fd-border/50" />
                  <span className="text-xs font-medium text-fd-muted-foreground">
                    {demos[selectedDemo].fileName}
                  </span>
                </div>
                <button
                  onClick={copyToClipboard}
                  className="p-2 rounded-lg hover:bg-fd-muted/80 transition-all text-fd-muted-foreground hover:text-fd-foreground hover:scale-105 active:scale-95"
                  title="Copy code"
                >
                  {copied ? (
                    <Check className="h-4 w-4 text-green-500" />
                  ) : (
                    <Copy className="h-4 w-4" />
                  )}
                </button>
              </div>
              <pre className="p-6 overflow-x-auto text-sm leading-relaxed max-h-[600px] overflow-y-auto bg-gradient-to-b from-transparent to-fd-muted/10">
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
              className="group inline-flex items-center gap-2 px-5 py-2.5 text-sm font-medium border-2 border-fd-primary/30 rounded-xl hover:bg-fd-primary/10 hover:border-fd-primary transition-all hover:shadow-lg hover:shadow-fd-primary/20 hover:scale-105"
            >
              <ExternalLink className="h-4 w-4 group-hover:rotate-12 transition-transform" />
              Try on GitHub
            </Link>
          )}
          <Link
            href="https://github.com/recreate-run/mix-cookbooks"
            target="_blank"
            rel="noopener noreferrer"
            className="group inline-flex items-center gap-2 px-5 py-2.5 text-sm font-medium border-2 border-fd-border/50 rounded-xl hover:bg-fd-muted hover:border-fd-border transition-all hover:shadow-lg hover:scale-105"
          >
            <BookOpen className="h-4 w-4 group-hover:scale-110 transition-transform" />
            Browse All Examples
          </Link>
        </div>
      </div>

      {/* Demo selector cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 mt-12 max-w-6xl mx-auto">
        {demos.map((demo, index) => (
          <button
            key={index}
            onClick={() => setSelectedDemo(index)}
            className={`group relative p-6 rounded-2xl text-left transition-all duration-300 ${
              selectedDemo === index
                ? 'bg-gradient-to-br from-fd-primary/10 via-fd-primary/5 to-transparent border-2 border-fd-primary shadow-lg shadow-fd-primary/20 scale-[1.02]'
                : 'bg-fd-card border-2 border-fd-border/50 hover:border-fd-primary/30 hover:shadow-lg hover:scale-[1.01]'
            }`}
          >
            {selectedDemo === index && (
              <div className="absolute inset-0 bg-gradient-to-br from-fd-primary/5 to-transparent rounded-2xl blur-xl -z-10" />
            )}
            <div>
              <h3 className={`font-bold text-base mb-2 transition-colors ${
                selectedDemo === index ? 'text-fd-primary' : 'text-fd-foreground'
              }`}>
                {demo.title}
              </h3>
              <p className="text-sm text-fd-muted-foreground leading-relaxed">
                {demo.description}
              </p>
            </div>
          </button>
        ))}
      </div>
    </div>
  );
}
