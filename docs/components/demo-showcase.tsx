import { ReactNode } from 'react';

interface DemoShowcaseProps {
  icon: string;
  title: string;
  description: string;
  fileName: string;
  code: string;
  videoSrc: string;
  videoCaption?: string;
}

export function DemoShowcase({
  icon,
  title,
  description,
  fileName,
  code,
  videoSrc,
  videoCaption
}: DemoShowcaseProps) {
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
          <div className="bg-fd-muted/50 px-4 py-2 border-b flex items-center gap-2">
            <div className="flex gap-1.5">
              <div className="h-3 w-3 rounded-full bg-red-500/20"></div>
              <div className="h-3 w-3 rounded-full bg-yellow-500/20"></div>
              <div className="h-3 w-3 rounded-full bg-green-500/20"></div>
            </div>
            <span className="text-xs text-fd-muted-foreground ml-2">{fileName}</span>
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
    </div>
  );
}
