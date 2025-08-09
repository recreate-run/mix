# Video editing heuristics

## FFmpeg: Pure Automation
- Format conversion, trimming, basic editing
- Batch processing identical files
- Simple text overlays (centered, bottom-aligned)
- Standard transitions (crossfades/dissolves)
- Exact specifications provided

## Remotion: Animated Text
- Kinetic typography, text animations
- Data-driven title generation
- Batch personalized text variations
- Code-driven animation control

## Hybrid Workflow: Creative Work
- Custom positioning/styling decisions
- Multi-element compositions
- Brand-specific requirements
- Color grading, visual hierarchy

**Workflow:** FFmpeg prep → GUI creative work → FFmpeg finishing

---

## Examples

### Format Conversion & Basic Editing

<example>
User: Trim the first 30 seconds from this video and extract the audio
Assistant: I'll trim the video and extract audio using FFmpeg for automated processing.

*Agent executes FFmpeg commands to trim video and extract audio in parallel*

<reasoning>
FFmpeg chosen: Simple operations with exact specifications (30 seconds), no creative decisions needed.
</reasoning>
</example>

<example>
User: Make a 30-second promotional video combining these 5 clips with smooth transitions
Assistant: I'll combine your clips using FFmpeg with automated crossfade transitions and timing.

*Agent executes FFmpeg with concat filter and crossfade transitions to create 30-second output*

<reasoning>
FFmpeg chosen: "Smooth transitions" = standard crossfades, specified duration with predictable timing.
</reasoning>
</example>

### Batch Operations

<example>
User: Convert all these 20 MOV files to MP4 and resize them to 720p
Assistant: I'll use FFmpeg to batch process all your MOV files with full automation.

*Agent creates FFmpeg batch script to process all files in parallel*

<reasoning>
FFmpeg chosen: Pure automation across identical operations, no creative decisions per file.
</reasoning>
</example>

### Animated Titles & Typography

<example>
User: Create animated title sequence with kinetic typography for our brand video, and generate 50 personalized video intros with animated names
Assistant: I'll use Remotion to create animated titles with code-driven kinetic typography and batch generate personalized variations.

*Agent creates Remotion composition to:*
- Design text animations with precise timing control
- Implement brand-consistent typography animations
- Program kinetic effects (text reveals, morphing, scaling)
- Define animated text template with name variables
- Batch generate 50 variations with different names
- Export all variations automatically

<reasoning>
Remotion chosen: Kinetic typography requires code-driven animation control, data-driven generation scales efficiently.
</reasoning>
</example>

### Static Text & Subtitles

<example>
User: Add simple subtitle text to this video
Assistant: I'll use FFmpeg to add subtitle text overlay for automated processing.

*Agent executes FFmpeg command to overlay subtitle text with standard positioning*

<reasoning>
FFmpeg chosen: Simple text overlay with standard positioning, no creative decisions needed.
Note: Use Hybrid only if custom positioning/styling requires human judgment.
</reasoning>
</example>

### Hybrid Workflows

<example>
User: Create a product demo video with our logo overlay, background music, and text captions
Assistant: I'll create your product demo using a hybrid workflow for maximum efficiency and quality.

**Step 1 - Preprocessing (FFmpeg):**
- Normalize product demo footage audio levels
- Convert to consistent format and resolution
- Extract audio properties for music sync analysis

**Step 2 - Creative Work (Blender):**
- Import preprocessed footage to timeline
- Position logo overlay (requires human judgment)
- Sync background music to video timeline
- Create and time text captions
- **Pause for review**: "Logo positioning and caption timing ready for approval"

**Step 3 - Finishing (FFmpeg):**
- Export final composition
- Generate multiple formats (1080p, 720p, social media clips)
- Apply compression optimizations

<reasoning>
Hybrid chosen: Multiple elements (logo + music + captions) with positioning decisions requiring human judgment.
</reasoning>
</example>

---

## Warning Signs

### ❌ Don't use FFmpeg when:
- Multiple elements need coordination  
- User might want to "adjust that" after seeing result

### ❌ Don't use Remotion when:
- Simple text overlays (use FFmpeg)
- Static titles with no animation
- One-off creative work

### ❌ Don't use Hybrid Workflows when:
- Simple, predictable transformations
- Batch processing identical files
- No creative decisions required

---

## Implementation Details

### Agent Pause Points
- Creative positioning ("Is logo placement appealing?")
- Visual style approval ("Match brand standards?")
- Pacing/timing ("Edit flow naturally?")
- Multi-element composition ("Elements balanced?")

### Handoff Signals  
- **"Pause for review"** - Setup complete, needs approval
- **"Creative input needed"** - Decision requires human judgment
- **"Ready for approval"** - Draft complete, review before export
- **"Timeline ready"** - Project structured, human can take control

**When in doubt:** Always use FFmpeg for pure automation.