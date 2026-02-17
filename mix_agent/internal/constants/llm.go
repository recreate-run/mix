package constants

import "mix/internal/llm/models"

// LLM provider and model defaults
const (
	DefaultProvider             models.ModelProvider   = models.ProviderAnthropic
	DefaultMainModel            models.ModelID         = models.Claude45Sonnet
	DefaultMainMaxTokens        int64                  = 4096
	DefaultMainReasoningEffort     models.ReasoningEffort = models.ReasoningEffortMedium
	DefaultSubAgentModel           models.ModelID         = models.Claude45Sonnet
	DefaultSubAgentMaxTokens       int64                  = 2048
	DefaultSubAgentReasoningEffort models.ReasoningEffort = models.ReasoningEffortMedium
)
