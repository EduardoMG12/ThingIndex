const FALLBACK_FILENAME = "documento";

export function normalizeContentType(value: string | null | undefined, fallback = "application/octet-stream") {
    const normalized = value?.split(";")[0]?.trim().toLowerCase();
    return normalized || fallback;
}

export function isSafePreviewContentType(value: string | null | undefined) {
    const contentType = normalizeContentType(value);
    return contentType === "application/pdf" || contentType === "text/plain";
}

export function sanitizeFilename(value: string | null | undefined, fallback = FALLBACK_FILENAME) {
    const pathStripped = value?.split(/[\\/]/).pop() ?? "";
    const cleaned = pathStripped
        .replace(/[\u0000-\u001f\u007f]+/g, " ")
        .replace(/["<>:|?*;]+/g, "_")
        .replace(/\s+/g, " ")
        .trim();

    if (!cleaned || cleaned === "." || cleaned === "..") {
        return fallback;
    }

    return cleaned;
}

export function extractFilenameFromDisposition(disposition: string | null, fallback: string) {
    const safeFallback = sanitizeFilename(fallback);
    if (!disposition) {
        return safeFallback;
    }

    const encodedMatch = disposition.match(/filename\*=UTF-8''([^;]+)/i);
    if (encodedMatch?.[1]) {
        try {
            return sanitizeFilename(decodeURIComponent(encodedMatch[1]), safeFallback);
        } catch {
            return sanitizeFilename(encodedMatch[1], safeFallback);
        }
    }

    const quotedMatch = disposition.match(/filename="([^"]+)"/i);
    if (quotedMatch?.[1]) {
        return sanitizeFilename(quotedMatch[1], safeFallback);
    }

    const plainMatch = disposition.match(/filename=([^;]+)/i);
    if (plainMatch?.[1]) {
        return sanitizeFilename(plainMatch[1].trim(), safeFallback);
    }

    return safeFallback;
}

export function buildContentDispositionHeader(
    dispositionType: "inline" | "attachment",
    upstreamDisposition: string | null,
    fallback = FALLBACK_FILENAME
) {
    const filename = extractFilenameFromDisposition(upstreamDisposition, fallback);
    return `${dispositionType}; filename*=UTF-8''${encodeURIComponent(filename)}`;
}
