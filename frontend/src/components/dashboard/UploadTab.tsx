"use client";

import { useState } from "react";
import { type FileRejection, useDropzone } from "react-dropzone";
import { ArrowRight, FolderOpen, Tag } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import CategoryCombobox from "@/components/dashboard/CategoryCombobox";
import CollectionCombobox from "@/components/dashboard/CollectionCombobox";
import DashboardQuickGuide from "@/components/dashboard/DashboardQuickGuide";
import UploadDropzone from "@/components/dashboard/upload/UploadDropzone";
import UploadFileList from "@/components/dashboard/upload/UploadFileList";
import { type DocumentUploadValidationRejection, useDocumentUpload } from "@/hooks/use-document-upload";
import { useI18n } from "@/lib/i18n/provider";
import {
    formatUploadSize,
    MAX_UPLOAD_SIZE_BYTES,
    normalizeUploadValidationCode,
    UPLOAD_ACCEPTED_FILE_TYPES,
} from "@/lib/upload-policy";

const UploadTab = () => {
    const { locale, t } = useI18n();
    const [defaultCategory, setDefaultCategory] = useState("");
    const [defaultCollection, setDefaultCollection] = useState("");
    const { files, isProcessing, queuedCount, addFiles, removeFile, processQueue } = useDocumentUpload();
    const hasFilesWithoutCollection = files.some((file) => !file.collection.trim());

    const reportUploadRejections = (rejections: Array<DocumentUploadValidationRejection | { file: File; code: "unknown" }>) => {
        for (const rejection of rejections) {
            toast.error(buildUploadRejectionMessage(rejection.file, rejection.code, t, locale));
        }
    };

    const onDrop = (accepted: File[]) => {
        if (!defaultCollection.trim()) {
            toast.error(t("dashboard.upload.setCollectionFirst"));
            return;
        }

        const result = addFiles(accepted, {
            category: defaultCategory,
            collection: defaultCollection,
        });
        reportUploadRejections(result.rejections);
    };

    const onDropRejected = (rejections: FileRejection[]) => {
        reportUploadRejections(rejections.map((rejection) => ({
            file: rejection.file,
            code: normalizeUploadValidationCode(rejection.errors[0]?.code),
        })));
    };

    const { getRootProps, getInputProps, isDragActive } = useDropzone({
        onDrop,
        onDropRejected,
        disabled: isProcessing,
        maxSize: MAX_UPLOAD_SIZE_BYTES,
        accept: {
            ...UPLOAD_ACCEPTED_FILE_TYPES,
        },
    });

    const handleProcessAll = async () => {
        if (hasFilesWithoutCollection) {
            toast.error(t("dashboard.upload.allNeedCollection"));
            return;
        }

        await processQueue({
            onFileError: (file, error) => {
                toast.error(`${file.name}: ${error.message}`);
            },
        });
    };

    return (
        <div className="surface-elevated rounded-lg overflow-hidden border border-border">
            <div className="border-b border-border p-4 bg-muted/20">
                <div className="grid grid-cols-2 gap-3">
                    <div className="space-y-1">
                        <Label className="flex items-center gap-1 text-[9px] uppercase tracking-wider text-muted-foreground">
                            <FolderOpen className="h-3 w-3" /> {t("dashboard.upload.category")}
                        </Label>
                        <CategoryCombobox value={defaultCategory} onChange={setDefaultCategory} />
                        <p className="text-[10px] text-muted-foreground">{t("dashboard.upload.categoryHelp")}</p>
                    </div>
                    <div className="space-y-1">
                        <Label className="flex items-center gap-1 text-[9px] uppercase tracking-wider text-muted-foreground">
                            <Tag className="h-3 w-3" /> {t("dashboard.upload.collection")} *
                        </Label>
                        <CollectionCombobox value={defaultCollection} onChange={setDefaultCollection} />
                        <p className="text-[10px] text-muted-foreground">{t("dashboard.upload.collectionHelp")}</p>
                    </div>
                </div>
            </div>

            <div className="p-4 space-y-4">
                <div className="grid gap-4 lg:grid-cols-[minmax(0,1.15fr)_minmax(300px,0.85fr)] lg:items-stretch">
                    <UploadDropzone
                        getRootProps={getRootProps}
                        getInputProps={getInputProps}
                        isDragActive={isDragActive}
                        disabled={isProcessing}
                    />
                    <DashboardQuickGuide className="h-full" />
                </div>

                <UploadFileList files={files} isProcessing={isProcessing} onRemove={removeFile} />

                {queuedCount > 0 && (
                    <Button onClick={handleProcessAll} disabled={isProcessing || hasFilesWithoutCollection} className="w-full gap-2">
                        {isProcessing ? (
                            <ArrowRight className="h-4 w-4 animate-pulse" />
                        ) : (
                            <ArrowRight className="h-4 w-4" />
                        )}
                        {isProcessing
                            ? t("dashboard.upload.sending")
                            : queuedCount > 1
                                ? t("dashboard.upload.sendDocumentsPlural", { count: queuedCount })
                                : t("dashboard.upload.sendDocuments", { count: queuedCount })}
                    </Button>
                )}
                {files.length > 0 && hasFilesWithoutCollection && (
                    <p className="text-[10px] text-destructive">
                        {t("dashboard.upload.defineCollectionForAll")}
                    </p>
                )}
            </div>
        </div>
    );
};

export default UploadTab;

function buildUploadRejectionMessage(
    file: File,
    code: "unknown" | DocumentUploadValidationRejection["code"],
    t: (key: string, variables?: Record<string, string | number>) => string,
    locale: string
) {
    const variables = {
        name: file.name,
        maxSize: formatUploadSize(MAX_UPLOAD_SIZE_BYTES, locale),
    };

    switch (code) {
        case "file-too-large":
            return t("dashboard.upload.validation.fileTooLarge", variables);
        case "file-invalid-type":
            return t("dashboard.upload.validation.fileInvalidType", variables);
        case "file-empty":
            return t("dashboard.upload.validation.fileEmpty", variables);
        default:
            return t("dashboard.upload.validation.fileRejected", variables);
    }
}
