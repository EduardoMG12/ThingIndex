import { describe, expect, it } from "vitest";

import {
    buildContentDispositionHeader,
    extractFilenameFromDisposition,
    isSafePreviewContentType,
    normalizeContentType,
    sanitizeFilename,
} from "@/lib/http/content-disposition";

describe("content disposition helpers", () => {
    it("normalizes content types without parameters", () => {
        expect(normalizeContentType("application/pdf; charset=binary")).toBe("application/pdf");
    });

    it("allows only safe preview content types", () => {
        expect(isSafePreviewContentType("application/pdf")).toBe(true);
        expect(isSafePreviewContentType("text/plain; charset=utf-8")).toBe(true);
        expect(isSafePreviewContentType("text/html")).toBe(false);
    });

    it("sanitizes filenames from path segments and control characters", () => {
        expect(sanitizeFilename("..\\segredo\r\nfinal.pdf")).toBe("segredo final.pdf");
        expect(sanitizeFilename(" \n ")).toBe("documento");
    });

    it("extracts and sanitizes filenames from content disposition", () => {
        expect(
            extractFilenameFromDisposition(`attachment; filename*=UTF-8''..%2Frelatorio%0D%0Afinal.pdf`, "fallback.pdf")
        ).toBe("relatorio final.pdf");
    });

    it("builds safe content disposition headers", () => {
        expect(buildContentDispositionHeader("attachment", `attachment; filename="../evil.pdf"`)).toBe(
            "attachment; filename*=UTF-8''evil.pdf"
        );
    });
});
