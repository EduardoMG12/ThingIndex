import { describe, expect, it } from "vitest";

import type { Document } from "@/lib/api/documents";
import {
    getDocumentOperationalIssue,
    isOrphanedDocument,
    isPreviewableDocument,
} from "@/lib/document-state";

function createDocument(overrides: Partial<Document> = {}): Document {
    return {
        id: 1,
        user_id: 7,
        title: "case.pdf",
        file_url: "/documents/1",
        file_path: "documents/case.pdf",
        size: 1024,
        mime_type: "application/pdf",
        status: "INDEXED",
        created_at: "2026-03-19T00:00:00Z",
        updated_at: "2026-03-19T00:00:00Z",
        ...overrides,
    };
}

describe("document-state", () => {
    it("identifies previewable documents", () => {
        expect(isPreviewableDocument(createDocument())).toBe(true);
        expect(isPreviewableDocument(createDocument({ mime_type: "text/plain" }))).toBe(true);
        expect(isPreviewableDocument(createDocument({ mime_type: "text/html" }))).toBe(false);
    });

    it("identifies orphaned documents", () => {
        expect(isOrphanedDocument(createDocument({ status: "ORPHANED" }))).toBe(true);
        expect(isOrphanedDocument(createDocument())).toBe(false);
    });

    it("builds an orphaned operational issue with normalized detail", () => {
        const issue = getDocumentOperationalIssue(
            createDocument({
                status: "ORPHANED",
                status_detail: "  original missing from bucket  ",
            })
        );

        expect(issue).toMatchObject({
            kind: "orphaned",
            title: "Arquivo original ausente",
            detail: "original missing from bucket",
        });
    });

    it("builds a retry scheduled issue when the document returned to the queue", () => {
        const issue = getDocumentOperationalIssue(
            createDocument({
                status: "QUEUED",
                processing_stage: "RETRY_SCHEDULED",
                last_processing_error: "temporary network failure",
            })
        );

        expect(issue).toMatchObject({
            kind: "retrying",
            title: "Nova tentativa automatica agendada",
            detail: "temporary network failure",
        });
    });

    it("builds a dlq issue when retries were exhausted", () => {
        const issue = getDocumentOperationalIssue(
            createDocument({
                status: "FAILED",
                processing_stage: "MOVED_TO_DLQ",
                status_detail: "retry budget exhausted",
            })
        );

        expect(issue).toMatchObject({
            kind: "dlq",
            title: "Falha esgotou retries e foi para a DLQ",
            detail: "retry budget exhausted",
        });
    });

    it("builds a failed issue for terminal processing errors", () => {
        const issue = getDocumentOperationalIssue(
            createDocument({
                status: "FAILED",
                last_processing_error: "parsing crashed",
            })
        );

        expect(issue).toMatchObject({
            kind: "failed",
            title: "Falha de processamento",
            detail: "parsing crashed",
        });
    });

    it("returns null when there is no operational issue", () => {
        expect(getDocumentOperationalIssue(createDocument())).toBeNull();
    });
});
