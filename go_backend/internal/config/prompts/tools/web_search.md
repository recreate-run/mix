- Allows Claude to search the web, images, and videos using the results to inform responses
- Provides up-to-date information for current events, recent data, visual content, and video content
- Returns search result information formatted as search result blocks
- Use this tool for accessing information beyond Claude's knowledge cutoff
- Searches are performed automatically within a single API call
- Supports web pages, image search, and video search with unified interface

Usage notes:

- Domain filtering is supported to include or block specific websites
- Account for "Today's date" in <env>. For example, if <env> says "Today's date:
2025-07-01", and the user wants the latest docs, do not use 2024 in the search
query. Use 2025.
- Image search includes safety filtering and can find visual content related to queries
- Video search provides access to video content with metadata like duration, views, and upload dates
- Web search is the default behavior when search_type is not specified

Parameters:

- query (required): The search query to use (minimum 2 characters)
- search_type (optional): "web" (default) for web pages, "images" for image search, or "videos" for video search
- allowed_domains (optional): Only include search results from these domains
- blocked_domains (optional): Never include search results from these domains  
- safesearch (optional): "strict" (default), "moderate", or "off" - applies to image and video search
- spellcheck (optional): boolean, enables spell correction (default: true for image and video search)
