import { describe, expect, it } from "vitest";

import type { Document } from "@/lib/api/documents";
import {
    createDocumentOperationalSnapshot,
    filterDocumentsByOperationalView,
    matchesDocumentOperationalFilter,
} from "@/lib/document-observability";

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

describe("document-observability", () => {
    it("builds operational counts and a critical health status", () => {
        const now = new Date("2026-03-20T12:00:00Z");

        const snapshot = createDocumentOperationalSnapshot([
            createDocument({
                id: 1,
                title: "orphaned.pdf",
                status: "ORPHANED",
                updated_at: "2026-03-20T11:30:00Z",
            }),
            createDocument({
                id: 2,
                title: "processing.pdf",
                status: "PROCESSING",
                index_started_at: "2026-03-20T10:00:00Z",
                updated_at: "2026-03-20T10:00:00Z",
            }),
            createDocument({
                id: 3,
                title: "retry.pdf",
                status: "QUEUED",
                processing_stage: "RETRY_SCHEDULED",
                updated_at: "2026-03-20T11:50:00Z",
            }),
            createDocument({
                id: 4,
                title: "healthy.pdf",
                status: "INDEXED",
                updated_at: "2026-03-20T11:55:00Z",
            }),
        ], now);

        expect(snapshot.health).toBe("critical");
        expect(snapshot.counts).toMatchObject({
            total: 4,
            attention: 3,
            manual: 1,
            stalled: 1,
            retrying: 1,
            recentFailures: 2,
            nominal: 1,
        });
        expect(snapshot.highlights.map((item) => item.document.title)).toEqual([
            "orphaned.pdf",
            "processing.pdf",
            "retry.pdf",
        ]);
    });

    it("filters documents by operational slices and search query", () => {
        const now = new Date("2026-03-20T12:00:00Z");
        const documents = [
            createDocument({
                id: 1,
                title: "Retries report.pdf",
                collection: "Operations",
                status: "QUEUED",
                processing_stage: "RETRY_SCHEDULED",
                updated_at: "2026-03-20T11:50:00Z",
            }),
            createDocument({
                id: 2,
                title: "Healthy note.pdf",
                collection: "General",
                status: "INDEXED",
                updated_at: "2026-03-20T11:55:00Z",
            }),
            createDocument({
                id: 3,
                title: "Queued manual.pdf",
                collection: "Operations",
                status: "QUEUED",
                processing_stage: "QUEUED",
                updated_at: "2026-03-20T11:10:00Z",
            }),
        ];

        expect(matchesDocumentOperationalFilter(documents[0], "retrying", now)).toBe(true);
        expect(matchesDocumentOperationalFilter(documents[1], "nominal", now)).toBe(true);
        expect(matchesDocumentOperationalFilter(documents[2], "stalled", now)).toBe(true);

        expect(
            filterDocumentsByOperationalView(documents, "attention", "operations", now).map((item) => item.id)
        ).toEqual([3, 1]);

        expect(
            filterDocumentsByOperationalView(documents, "nominal", "healthy", now).map((item) => item.id)
        ).toEqual([2]);
    });
});
