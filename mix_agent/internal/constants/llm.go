package constants

import "mix/internal/llm/models"

// LLM provider and model defaults
const (
	DefaultProvider             models.ModelProvider   = models.ProviderAnthropic
	DefaultMainModel            models.ModelID         = "claude-sonnet-4-5"
	DefaultMainMaxTokens        int64                  = 4096
	DefaultMainReasoningEffort  models.ReasoningEffort = "medium"
	DefaultSubAgentModel        models.ModelID         = "claude-sonnet-4-5"
	DefaultSubAgentMaxTokens    int64                  = 2048
	DefaultSubAgentReasoningEffort models.ReasoningEffort = "medium"
)
