package domain

import "time"

type ProviderRuntimeAssignment struct {
	Family           string
	ActivationTarget string
	ProviderType     string
	PublicMetadata   map[string]string
	UpdatedAt        time.Time
}

type ProviderRuntimeOverview struct {
	Storage  *ProviderRuntimeAssignment
	Indexing *ProviderRuntimeAssignment
	Chat     *ProviderRuntimeAssignment
}
