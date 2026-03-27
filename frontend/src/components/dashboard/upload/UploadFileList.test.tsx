import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import UploadFileList from "@/components/dashboard/upload/UploadFileList";
import type { DocumentUploadFile } from "@/hooks/use-document-upload";
import { createTestWrapper } from "@/test/test-utils";

function createUploadFile(overrides: Partial<DocumentUploadFile> = {}): DocumentUploadFile {
    return {
        id: "file-1",
        file: new File(["<p>sample</p>"], "sample-document.html", { type: "text/html" }),
        name: "sample-document.html",
        size: 2048,
        type: "text/html",
        status: "retrying",
        progress: 60,
        category: "",
        collection: "",
        statusDetail: "Waiting before the next retry.",
        ...overrides,
    };
}

describe("UploadFileList", () => {
    it("renders nothing when the queue is empty", () => {
        const { container } = render(<UploadFileList files={[]} onRemove={vi.fn()} isProcessing={false} />, {
            wrapper: createTestWrapper(),
        });

        expect(container).toBeEmptyDOMElement();
    });

    it("renders translated fallbacks, status details and persisted issue hints", () => {
        render(
            <UploadFileList
                files={[
                    createUploadFile({
                        status: "orphaned",
                        backendDocumentId: 42,
                        error: "Original file missing from storage.",
                    }),
                ]}
                onRemove={vi.fn()}
                isProcessing={false}
            />,
            {
                wrapper: createTestWrapper(),
            }
        );

        expect(screen.getByText("sample-document.html")).toBeInTheDocument();
        expect(screen.getByText("Orphaned")).toBeInTheDocument();
        expect(screen.getByText("Uncategorized")).toBeInTheDocument();
        expect(screen.getByText("Collection required")).toBeInTheDocument();
        expect(screen.getByText("Original file missing from storage.")).toBeInTheDocument();
        expect(
            screen.getByText("The upload already succeeded. Resolve the issue in the documents library to avoid duplication.")
        ).toBeInTheDocument();
    });

    it("removes a queued file when the dismiss action is clicked", async () => {
        const user = userEvent.setup();
        const onRemove = vi.fn();

        render(<UploadFileList files={[createUploadFile()]} onRemove={onRemove} isProcessing={false} />, {
            wrapper: createTestWrapper(),
        });

        await user.click(screen.getByRole("button"));
        expect(onRemove).toHaveBeenCalledWith("file-1");
    });
});
