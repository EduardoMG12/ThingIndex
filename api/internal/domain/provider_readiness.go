package domain

import "time"

const (
	ProviderReadinessStatusReady     = "READY"
	ProviderReadinessStatusAttention = "ATTENTION"
	ProviderReadinessStatusFailed    = "FAILED"

	ProviderReadinessCheckPassed      = "PASSED"
	ProviderReadinessCheckWarning     = "WARNING"
	ProviderReadinessCheckFailed      = "FAILED"
	ProviderReadinessCheckUnsupported = "UNSUPPORTED"

	ProviderReadinessCheckCredentials    = "CREDENTIALS"
	ProviderReadinessCheckConnection     = "CONNECTION"
	ProviderReadinessCheckModelDiscovery = "MODEL_DISCOVERY"
)

type ProviderReadinessCheck struct {
	Code     string            `json:"code"`
	Status   string            `json:"status"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type ProviderReadinessReport struct {
	Family           string                   `json:"family"`
	ProviderType     string                   `json:"provider_type"`
	Status           string                   `json:"status"`
	CheckedAt        time.Time                `json:"checked_at"`
	Checks           []ProviderReadinessCheck `json:"checks"`
	DiscoveredModels []string                 `json:"discovered_models,omitempty"`
}
