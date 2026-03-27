"use client";

import { useCallback, useEffect, useRef, useState } from "react";

import { fetchDocument, uploadDocument, type Document } from "@/lib/api/documents";
import { useI18n } from "@/lib/i18n/provider";
import { type UploadValidationCode, validateUploadFile } from "@/lib/upload-policy";

const INDEX_STATUS_POLL_INTERVAL_MS = 2000;
const INDEX_STATUS_MAX_ATTEMPTS = 90;

type UploadLifecycleStatus =
    | "pending"
    | "uploading"
    | "queued"
    | "retrying"
    | "processing"
    | "indexed"
    | "upload_error"
    | "failed"
    | "orphaned";

export type DocumentUploadFile = {
    id: string;
    file: File;
    name: string;
    size: number;
    type: string;
    status: UploadLifecycleStatus;
    progress: number;
    category: string;
    collection: string;
    backendDocumentId?: number;
    processingStage?: string;
    statusDetail?: string;
    error?: string;
};

type DocumentUploadDefaults = {
    category?: string;
    collection?: string;
};

export type DocumentUploadValidationRejection = {
    file: File;
    code: UploadValidationCode;
};

export type DocumentUploadProcessOptions = {
    onFileSuccess?: (file: DocumentUploadFile, document: Document) => void;
    onFileError?: (file: DocumentUploadFile, error: Error) => void;
};

export function useDocumentUpload() {
    const { t } = useI18n();
    const [files, setFiles] = useState<DocumentUploadFile[]>([]);
    const [isProcessing, setIsProcessing] = useState(false);
    const filesRef = useRef<DocumentUploadFile[]>(files);
    const pollControllersRef = useRef<Map<string, AbortController>>(new Map());

    useEffect(() => {
        filesRef.current = files;
    }, [files]);

    useEffect(() => {
        const pollControllers = pollControllersRef.current;

        return () => {
            for (const controller of pollControllers.values()) {
                controller.abort();
            }
            pollControllers.clear();
        };
    }, []);

    const updateFileState = useCallback((id: string, updater: (file: DocumentUploadFile) => DocumentUploadFile) => {
        setFiles((prev) => prev.map((file) => (file.id === id ? updater(file) : file)));
    }, []);

    const mapDocumentStatus = useCallback((document: Document): Pick<DocumentUploadFile, "status" | "progress" | "processingStage" | "statusDetail" | "error"> => {
        switch (document.status) {
            case "INDEXED":
                return {
                    status: "indexed",
                    progress: 100,
                    processingStage: document.processing_stage,
                    statusDetail: document.status_detail,
                    error: undefined,
                };
            case "PROCESSING":
                return {
                    status: "processing",
                    progress: 70,
                    processingStage: document.processing_stage,
                    statusDetail: document.status_detail,
                    error: undefined,
                };
            case "ORPHANED":
                return {
                    status: "orphaned",
                    progress: 0,
                    processingStage: document.processing_stage,
                    statusDetail: document.status_detail,
                    error: document.last_processing_error || document.status_detail || undefined,
                };
            case "FAILED":
                return {
                    status: "failed",
                    progress: 0,
                    processingStage: document.processing_stage,
                    statusDetail: document.status_detail,
                    error: document.last_processing_error || document.status_detail || t("dashboard.upload.processingFailed"),
                };
            case "QUEUED":
                if (document.processing_stage === "RETRY_SCHEDULED") {
                    return {
                        status: "retrying",
                        progress: 60,
                        processingStage: document.processing_stage,
                        statusDetail: document.status_detail,
                        error: undefined,
                    };
                }

                return {
                    status: "queued",
                    progress: 55,
                    processingStage: document.processing_stage,
                    statusDetail: document.status_detail,
                    error: undefined,
                };
            default:
                return {
                    status: "queued",
                    progress: 55,
                    processingStage: document.processing_stage,
                    statusDetail: document.status_detail,
                    error: undefined,
                };
        }
    }, [t]);

    const monitorIndexing = useCallback(
        async (fileId: string, documentId: number) => {
            pollControllersRef.current.get(fileId)?.abort();

            const controller = new AbortController();
            pollControllersRef.current.set(fileId, controller);

            try {
                for (let attempt = 0; attempt < INDEX_STATUS_MAX_ATTEMPTS; attempt += 1) {
                    if (attempt > 0) {
                        await new Promise((resolve) => window.setTimeout(resolve, INDEX_STATUS_POLL_INTERVAL_MS));
                    }

                    if (controller.signal.aborted) {
                        return;
                    }

                    const latestDocument = await fetchDocument(documentId, controller.signal);
                    const nextState = mapDocumentStatus(latestDocument);

                    updateFileState(fileId, (file) => ({
                        ...file,
                        status: nextState.status,
                        progress: nextState.progress,
                        processingStage: nextState.processingStage,
                        statusDetail: nextState.statusDetail,
                        error: nextState.error,
                        category: latestDocument.category?.name ?? file.category,
                        collection: latestDocument.collection ?? file.collection,
                        backendDocumentId: latestDocument.id,
                    }));

                    if (latestDocument.status === "INDEXED" || latestDocument.status === "FAILED" || latestDocument.status === "ORPHANED") {
                        return;
                    }
                }

                updateFileState(fileId, (file) => ({
                    ...file,
                    status: file.status === "retrying" ? "retrying" : "processing",
                    progress: file.status === "retrying" ? Math.max(file.progress, 60) : Math.max(file.progress, 70),
                    error: undefined,
                    statusDetail: file.statusDetail || t("dashboard.upload.stillIndexing"),
                }));
            } catch (error) {
                if (controller.signal.aborted) {
                    return;
                }

                updateFileState(fileId, (file) => ({
                    ...file,
                    status: resolvePollingFallbackStatus(file.status),
                    progress: file.progress > 0 ? file.progress : 55,
                    error: undefined,
                    statusDetail:
                        error instanceof Error && error.message
                            ? t("dashboard.upload.statusUnavailable", { message: error.message })
                            : t("dashboard.upload.statusUnavailableFallback"),
                }));
            } finally {
                if (pollControllersRef.current.get(fileId) === controller) {
                    pollControllersRef.current.delete(fileId);
                }
            }
        },
        [mapDocumentStatus, t, updateFileState]
    );

    const addFiles = useCallback((incoming: File[], defaults: DocumentUploadDefaults) => {
        if (!incoming.length) {
            return { acceptedCount: 0, rejections: [] as DocumentUploadValidationRejection[] };
        }

        const acceptedFiles: DocumentUploadFile[] = [];
        const rejections: DocumentUploadValidationRejection[] = [];

        for (const file of incoming) {
            const validationCode = validateUploadFile(file);
            if (validationCode) {
                rejections.push({ file, code: validationCode });
                continue;
            }

            acceptedFiles.push({
                id: crypto.randomUUID(),
                file,
                name: file.name,
                size: file.size,
                type: file.type,
                status: "pending",
                progress: 0,
                category: defaults.category?.trim() ?? "",
                collection: defaults.collection?.trim() ?? "",
            });
        }

        if (acceptedFiles.length > 0) {
            setFiles((prev) => [...prev, ...acceptedFiles]);
        }

        return {
            acceptedCount: acceptedFiles.length,
            rejections,
        };
    }, []);

    const removeFile = useCallback((id: string) => {
        pollControllersRef.current.get(id)?.abort();
        pollControllersRef.current.delete(id);
        setFiles((prev) => prev.filter((file) => file.id !== id));
    }, []);

    const processQueue = useCallback(
        async (options?: DocumentUploadProcessOptions) => {
            if (isProcessing) {
                return;
            }

            const queuedFiles = filesRef.current.filter((file) => !file.backendDocumentId && (file.status === "pending" || file.status === "upload_error"));

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

                        const nextState = mapDocumentStatus(uploadedDocument);

                        updateFileState(currentFile.id, (file) => ({
                            ...file,
                            status: nextState.status,
                            progress: nextState.progress,
                            category: uploadedDocument.category?.name ?? "",
                            collection: uploadedDocument.collection ?? "",
                            backendDocumentId: uploadedDocument.id,
                            processingStage: nextState.processingStage,
                            statusDetail: nextState.statusDetail,
                            error: nextState.error,
                        }));

                        options?.onFileSuccess?.(currentFile, uploadedDocument);
                        void monitorIndexing(currentFile.id, uploadedDocument.id);
                    } catch (error) {
                        const normalizedError =
                            error instanceof Error ? error : new Error("Nao foi possivel enviar o documento.");

                        setFiles((prev) =>
                            prev.map((file) =>
                                file.id === currentFile.id
                                    ? { ...file, status: "upload_error", progress: 0, error: normalizedError.message }
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
        [isProcessing, mapDocumentStatus, monitorIndexing, updateFileState]
    );

    return { files, isProcessing, addFiles, removeFile, processQueue };
}

function resolvePollingFallbackStatus(status: UploadLifecycleStatus): UploadLifecycleStatus {
    switch (status) {
        case "indexed":
        case "failed":
        case "orphaned":
        case "retrying":
        case "processing":
            return status;
        default:
            return "queued";
    }
}
