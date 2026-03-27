"use client";

const DEFAULT_MAX_UPLOAD_SIZE_BYTES = 20 * 1024 * 1024;

export const UPLOAD_ACCEPTED_FILE_TYPES = {
    "application/pdf": [".pdf"],
    "text/html": [".html", ".htm"],
    "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": [".xlsx"],
} as const;

export type UploadValidationCode =
    | "file-too-large"
    | "file-invalid-type"
    | "file-empty";

type UploadFileLike = {
    name: string;
    size: number;
    type?: string;
};

function resolveMaxUploadSizeBytes() {
    const rawValue = process.env.NEXT_PUBLIC_MAX_UPLOAD_SIZE_BYTES;
    const parsed = Number.parseInt(rawValue ?? "", 10);
    return Number.isFinite(parsed) && parsed > 0 ? parsed : DEFAULT_MAX_UPLOAD_SIZE_BYTES;
}

export const MAX_UPLOAD_SIZE_BYTES = resolveMaxUploadSizeBytes();

const allowedMimeTypes = new Set<string>(Object.keys(UPLOAD_ACCEPTED_FILE_TYPES));
const allowedExtensions = new Set<string>(
    Object.values(UPLOAD_ACCEPTED_FILE_TYPES).flatMap((extensions) => extensions)
);

export function normalizeUploadValidationCode(code?: string | null): UploadValidationCode | "unknown" {
    switch (code) {
        case "file-too-large":
        case "file-invalid-type":
            return code;
        default:
            return "unknown";
    }
}

export function validateUploadFile(file: UploadFileLike): UploadValidationCode | null {
    if (file.size <= 0) {
        return "file-empty";
    }

    if (file.size > MAX_UPLOAD_SIZE_BYTES) {
        return "file-too-large";
    }

    const normalizedType = normalizeMimeType(file.type);
    const normalizedExtension = extractExtension(file.name);
    const matchesMimeType = normalizedType !== "" && allowedMimeTypes.has(normalizedType);
    const matchesExtension = normalizedExtension !== "" && allowedExtensions.has(normalizedExtension);

    if (!matchesMimeType && !matchesExtension) {
        return "file-invalid-type";
    }

    return null;
}

export function formatUploadSize(bytes: number, locale: string) {
    if (bytes < 1024) {
        return `${bytes} B`;
    }

    if (bytes < 1024 * 1024) {
        return `${new Intl.NumberFormat(locale, { maximumFractionDigits: 1 }).format(bytes / 1024)} KB`;
    }

    return `${new Intl.NumberFormat(locale, { maximumFractionDigits: 1 }).format(bytes / (1024 * 1024))} MB`;
}

function extractExtension(filename: string) {
    const lastDotIndex = filename.lastIndexOf(".");
    if (lastDotIndex < 0) {
        return "";
    }

    return filename.slice(lastDotIndex).toLowerCase();
}

function normalizeMimeType(value?: string) {
    return value?.trim().toLowerCase() ?? "";
}
