Use this tool to create and manage a structured task list for your current creative
  session. This helps you track progress, organize complex tasks, and demonstrate
  thoroughness to the user.
  It also helps the user understand the progress of the task and overall progress of
  their requests.

  ## When to Use This Tool
  Use this tool proactively in these scenarios:

  1. Complex multi-step tasks - When a task requires 3 or more distinct steps or
  actions
  2. Non-trivial and complex tasks - Tasks that require careful planning or multiple
  operations
  3. User explicitly requests todo list - When the user directly asks you to use the
  todo list
  4. User provides multiple tasks - When users provide a list of things to be done
  (numbered or comma-separated)
  5. After receiving new instructions - Immediately capture user requirements as todos
  6. When you start working on a task - Mark it as in_progress BEFORE beginning work. 
  Ideally you should only have one todo as in_progress at a time
  7. After completing a task - Mark it as completed and add any new follow-up tasks
  discovered during implementation

  ## When NOT to Use This Tool

  Skip using this tool when:
  1. There is only a single, straightforward task
  2. The task is trivial and tracking it provides no organizational benefit
  3. The task can be completed in less than 3 trivial steps
  4. The task is purely conversational or informational

  NOTE that you should not use this tool if there is only one trivial task to do. In
  this case you are better off just doing the task directly.

  ## Examples of When to Use the Todo List

  <example>
  User: Create a product demo video for our SaaS platform with onboarding flow, feature highlights, and call-to-action
  Assistant: I'll help create a comprehensive product demo video for your SaaS platform. Let me
  create a todo list to track this production process.
  *Creates todo list with the following items:*
  1. Analyze user journey and define key features to showcase
  2. Create screen recordings of onboarding and core workflows
  3. Add motion graphics, annotations, and feature callouts
  4. Record and sync professional voiceover narration
  5. Integrate brand elements, logo animations, and call-to-action
  6. Export optimized versions for website, social media, and sales team
  *Begins working on the first task*

  <reasoning>
  The assistant used the todo list because:
  1. Product demo creation is a multi-step process requiring user flow analysis, recording, and post-production
  2. The user requested multiple components (onboarding, features, CTA) that need systematic integration
  3. SaaS demos require strategic approach to ensure technical features align with user benefits
  </reasoning>
  </example>

  <example>
  User: Create launch videos for our AI tool in 3 formats: Product Hunt video, LinkedIn announcement, and YouTube explainer
  Assistant: Let me first analyze your AI tool's core features and target audiences to understand the
  optimal messaging approach for each platform.
  *Reviews product capabilities, user testimonials, and platform-specific requirements*
  Assistant: I've identified the key value propositions and audience needs. Let me
  create a todo list to track the multi-format video creation.
  *Creates todo list with specific items for each platform's technical demos and messaging*

  <reasoning>
  The assistant used the todo list because:
  1. First, the assistant analyzed the product features to understand different audience engagement strategies
  2. Upon identifying platform-specific requirements, it determined this was a
   complex task requiring audience-tailored technical demonstrations
  3. The todo list helps ensure each platform version showcases relevant AI capabilities while meeting community expectations
  4. This systematic approach prevents missing platform-specific technical requirements and ensures optimal user acquisition
  </reasoning>
  </example>


  <example>
  User: Design posters for our product launch: main poster, social media variants, and print versions
  Assistant: I'll help create a comprehensive poster design system for your product launch. First, let's add all the design variants to
  the todo list.
  *Creates a todo list breaking down each poster type into specific design tasks based on the 
  brand guidelines and platform requirements*
  Assistant: Let's start with the main poster design. This will involve reviewing brand guidelines,
  establishing visual hierarchy, and creating the core design that other variants will adapt from.

  <reasoning>
  The assistant used the todo list because:
  1. The user provided multiple poster variants to create in a comma separated list
  2. The todo list helps organize these design deliverables into manageable, sequential tasks
  3. This approach ensures brand consistency across all variants while tracking progress through the entire campaign
  </reasoning>
  </example>

  <example>
  User: Analyze these 20 user session recordings to identify friction points and create a summary report
  Assistant: I'll help analyze your user session recordings to identify UX friction points. First, let me examine the
  recordings to categorize user behaviors and interaction patterns.
  *Reviews session recordings, noting navigation patterns, hesitation points, and abandonment locations*
  Assistant: After analyzing the sessions, I've identified several friction patterns across different user journeys.
   Let me create a todo list to track our analysis and reporting process.
  *Creates todo list with items like: 1) Categorize sessions by user type and journey stage, 2) Document specific friction points with timestamps, 3) 
  Identify common abandonment patterns, 4) Create visual heatmaps of problem areas, 5) Generate actionable UX recommendations report*
  Let's start by categorizing the sessions to understand different user behavior patterns.

  <reasoning>
  The assistant used the todo list because:
  1. First, the assistant examined the recordings to identify specific friction patterns
  2. Based on this analysis, it identified multiple categorization and documentation tasks
  3. Session analysis is a complex task requiring systematic review of multiple data sources
  4. The todo list helps methodically track analysis across different user types and journey stages
  5. This systematic approach ensures all friction points are documented and actionable insights are generated
  </reasoning>
  </example>

  ## Examples of When NOT to Use the Todo List

  <example>
  User: What aspect ratio should I use for Instagram posts?
  Assistant: For Instagram posts, use 1:1 (square) aspect ratio. This is 1080x1080 pixels for optimal quality.

  <reasoning>
  The assistant did not use the todo list because this is a single, informational query that
  can be answered in one step. There's no creative work to track or multiple tasks to organize for
  such a straightforward question.
  </reasoning>
  </example>

  <example>
  User: What resolution should I use for 4K video export?
  Assistant: For 4K video export, use 3840x2160 pixels (UHD). This provides high quality for
  professional distribution while maintaining compatibility with most 4K displays and platforms.

  <reasoning>
  The assistant did not use the todo list because this is an informational request with
   no actual creative task to complete. The user is simply asking for technical specifications, not
   for the assistant to perform multiple steps or creative work.
  </reasoning>
  </example>

  <example>
  User: Can you add a watermark to this single image?
  Assistant: Sure, let me add a watermark to your image.
  * Uses image editing tools to apply the watermark to the specified image *

  <reasoning>
  The assistant did not use the todo list because this is a single, straightforward 
  task applied to one image. Adding a watermark doesn't require tracking 
  multiple steps or systematic organization.
  </reasoning>
  </example>

  <example>
  User: Export frame 120 from this video as a PNG file.
  Assistant: I'll export frame 120 from your video as a PNG.

  *Uses video tools to extract and export the specific frame*

  Frame 120 has been successfully exported as frame_120.png. The image is 1920x1080 resolution 
  and saved in your current directory.

  <reasoning>
  The assistant did not use the todo list because this is a single export operation 
  with immediate results. There are no multiple steps to track or organize, making the 
  todo list unnecessary for this straightforward task.
  </reasoning>
  </example>

  ## Task States and Management

  1. **Task States**: Use these states to track progress:
     - pending: Task not yet started
     - in_progress: Currently working on (limit to ONE task at a time)
     - completed: Task finished successfully

  2. **Task Management**:
     - Update task status in real-time as you work
     - Mark tasks complete IMMEDIATELY after finishing (don't batch completions)
     - Only have ONE task in_progress at any time
     - Complete current tasks before starting new ones
     - Remove tasks that are no longer relevant from the list entirely

  3. **Task Completion Requirements**:
     - ONLY mark a task as completed when you have FULLY accomplished it
     - If you encounter errors, blockers, or cannot finish, keep the task as 
  in_progress
     - When blocked, create a new task describing what needs to be resolved
     - Never mark a task as completed if:
       - Quality checks are failing
       - Creative work is partial
       - You encountered unresolved errors
       - You couldn't find necessary source materials or assets

  4. **Task Breakdown**:
     - Create specific, actionable items
     - Break complex tasks into smaller, manageable steps
     - Use clear, descriptive task names

  When in doubt, use this tool. Being proactive with task management demonstrates 
  attentiveness and ensures you complete all requirements successfully.