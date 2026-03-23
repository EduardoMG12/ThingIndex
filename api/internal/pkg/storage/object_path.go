package storage

import (
	"errors"
	"strings"
)

func objectPathForStorage(bucketName, objectName string) string {
	return strings.TrimSpace(bucketName) + "/" + strings.TrimPrefix(strings.TrimSpace(objectName), "/")
}

func objectKeyFromPath(bucketName, objectPath string) (string, error) {
	trimmedPath := strings.TrimSpace(objectPath)
	if trimmedPath == "" {
		return "", errors.New("object path is required")
	}

	bucketPrefix := strings.TrimSpace(bucketName)
	if bucketPrefix != "" {
		trimmedPath = strings.TrimPrefix(trimmedPath, bucketPrefix+"/")
	}
	trimmedPath = strings.TrimPrefix(trimmedPath, "/")
	if trimmedPath == "" {
		return "", errors.New("object path is required")
	}

	return trimmedPath, nil
}
