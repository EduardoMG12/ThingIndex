import { describe, expect, it } from "vitest";

import {
    MAX_UPLOAD_SIZE_BYTES,
    formatUploadSize,
    normalizeUploadValidationCode,
    validateUploadFile,
} from "@/lib/upload-policy";

function createUploadFile(overrides: Partial<{ name: string; size: number; type: string }> = {}) {
    return {
        name: "case.pdf",
        size: 1024,
        type: "application/pdf",
        ...overrides,
    };
}

describe("upload-policy", () => {
    it("accepts supported mime types", () => {
        expect(validateUploadFile(createUploadFile())).toBeNull();
    });

    it("accepts supported extensions when the mime type is missing", () => {
        expect(
            validateUploadFile(
                createUploadFile({
                    name: "evidence.HTML",
                    type: "",
                })
            )
        ).toBeNull();
    });

    it("rejects empty files", () => {
        expect(validateUploadFile(createUploadFile({ size: 0 }))).toBe("file-empty");
    });

    it("rejects files larger than the configured upload limit", () => {
        expect(validateUploadFile(createUploadFile({ size: MAX_UPLOAD_SIZE_BYTES + 1 }))).toBe("file-too-large");
    });

    it("rejects unsupported files when neither mime type nor extension match", () => {
        expect(
            validateUploadFile(
                createUploadFile({
                    name: "malware.exe",
                    type: "application/octet-stream",
                })
            )
        ).toBe("file-invalid-type");
    });

    it("formats upload sizes using locale-aware units", () => {
        expect(formatUploadSize(512, "en-US")).toBe("512 B");
        expect(formatUploadSize(2048, "en-US")).toBe("2 KB");
        expect(formatUploadSize(3 * 1024 * 1024, "en-US")).toBe("3 MB");
    });

    it("normalizes unsupported dropzone error codes to unknown", () => {
        expect(normalizeUploadValidationCode("file-too-large")).toBe("file-too-large");
        expect(normalizeUploadValidationCode("file-invalid-type")).toBe("file-invalid-type");
        expect(normalizeUploadValidationCode("file-empty")).toBe("unknown");
        expect(normalizeUploadValidationCode(null)).toBe("unknown");
    });
});
