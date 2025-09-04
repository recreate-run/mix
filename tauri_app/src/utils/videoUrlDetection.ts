// Video URL detection utilities for attachment previews

export interface VideoUrlInfo {
  url: string;
  platform: 'youtube' | 'vimeo' | 'direct' | 'unknown';
  videoId?: string;
  thumbnailUrl?: string;
  title?: string;
}

// Video URL patterns
const VIDEO_URL_PATTERNS = {
  youtube: [
    /(?:https?:\/\/)?(?:www\.)?(?:youtube\.com\/watch\?v=|youtu\.be\/)([a-zA-Z0-9_-]+)/,
    /(?:https?:\/\/)?(?:www\.)?youtube\.com\/embed\/([a-zA-Z0-9_-]+)/
  ],
  vimeo: [
    /(?:https?:\/\/)?(?:www\.)?vimeo\.com\/(\d+)/,
    /(?:https?:\/\/)?player\.vimeo\.com\/video\/(\d+)/
  ],
  direct: [
    /https?:\/\/[^\s]+\.(?:mp4|webm|ogg|mov|avi|wmv|flv|m4v)(?:\?[^\s]*)?/i
  ]
};

/**
 * Detects video URLs in text and returns their information
 */
export function detectVideoUrls(text: string): VideoUrlInfo[] {
  const videoUrls: VideoUrlInfo[] = [];
  
  // YouTube detection
  for (const pattern of VIDEO_URL_PATTERNS.youtube) {
    const matches = text.matchAll(new RegExp(pattern, 'g'));
    for (const match of matches) {
      const videoId = match[1];
      if (videoId) {
        videoUrls.push({
          url: match[0],
          platform: 'youtube',
          videoId,
          thumbnailUrl: `https://img.youtube.com/vi/${videoId}/maxresdefault.jpg`,
          title: `YouTube Video: ${videoId}`
        });
      }
    }
  }
  
  // Vimeo detection
  for (const pattern of VIDEO_URL_PATTERNS.vimeo) {
    const matches = text.matchAll(new RegExp(pattern, 'g'));
    for (const match of matches) {
      const videoId = match[1];
      if (videoId) {
        videoUrls.push({
          url: match[0],
          platform: 'vimeo',
          videoId,
          title: `Vimeo Video: ${videoId}`
          // Note: Vimeo thumbnails require API call, we'll handle this later
        });
      }
    }
  }
  
  // Direct video file detection
  for (const pattern of VIDEO_URL_PATTERNS.direct) {
    const matches = text.matchAll(new RegExp(pattern, 'g'));
    for (const match of matches) {
      const url = match[0];
      const filename = url.split('/').pop()?.split('?')[0] || 'video';
      videoUrls.push({
        url,
        platform: 'direct',
        title: filename
      });
    }
  }
  
  // Remove duplicates
  return videoUrls.filter((video, index, self) => 
    index === self.findIndex(v => v.url === video.url)
  );
}

/**
 * Creates a video attachment object from URL info
 */
export function createVideoUrlAttachment(videoInfo: VideoUrlInfo) {
  return {
    id: `video-url:${videoInfo.url}`,
    name: videoInfo.title || 'Video',
    type: 'video' as const,
    url: videoInfo.url,
    thumbnailUrl: videoInfo.thumbnailUrl,
    platform: videoInfo.platform
  };
}

/**
 * Checks if a string contains any video URLs
 */
export function hasVideoUrls(text: string): boolean {
  const allPatterns = [
    ...VIDEO_URL_PATTERNS.youtube,
    ...VIDEO_URL_PATTERNS.vimeo,
    ...VIDEO_URL_PATTERNS.direct
  ];
  
  return allPatterns.some(pattern => pattern.test(text));
}

/**
 * Converts a YouTube URL to its embed format
 */
export function getYouTubeEmbedUrl(url: string): string | null {
  for (const pattern of VIDEO_URL_PATTERNS.youtube) {
    const match = url.match(pattern);
    if (match && match[1]) {
      const videoId = match[1];
      return `https://www.youtube.com/embed/${videoId}`;
    }
  }
  return null;
}

/**
 * Checks if a URL is a YouTube URL
 */
export function isYouTubeUrl(url: string): boolean {
  return VIDEO_URL_PATTERNS.youtube.some(pattern => pattern.test(url));
}