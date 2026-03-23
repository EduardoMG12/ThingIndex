package handlers

import (
	"mime"
	"path"
	"strings"
)

const fallbackDownloadFilename = "documento"

func buildSafeContentDisposition(disposition, filename string) string {
	dispositionType := strings.ToLower(strings.TrimSpace(disposition))
	if dispositionType != "inline" {
		dispositionType = "attachment"
	}

	mediaType := mime.FormatMediaType(dispositionType, map[string]string{
		"filename": sanitizeContentDispositionFilename(filename),
	})
	if mediaType == "" {
		return dispositionType + `; filename="` + fallbackDownloadFilename + `"`
	}

	return mediaType
}

func buildDocumentIndexExportFilename(title string) string {
	safeTitle := sanitizeContentDispositionFilename(title)
	ext := path.Ext(safeTitle)
	if ext != "" {
		safeTitle = strings.TrimSuffix(safeTitle, ext)
		safeTitle = strings.TrimSpace(safeTitle)
	}
	if safeTitle == "" {
		safeTitle = fallbackDownloadFilename
	}

	return safeTitle + ".thingindex-index.json"
}

func sanitizeContentDispositionFilename(value string) string {
	trimmed := strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	if idx := strings.LastIndex(trimmed, "/"); idx >= 0 {
		trimmed = trimmed[idx+1:]
	}

	cleaned := strings.Map(func(r rune) rune {
		switch {
		case r < 0x20 || r == 0x7f:
			return -1
		case strings.ContainsRune(`"<>:|?*;`, r):
			return '_'
		default:
			return r
		}
	}, trimmed)

	cleaned = strings.Join(strings.Fields(strings.TrimSpace(cleaned)), " ")
	if cleaned == "" || cleaned == "." || cleaned == ".." {
		return fallbackDownloadFilename
	}

	return cleaned
}
