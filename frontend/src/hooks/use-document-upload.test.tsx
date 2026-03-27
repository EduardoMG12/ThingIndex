import { renderHook, waitFor, act } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { Document } from "@/lib/api/documents";
import { useDocumentUpload } from "@/hooks/use-document-upload";
import { createTestWrapper } from "@/test/test-utils";
import { fetchDocument, uploadDocument } from "@/lib/api/documents";

vi.mock("@/lib/api/documents", () => ({
    fetchDocument: vi.fn(),
    uploadDocument: vi.fn(),
}));

function createDocument(overrides: Partial<Document> = {}): Document {
    return {
        id: 1,
        user_id: 3,
        title: "sample-document.html",
        file_url: "/documents/1",
        file_path: "documents/sample-document.html",
        size: 128,
        mime_type: "text/html",
        status: "QUEUED",
        collection: "Case A",
        created_at: "2026-03-19T00:00:00Z",
        updated_at: "2026-03-19T00:00:00Z",
        ...overrides,
    };
}

describe("useDocumentUpload", () => {
    beforeEach(() => {
        vi.spyOn(globalThis.crypto, "randomUUID").mockReturnValue("upload-file-1");
        vi.mocked(uploadDocument).mockReset();
        vi.mocked(fetchDocument).mockReset();
    });

    it("adds valid files and reports validation rejections", async () => {
        const { result } = renderHook(() => useDocumentUpload(), {
            wrapper: createTestWrapper(),
        });

        const acceptedFile = new File(["<p>case</p>"], "case.html", { type: "text/html" });
        const rejectedFile = new File(["binary"], "case.exe", { type: "application/octet-stream" });

        let response:
            | {
                  acceptedCount: number;
                  rejections: Array<{ file: File; code: "file-too-large" | "file-invalid-type" | "file-empty" }>;
              }
            | undefined;

        act(() => {
            response = result.current.addFiles([acceptedFile, rejectedFile], {
                category: "Contracts",
                collection: "Case A",
            });
        });

        await waitFor(() => {
            expect(result.current.files).toHaveLength(1);
        });

        expect(response).toMatchObject({
            acceptedCount: 1,
            rejections: [{ code: "file-invalid-type" }],
        });
        expect(result.current.files[0]).toMatchObject({
            id: "upload-file-1",
            name: "case.html",
            status: "pending",
            category: "Contracts",
            collection: "Case A",
        });
    });

    it("uploads queued files and promotes them to indexed after polling", async () => {
        const onFileSuccess = vi.fn();
        vi.mocked(uploadDocument).mockResolvedValue(createDocument());
        vi.mocked(fetchDocument).mockResolvedValue(
            createDocument({
                status: "INDEXED",
                processing_stage: "DONE",
            })
        );

        const { result } = renderHook(() => useDocumentUpload(), {
            wrapper: createTestWrapper(),
        });

        act(() => {
            result.current.addFiles([new File(["<p>case</p>"], "case.html", { type: "text/html" })], {
                category: "Contracts",
                collection: "Case A",
            });
        });

        await waitFor(() => {
            expect(result.current.files).toHaveLength(1);
        });

        await act(async () => {
            await result.current.processQueue({ onFileSuccess });
        });

        await waitFor(() => {
            expect(result.current.files[0]).toMatchObject({
                status: "indexed",
                progress: 100,
                backendDocumentId: 1,
            });
        });

        expect(uploadDocument).toHaveBeenCalledWith({
            title: "case.html",
            file: expect.any(File),
            category: "Contracts",
            collection: "Case A",
        });
        expect(fetchDocument).toHaveBeenCalledWith(1, expect.any(AbortSignal));
        expect(onFileSuccess).toHaveBeenCalledWith(
            expect.objectContaining({ name: "case.html" }),
            expect.objectContaining({ id: 1 })
        );
    });

    it("surfaces upload failures without dropping the file from the queue", async () => {
        const onFileError = vi.fn();
        vi.mocked(uploadDocument).mockRejectedValue(new Error("Upload pipeline failed. Request ID: req-upload-123"));

        const { result } = renderHook(() => useDocumentUpload(), {
            wrapper: createTestWrapper(),
        });

        act(() => {
            result.current.addFiles([new File(["<p>case</p>"], "case.html", { type: "text/html" })], {
                collection: "Case A",
            });
        });

        await waitFor(() => {
            expect(result.current.files).toHaveLength(1);
        });

        await act(async () => {
            await result.current.processQueue({ onFileError });
        });

        await waitFor(() => {
            expect(result.current.files[0]).toMatchObject({
                status: "upload_error",
                progress: 0,
                error: "Upload pipeline failed. Request ID: req-upload-123",
            });
        });

        expect(onFileError).toHaveBeenCalledWith(
            expect.objectContaining({ name: "case.html" }),
            expect.objectContaining({ message: "Upload pipeline failed. Request ID: req-upload-123" })
        );
    });

    it("keeps the file queued and exposes a translated detail when polling fails", async () => {
        vi.mocked(uploadDocument).mockResolvedValue(createDocument());
        vi.mocked(fetchDocument).mockRejectedValue(new Error("network unavailable"));

        const { result } = renderHook(() => useDocumentUpload(), {
            wrapper: createTestWrapper(),
        });

        act(() => {
            result.current.addFiles([new File(["<p>case</p>"], "case.html", { type: "text/html" })], {
                collection: "Case A",
            });
        });

        await waitFor(() => {
            expect(result.current.files).toHaveLength(1);
        });

        await act(async () => {
            await result.current.processQueue();
        });

        await waitFor(() => {
            expect(result.current.files[0]).toMatchObject({
                status: "queued",
                statusDetail: "Indexing status unavailable: network unavailable",
            });
        });
    });
});
