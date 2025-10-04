import '@/app/global.css';
import { RootProvider } from 'fumadocs-ui/provider';
import { Inter, Space_Mono } from 'next/font/google';
import type { Metadata } from 'next';
import type { ReactNode } from 'react';

export const metadata: Metadata = {
  icons: {
    icon: '/icon.png',
    shortcut: '/favicon.ico',
    apple: '/apple-icon.png',
  },
};

const inter = Inter({
  subsets: ['latin'],
});

const spaceMono = Space_Mono({
  weight: ['400', '700'],
  subsets: ['latin'],
  variable: '--font-space-mono',
});

export default function Layout({ children }: { children: ReactNode }) {
  return (
    <html lang="en" className={`${inter.className} ${spaceMono.variable}`} suppressHydrationWarning>
      <body className="flex flex-col min-h-[100vh]" suppressHydrationWarning>
        <RootProvider
          search={{
            enabled: true,
          }}
          theme={{
            enabled: true,
            defaultTheme: 'system',
          }}
        >
          {children}
        </RootProvider>
      </body>
    </html>
  );
}
