package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"thing-index/internal/domain"
)

func (c *Config) DefaultStorageProviderType() string {
	if c == nil {
		return domain.ProviderMinio
	}

	providerType := strings.ToUpper(strings.TrimSpace(c.StorageProvider))
	if providerType == "" {
		return domain.ProviderMinio
	}

	return providerType
}

func (c *Config) DefaultStorageCredentials() (string, map[string]string, error) {
	if c == nil {
		return "", nil, errors.New("config is not loaded")
	}

	providerType := c.DefaultStorageProviderType()
	switch providerType {
	case domain.ProviderMinio:
		return providerType, map[string]string{
			"endpoint":    strings.TrimSpace(c.MinioEndpoint),
			"access_key":  strings.TrimSpace(c.MinioAccessKey),
			"secret_key":  strings.TrimSpace(c.MinioSecretKey),
			"bucket_name": strings.TrimSpace(c.MinioBucketName),
			"use_ssl":     strings.TrimSpace(c.MinioUseSSL),
		}, nil
	case domain.ProviderGCPStorage:
		credentials := map[string]string{
			"project_id":  strings.TrimSpace(c.GCPStorageProjectID),
			"bucket_name": strings.TrimSpace(c.GCPStorageBucketName),
		}
		if location := strings.TrimSpace(c.GCPStorageLocation); location != "" {
			credentials["location"] = location
		}
		if credentialsJSON := strings.TrimSpace(c.GCPStorageCredentialsJSON); credentialsJSON != "" {
			credentials["credentials_json"] = credentialsJSON
		}
		if emulatorHost := strings.TrimSpace(c.GCPStorageEmulatorHost); emulatorHost != "" {
			credentials["emulator_host"] = emulatorHost
		}
		return providerType, credentials, nil
	default:
		return "", nil, fmt.Errorf("STORAGE_PROVIDER %s is not supported for runtime bootstrap", providerType)
	}
}

func (c *Config) validateDefaultStorageConfiguration() error {
	if c == nil {
		return errors.New("config is not loaded")
	}

	providerType, credentials, err := c.DefaultStorageCredentials()
	if err != nil {
		return err
	}

	switch providerType {
	case domain.ProviderMinio:
		for _, key := range []string{"endpoint", "access_key", "secret_key", "bucket_name"} {
			if strings.TrimSpace(credentials[key]) == "" {
				return fmt.Errorf("missing required configuration for %s: %s", providerType, storageConfigEnvName(providerType, key))
			}
		}
	case domain.ProviderGCPStorage:
		for _, key := range []string{"project_id", "bucket_name"} {
			if strings.TrimSpace(credentials[key]) == "" {
				return fmt.Errorf("missing required configuration for %s: %s", providerType, storageConfigEnvName(providerType, key))
			}
		}

		hasCredentialsJSON := strings.TrimSpace(credentials["credentials_json"]) != ""
		hasEmulatorHost := strings.TrimSpace(credentials["emulator_host"]) != ""
		if !hasCredentialsJSON && !hasEmulatorHost {
			return fmt.Errorf("missing required configuration for %s: GCP_STORAGE_CREDENTIALS_JSON or GCP_STORAGE_EMULATOR_HOST", providerType)
		}
		if hasCredentialsJSON && !json.Valid([]byte(credentials["credentials_json"])) {
			return errors.New("GCP_STORAGE_CREDENTIALS_JSON must contain valid JSON")
		}
	default:
		return fmt.Errorf("STORAGE_PROVIDER %s is not supported for runtime bootstrap", providerType)
	}

	return nil
}

func storageConfigEnvName(providerType, key string) string {
	switch providerType {
	case domain.ProviderMinio:
		switch key {
		case "endpoint":
			return "MINIO_ENDPOINT"
		case "access_key":
			return "MINIO_ACCESS_KEY"
		case "secret_key":
			return "MINIO_SECRET_KEY"
		case "bucket_name":
			return "MINIO_BUCKET_NAME"
		}
	case domain.ProviderGCPStorage:
		switch key {
		case "project_id":
			return "GCP_STORAGE_PROJECT_ID"
		case "bucket_name":
			return "GCP_STORAGE_BUCKET_NAME"
		case "credentials_json":
			return "GCP_STORAGE_CREDENTIALS_JSON"
		case "emulator_host":
			return "GCP_STORAGE_EMULATOR_HOST"
		}
	}

	return key
}
