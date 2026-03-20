import { describe, expect, it } from "vitest";

import type { Document } from "@/lib/api/documents";
import {
    getDocumentOperationalIssue,
    getLatestProcessingAttempt,
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
                processing_attempts: [
                    {
                        id: 5,
                        stage: "FAILED_PARSING_STRUCTURE",
                        failure_kind: "TRANSIENT",
                        disposition: "RETRY_SCHEDULED",
                        error_message: "temporary network failure",
                        retry_count: 1,
                        next_retry_attempt: 2,
                        max_retry_attempts: 3,
                        created_at: "2026-03-20T10:00:00Z",
                    },
                ],
            })
        );

        expect(issue).toMatchObject({
            kind: "retrying",
            title: "Nova tentativa automatica agendada",
            detail: "temporary network failure",
            attempt: {
                failure_kind: "TRANSIENT",
                next_retry_attempt: 2,
            },
        });
    });

    it("builds a dlq issue when retries were exhausted", () => {
        const issue = getDocumentOperationalIssue(
            createDocument({
                status: "FAILED",
                processing_stage: "MOVED_TO_DLQ",
                processing_attempts: [
                    {
                        id: 7,
                        stage: "FAILED_RESOLVING_STORAGE",
                        failure_kind: "TRANSIENT",
                        disposition: "MOVED_TO_DLQ",
                        detail: "retry budget exhausted",
                        error_message: "storage timeout",
                        retry_count: 3,
                        max_retry_attempts: 3,
                        created_at: "2026-03-20T10:05:00Z",
                    },
                ],
            })
        );

        expect(issue).toMatchObject({
            kind: "dlq",
            title: "Falha esgotou retries e foi para a DLQ",
            detail: "storage timeout",
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

    it("returns the most recent processing attempt when present", () => {
        const attempt = getLatestProcessingAttempt(
            createDocument({
                processing_attempts: [
                    {
                        id: 9,
                        stage: "FAILED_PERSISTING_INDEX",
                        failure_kind: "PERMANENT",
                        disposition: "FAILED_PERMANENT",
                        retry_count: 0,
                        created_at: "2026-03-20T10:06:00Z",
                    },
                ],
            })
        );

        expect(attempt).toMatchObject({
            id: 9,
            failure_kind: "PERMANENT",
            disposition: "FAILED_PERMANENT",
        });
    });
});
