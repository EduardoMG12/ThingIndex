"use client";

import { useCallback, useEffect, useRef, useState } from "react";

import { uploadDocument, type Document } from "@/lib/api/documents";

export type DocumentUploadFile = {
    id: string;
    file: File;
    name: string;
    size: number;
    type: string;
    status: "pending" | "uploading" | "done" | "error";
    progress: number;
    category: string;
    collection: string;
    error?: string;
};

type DocumentUploadDefaults = {
    category?: string;
    collection?: string;
};

export type DocumentUploadProcessOptions = {
    onFileSuccess?: (file: DocumentUploadFile, document: Document) => void;
    onFileError?: (file: DocumentUploadFile, error: Error) => void;
};

export function useDocumentUpload() {
    const [files, setFiles] = useState<DocumentUploadFile[]>([]);
    const [isProcessing, setIsProcessing] = useState(false);
    const filesRef = useRef<DocumentUploadFile[]>(files);

    useEffect(() => {
        filesRef.current = files;
    }, [files]);

    const addFiles = useCallback((incoming: File[], defaults: DocumentUploadDefaults) => {
        if (!incoming.length) return;

        const mappedFiles: DocumentUploadFile[] = incoming.map((file) => ({
            id: crypto.randomUUID(),
            file,
            name: file.name,
            size: file.size,
            type: file.type,
            status: "pending",
            progress: 0,
            category: defaults.category?.trim() ?? "",
            collection: defaults.collection?.trim() ?? "",
        }));

        setFiles((prev) => [...prev, ...mappedFiles]);
    }, []);

    const removeFile = useCallback((id: string) => {
        setFiles((prev) => prev.filter((file) => file.id !== id));
    }, []);

    const processQueue = useCallback(
        async (options?: DocumentUploadProcessOptions) => {
            if (isProcessing) {
                return;
            }

            const queuedFiles = filesRef.current.filter((file) => file.status !== "done");

            if (queuedFiles.length === 0) {
                return;
            }

            setIsProcessing(true);

            try {
                for (const currentFile of queuedFiles) {
                    setFiles((prev) =>
                        prev.map((file) =>
                            file.id === currentFile.id
                                ? { ...file, status: "uploading", progress: 45, error: undefined }
                                : file
                        )
                    );

                    try {
                        const uploadedDocument = await uploadDocument({
                            title: currentFile.name,
                            file: currentFile.file,
                            category: currentFile.category,
                            collection: currentFile.collection,
                        });

                        setFiles((prev) =>
                            prev.map((file) =>
                                file.id === currentFile.id
                                    ? {
                                          ...file,
                                          status: "done",
                                          progress: 100,
                                          category: uploadedDocument.category?.name ?? "",
                                          collection: uploadedDocument.collection ?? "",
                                      }
                                    : file
                            )
                        );

                        options?.onFileSuccess?.(currentFile, uploadedDocument);
                    } catch (error) {
                        const normalizedError =
                            error instanceof Error ? error : new Error("Nao foi possivel enviar o documento.");

                        setFiles((prev) =>
                            prev.map((file) =>
                                file.id === currentFile.id
                                    ? { ...file, status: "error", progress: 0, error: normalizedError.message }
                                    : file
                            )
                        );

                        options?.onFileError?.(currentFile, normalizedError);
                    }
                }
            } finally {
                setIsProcessing(false);
            }
        },
        [isProcessing]
    );

    return { files, isProcessing, addFiles, removeFile, processQueue };
}
