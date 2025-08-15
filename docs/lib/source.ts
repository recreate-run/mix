import { docs } from '@/.source';
import { loader } from 'fumadocs-core/source';
import { createElement } from 'react';
import { BookOpen, Rocket, Settings, FileText } from 'lucide-react';

const icons = {
  BookOpen,
  Rocket,
  Settings,
  FileText,
};

// See https://fumadocs.vercel.app/docs/headless/source-api for more info
export const source = loader({
  // it assigns a URL to your pages
  baseUrl: '/docs',
  source: docs.toFumadocsSource(),
  icon(icon) {
    if (icon && icon in icons)
      return createElement(icons[icon as keyof typeof icons]);
  },
});
