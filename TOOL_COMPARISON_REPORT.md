# Tool Comparison: Composer vs Mix Agent

**Date:** 2026-02-06
**Purpose:** Identify feature gaps between Composer's browser agent tools and Mix's agent toolset

---

## Executive Summary

Mix agent provides **12 active tools** vs Composer's **17 tools**. While there's significant overlap in core file operations and basic browser automation, Mix lacks critical browser capabilities including **file uploads**, **DOM search**, **text extraction**, **multi-tab support**, and **user notifications**. Mix compensates with specialized code search tools (Glob/Grep) and web content analysis (WebFetch).

**Architecture Difference:** Composer uses accessibility trees and text extraction (content-focused), Mix uses visual screenshots with numbered overlays (UI-focused).

---

## Core Tool Comparison

| Composer Tool | Mix Equivalent | Match Quality | Key Gaps |
|---------------|----------------|---------------|----------|
| **navigate** | Browser (`open`) | ⚠️ Partial | No back/forward navigation, no `file://` URLs |
| **read_page** | Browser (`screenshot`) | ⚠️ Different | Returns images vs accessibility tree; no text extraction |
| **file_upload** | ❌ None | ❌ Missing | Cannot upload files to forms |
| **action** | Browser (individual) | ⚠️ Limited | No right-click, double-click, drag, form_input, or action sequences |
| **find** | ❌ None | ❌ Missing | No DOM search beyond viewport |
| **get_page_text** | ❌ None | ❌ Missing | No text extraction capability |
| **create_tab** | ❌ None | ❌ Missing | Single context only |
| **list_tabs** | ❌ None | ❌ Missing | No multi-tab support |
| **read** | ReadText | ✅ Similar | Mix requires absolute paths, rejects binaries |
| **write** | Write | ✅ Similar | Mix requires reading files before overwrite |
| **edit** | Edit | ✅ Nearly Identical | Mix enforces read-before-edit |
| **bash** | Bash | ✅ Very Similar | Mix has safe/banned command lists |
| **task** | Task | ⚠️ Different | Composer: parallel batch processing; Mix: single synchronous subagent |
| **execution** | Task | ⚠️ Different | Composer shares browser context; Mix uses isolated sessions |
| **todo_write** | TodoWrite | ✅ Similar | Mix adds priority levels |
| **web_search** | WebSearch | ✅ Similar | Mix uses Brave (max 3 results) vs Exa (max 10) |
| **notify_user** | ❌ None | ❌ Missing | No desktop notifications |

---

## Critical Missing Features in Mix

### 1. **File Upload** (HIGH IMPACT)
- **Composer:** Uploads workspace files to browser file inputs
- **Mix:** No capability
- **Impact:** Cannot automate file upload workflows (forms, attachments, imports)

### 2. **DOM Search** (HIGH IMPACT)
- **Composer:** Natural language search across entire DOM (beyond viewport), returns up to 100 elements
- **Mix:** Only detects visible interactive elements in screenshots
- **Impact:** Cannot find off-screen elements, limited search precision

### 3. **Text Extraction** (HIGH IMPACT)
- **Composer:** `get_page_text` extracts all text content, prioritizes main content
- **Mix:** No text extraction, only visual screenshots
- **Impact:** Cannot extract structured data, scrape content, or analyze text-heavy pages efficiently

### 4. **Multi-Tab Management** (MEDIUM IMPACT)
- **Composer:** `create_tab`, `list_tabs` for managing multiple browser tabs
- **Mix:** Single browser context per session
- **Impact:** Cannot handle multi-tab workflows (compare pages, manage multiple logins)

### 5. **User Notifications** (MEDIUM IMPACT)
- **Composer:** `notify_user` sends desktop notifications for interrupting user
- **Mix:** No notification capability
- **Impact:** Cannot proactively request CAPTCHA help, credentials, or critical decisions

### 6. **Advanced Browser Actions** (MEDIUM IMPACT)
- **Missing:** Right-click, double-click, drag-and-drop, form_input (direct value setting), wait action
- **Impact:** Limited interaction patterns, no context menus, no drag operations

### 7. **Browser History Navigation** (LOW IMPACT)
- **Composer:** Forward/back navigation
- **Mix:** Only direct URL navigation
- **Impact:** Cannot use browser history

### 8. **Action Sequences** (LOW IMPACT)
- **Composer:** Execute multiple actions in single API call
- **Mix:** One action per call
- **Impact:** More API roundtrips required

---

## Tools Unique to Mix

| Tool | Purpose | Value |
|------|---------|-------|
| **Glob** | Fast file pattern matching with `**/*.js` syntax | Superior file discovery vs bash `find` |
| **Grep** | Regex content search with ripgrep | Powerful code search with context |
| **WebFetch** | Fetch URLs, convert HTML→markdown, analyze with prompt | Content analysis beyond search |
| **ExitPlanMode** | Plan approval workflow | Structured planning process |

---

## Browser Tool Action Coverage

| Action Type | Composer | Mix | Status |
|-------------|----------|-----|--------|
| Navigate to URL | ✅ | ✅ | Complete |
| Back/forward | ✅ | ❌ | Missing |
| Screenshot | ✅ | ✅ | Complete |
| Left click | ✅ | ✅ | Complete |
| Right click | ✅ | ❌ | Missing |
| Double/triple click | ✅ | ❌ | Missing |
| Type text | ✅ | ✅ | Complete |
| Form input | ✅ | ❌ | Missing |
| Scroll | ✅ | ✅ | Complete |
| Drag & drop | ✅ | ❌ | Missing |
| Wait/pause | ✅ | ❌ | Missing |
| File upload | ✅ | ❌ | Missing |

---

## Recommendations

### Priority 1 (High Value, High Impact)
1. **Add DOM search capability** - Critical for finding off-screen elements
2. **Add text extraction** - Essential for content scraping and data extraction
3. **Add file upload** - Required for form automation and file-based workflows

### Priority 2 (Medium Value, Medium Impact)
4. **Add multi-tab support** - Useful for comparison and multi-context workflows
5. **Add right-click and context menu** - Needed for advanced interactions
6. **Add form_input action** - Direct value setting for React/modern apps
7. **Add user notifications** - Critical for handling CAPTCHA, 2FA, credentials

### Priority 3 (Nice to Have)
8. **Add action sequencing** - Reduce API roundtrips
9. **Add drag-and-drop** - UI testing and reordering workflows
10. **Add browser history navigation** - Convenience feature

### Architectural Considerations
- **Consider hybrid approach:** Combine Mix's visual overlay system with accessibility tree extraction
- **Evaluate trade-offs:** Composer's content-focused approach vs Mix's UI-focused approach
- **Session management:** Composer's shared browser context for subagents vs Mix's isolation

---

## Conclusion

Mix has a solid foundation with 12 active tools and strong code search capabilities. However, **6 critical gaps** exist in browser automation compared to Composer. The most impactful additions would be: **DOM search**, **text extraction**, and **file upload**. These three features would significantly enhance Mix's browser automation capabilities for real-world workflows.

**Current Browser Automation Score:** Mix covers ~60% of Composer's browser capabilities.

**With Priority 1 additions:** Would reach ~85% coverage of critical use cases.
