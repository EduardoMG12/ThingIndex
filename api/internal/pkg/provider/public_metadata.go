package provider

import (
	"strings"

	"thing-index/internal/domain"
)

func PublicMetadata(family string, credentials map[string]string) map[string]string {
	publicMetadata := make(map[string]string)

	switch strings.ToUpper(strings.TrimSpace(family)) {
	case domain.FamilyLLM:
		if model := strings.TrimSpace(credentials["model"]); model != "" {
			publicMetadata["model"] = model
		}
	case domain.FamilyStorage:
		if bucket := strings.TrimSpace(credentials["bucket_name"]); bucket != "" {
			publicMetadata["bucket_name"] = bucket
		}
		if region := strings.TrimSpace(credentials["region"]); region != "" {
			publicMetadata["region"] = region
		}
		if projectID := strings.TrimSpace(credentials["project_id"]); projectID != "" {
			publicMetadata["project_id"] = projectID
		}
		if emulatorHost := strings.TrimSpace(credentials["emulator_host"]); emulatorHost != "" {
			publicMetadata["emulator_host"] = emulatorHost
		}
	}

	if len(publicMetadata) == 0 {
		return nil
	}

	return publicMetadata
}
