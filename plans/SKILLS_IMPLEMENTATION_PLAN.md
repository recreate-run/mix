# Agent Skills Implementation Plan for Mix

## Overview

This plan implements the Agent Skills feature from Anthropic's blog post using Mix's existing infrastructure. Skills will be filesystem-based directories containing a `SKILL.md` file with YAML frontmatter (name, description) and optional bundled resources. The implementation leverages progressive disclosure: skill metadata loads at startup into the system prompt, while full content loads on-demand via existing tools.

## Architecture Decision

**Skills as Filesystem Directories** at `~/.mix/skills/{skill_name}/SKILL.md`

**Why This Approach:**
- Leverages existing Read tool (no new tool interface needed)
- Skills execute via existing Bash tool for bundled scripts
- Variable substitution system already supports dynamic injection
- Zero breaking changes to current architecture
- Portable and shareable (can be git repos)

## Implementation Steps

### Step 1: Create Skills Discovery Service

**File:** `mix_agent/internal/skills/discovery.go` (new file)

**Purpose:** Scan `~/.mix/skills/` at startup, parse YAML frontmatter, build in-memory registry.

**Complete Implementation:**
```go
package skills

import (
    "fmt"
    "os"
    "path/filepath"
    "strings"
    "gopkg.in/yaml.v3"
)

type SkillMetadata struct {
    Name        string `yaml:"name"`
    Description string `yaml:"description"`
    Path        string `yaml:"-"` // Full path to SKILL.md
}

type Service interface {
    DiscoverSkills() ([]SkillMetadata, error)
    GetSkillsList() string
}

type service struct {
    skills []SkillMetadata
}

func NewService() Service {
    return &service{}
}

func (s *service) DiscoverSkills() ([]SkillMetadata, error) {
    homeDir, err := os.UserHomeDir()
    if err != nil {
        return nil, err
    }

    skillsDir := filepath.Join(homeDir, ".mix", "skills")
    entries, err := os.ReadDir(skillsDir)
    if err != nil {
        if os.IsNotExist(err) {
            return []SkillMetadata{}, nil
        }
        return nil, err
    }

    var skills []SkillMetadata
    for _, entry := range entries {
        if !entry.IsDir() {
            continue
        }

        skillPath := filepath.Join(skillsDir, entry.Name(), "SKILL.md")
        content, err := os.ReadFile(skillPath)
        if err != nil {
            continue
        }

        var metadata SkillMetadata
        if err := parseFrontmatter(content, &metadata); err != nil {
            continue
        }

        metadata.Path = skillPath
        skills = append(skills, metadata)
    }

    s.skills = skills
    return skills, nil
}

func (s *service) GetSkillsList() string {
    if len(s.skills) == 0 {
        return "No skills available."
    }

    var builder strings.Builder
    for _, skill := range s.skills {
        builder.WriteString(fmt.Sprintf("- %s: %s (path: %s)\n",
            skill.Name, skill.Description, skill.Path))
    }
    return builder.String()
}

func parseFrontmatter(content []byte, metadata *SkillMetadata) error {
    parts := strings.SplitN(string(content), "---", 3)
    if len(parts) < 3 {
        return fmt.Errorf("no frontmatter found")
    }
    return yaml.Unmarshal([]byte(parts[1]), metadata)
}
```

**Integration Point:** Called during app initialization in `internal/app/app.go:34-127`

### Step 2: Update App Initialization

**File:** `mix_agent/internal/app/app.go`

**Line:** After line 73 (after `InitAPICredentials`)

**Changes:**
```go
// Initialize skills service (after InitAPICredentials)
if err := config.InitSkills(); err != nil {
    logging.Warn("Failed to initialize skills", "error", err)
}
```

**Note:** Skills service is now globally accessible via `config.GetSkills()`, similar to user preferences pattern. No changes needed to App struct.

### Step 3: Add Global Skills Service Accessor

**File:** `mix_agent/internal/config/config.go`

**Line:** After line 110 (after `apiCredentialsService` declaration)

**Add global service variable:**
```go
// Global skills service
var skillsService skills.Service
```

**Line:** After line 367 (after `InitAPICredentials` function)

**Add initialization and accessor functions:**
```go
// InitSkills initializes the skills discovery service
func InitSkills() error {
    cfgMutex.Lock()
    defer cfgMutex.Unlock()

    skillsService = skills.NewService()
    _, err := skillsService.DiscoverSkills()
    if err != nil {
        return fmt.Errorf("failed to discover skills: %w", err)
    }
    return nil
}

// GetSkills returns the skills service
func GetSkills() skills.Service {
    cfgMutex.RLock()
    defer cfgMutex.RUnlock()
    return skillsService
}
```

### Step 4: Extend Variable Substitution System

**File:** `mix_agent/internal/llm/prompt/loader.go`

**Line:** 36-44 (in LoadPrompt function after adding today_date)

**Changes:** Add `$<skills_list>` variable using global service accessor

```go
// Line 36-44 - Add after today_date:
allVars["today_date"] = time.Now().Format("2006-01-02")

// NEW: Add skills list from global service
if skillsService := config.GetSkills(); skillsService != nil {
    allVars["skills_list"] = skillsService.GetSkillsList()
}
```

### Step 5: Update System Prompt

**File:** `mix_agent/internal/config/prompts/system.md`

**Line:** After line 52 (before env section)

**Add New Section:**
```markdown
## Available Skills

Skills are specialized capabilities you can load on-demand to help with specific tasks. Each skill is a directory containing instructions and code.

$<skills_list>

To use a skill, read its SKILL.md file using the Read tool at the path shown above. Skills may reference additional files or scripts within their directory—read those as needed using standard tools.
```

### Step 6: Ensure .mix/skills Directory Exists

**File:** `mix_agent/internal/config/config.go`

**Line:** 292-309 (`ensureEmbeddedDataDirectory` function)

**Changes:** Create skills directory alongside .mix directory

At line 305 (after creating .mix directory):
```go
// Create .mix/skills directory for user skills
skillsDir := filepath.Join(targetMixDir, "skills")
if err := os.MkdirAll(skillsDir, 0o755); err != nil {
    return fmt.Errorf("failed to create skills directory: %w", err)
}
```

### Step 7: Optional REST API Endpoint

**File:** `mix_agent/internal/http/rest_skills.go` (new file)

**Purpose:** Allow frontend to list discovered skills

**Implementation:**
```go
// GET /api/skills - List all discovered skills
func (s *Server) handleListSkills(w http.ResponseWriter, r *http.Request) {
    skills, err := s.app.Skills.DiscoverSkills()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    json.NewEncoder(w).Encode(skills)
}
```

**Update:** `mix_agent/internal/http/rest_docs.go` with OpenAPI spec for new endpoint

## Implementation Order

1. **Create Skills Service** (`internal/skills/discovery.go`) - Complete service with YAML parsing
2. **Add Global Accessors** (`config.go:110,367`) - InitSkills() and GetSkills() functions
3. **Ensure Directory Exists** (`config.go:305`) - Create ~/.mix/skills/ on startup
4. **Initialize in App** (`app.go:73`) - Call config.InitSkills() after credentials init
5. **Variable Substitution** (`prompt/loader.go:36-44`) - Inject $<skills_list> via GetSkills()
6. **System Prompt** (`prompts/system.md:52`) - Add Available Skills documentation section
7. **REST API** (Optional) - Frontend skill browsing endpoint

## Example Skill Structure

```
~/.mix/skills/
└── pdf-processing/
    ├── SKILL.md          # Core skill with YAML frontmatter
    ├── forms.md          # Referenced by SKILL.md
    └── extract_fields.py # Executable script
```

**SKILL.md format:**
```markdown
---
name: PDF Processing
description: Extract and fill PDF form fields
---

Use this skill when working with PDF forms. See `forms.md` for filling instructions.

To extract fields: `python pdf-processing/extract_fields.py <file.pdf>`
```

## Dependencies and Integration

**Required Go Package:**
- `gopkg.in/yaml.v3` - YAML frontmatter parsing (add to go.mod)

**Integration Pattern:**
This implementation follows Mix's existing global service pattern used by UserPreferences and APICredentials services. The skills service is initialized once during app startup (`app.go:73`) and accessed globally via `config.GetSkills()` wherever needed. This avoids context propagation complexity while maintaining thread-safe access through config.cfgMutex.

**Progressive Disclosure Flow:**
1. App startup → config.InitSkills() → Discovers all skills in ~/.mix/skills/
2. Session creation → createSessionProvider() → Loads system prompt
3. Prompt loading → LoadPrompt() → Calls config.GetSkills().GetSkillsList()
4. System prompt → Includes skill metadata (name, description, path)
5. Agent execution → Reads SKILL.md via Read tool when skill is relevant
6. Skill execution → Runs bundled scripts via Bash tool as needed
