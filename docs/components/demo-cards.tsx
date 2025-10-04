'use client';

import { useState } from 'react';
import { Demo } from '@/lib/demos';

interface DemoCardsProps {
  demos: Demo[];
}

export function DemoCards({ demos }: DemoCardsProps) {
  const [selectedDemo, setSelectedDemo] = useState(0);

  if (demos.length === 0) return null;

  return (
    <div className="container px-4 py-16">
      <div className="text-center mb-12">
        <h2 className="text-3xl font-bold mb-3">See It In Action</h2>
        <p className="text-fd-muted-foreground max-w-2xl mx-auto">
          Explore real-world examples of Mix multimodal capabilities
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
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-12 items-start">
          {/* Code section */}
          <div className="space-y-4">
            <div className="rounded-xl border bg-fd-card overflow-hidden shadow-lg">
              <div className="bg-fd-muted/50 px-4 py-2 border-b flex items-center gap-2">
                <div className="flex gap-1.5">
                  <div className="h-3 w-3 rounded-full bg-red-500/20"></div>
                  <div className="h-3 w-3 rounded-full bg-yellow-500/20"></div>
                  <div className="h-3 w-3 rounded-full bg-green-500/20"></div>
                </div>
                <span className="text-xs text-fd-muted-foreground ml-2">
                  {demos[selectedDemo].fileName}
                </span>
              </div>
              <pre className="p-6 overflow-x-auto text-sm leading-relaxed">
                <code className="language-python">{demos[selectedDemo].code}</code>
              </pre>
            </div>
          </div>

          {/* Video section */}
          <div className="lg:sticky lg:top-8">
            <div className="rounded-xl overflow-hidden border shadow-2xl bg-fd-card">
              <video
                key={selectedDemo} // Force re-render when demo changes
                src={demos[selectedDemo].videoSrc}
                controls
                className="w-full h-auto"
                poster={demos[selectedDemo].videoSrc}
              >
                Your browser does not support the video tag.
              </video>
            </div>
            {demos[selectedDemo].videoCaption && (
              <p className="text-sm text-fd-muted-foreground mt-4 text-center">
                {demos[selectedDemo].videoCaption}
              </p>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
