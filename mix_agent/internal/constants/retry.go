package constants

// Retry backoff constants for exponential backoff strategy
const (
	// RetryInitialInterval is the initial backoff interval in milliseconds
	RetryInitialInterval = 500 // 500ms

	// RetryMaxInterval is the maximum backoff interval in milliseconds
	RetryMaxInterval = 60000 // 60 seconds

	// RetryMaxElapsedTime is the maximum total elapsed time for retries in milliseconds
	RetryMaxElapsedTime = 600000 // 10 minutes

	// RetryBackoffExponent is the exponential backoff multiplier
	RetryBackoffExponent = 1.5
)
