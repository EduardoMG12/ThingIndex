"use client";

import { useState } from "react";
import { useDropzone } from "react-dropzone";
import { ArrowRight, FolderOpen, Tag } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import CategoryCombobox from "@/components/dashboard/CategoryCombobox";
import CollectionCombobox from "@/components/dashboard/CollectionCombobox";
import UploadDropzone from "@/components/dashboard/upload/UploadDropzone";
import UploadFileList from "@/components/dashboard/upload/UploadFileList";
import { useDocumentUpload } from "@/hooks/use-document-upload";

const UploadTab = () => {
    const [defaultCategory, setDefaultCategory] = useState("");
    const [defaultCollection, setDefaultCollection] = useState("");
    const { files, isProcessing, addFiles, removeFile, processQueue } = useDocumentUpload();
    const hasFilesWithoutCollection = files.some((file) => !file.collection.trim());

    const onDrop = (accepted: File[]) => {
        if (!defaultCollection.trim()) {
            toast.error("Informe uma collection antes de adicionar arquivos.");
            return;
        }

        addFiles(accepted, {
            category: defaultCategory,
            collection: defaultCollection,
        });
    };

    const { getRootProps, getInputProps, isDragActive } = useDropzone({
        onDrop,
        disabled: isProcessing,
        accept: {
            "application/pdf": [".pdf"],
            "text/html": [".html", ".htm"],
            "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": [".xlsx"],
        },
    });

    const handleProcessAll = async () => {
        if (hasFilesWithoutCollection) {
            toast.error("Todos os arquivos precisam de uma collection antes do envio.");
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
                            <FolderOpen className="h-3 w-3" /> Categoria
                        </Label>
                        <CategoryCombobox value={defaultCategory} onChange={setDefaultCategory} />
                        <p className="text-[10px] text-muted-foreground">Opcional. Em branco, o arquivo sobe sem categoria.</p>
                    </div>
                    <div className="space-y-1">
                        <Label className="flex items-center gap-1 text-[9px] uppercase tracking-wider text-muted-foreground">
                            <Tag className="h-3 w-3" /> Collection *
                        </Label>
                        <CollectionCombobox value={defaultCollection} onChange={setDefaultCollection} />
                        <p className="text-[10px] text-muted-foreground">Obrigatoria. A collection centraliza o contexto relacional entre arquivos.</p>
                    </div>
                </div>
            </div>

            <div className="p-4 space-y-4">
                <UploadDropzone
                    getRootProps={getRootProps}
                    getInputProps={getInputProps}
                    isDragActive={isDragActive}
                    disabled={isProcessing}
                />

                <UploadFileList files={files} isProcessing={isProcessing} onRemove={removeFile} />

                {files.length > 0 && (
                    <Button onClick={handleProcessAll} disabled={isProcessing || hasFilesWithoutCollection} className="w-full gap-2">
                        {isProcessing ? (
                            <ArrowRight className="h-4 w-4 animate-pulse" />
                        ) : (
                            <ArrowRight className="h-4 w-4" />
                        )}
                        {isProcessing ? "Enviando arquivos..." : `Enviar ${files.length} documento${files.length > 1 ? "s" : ""}`}
                    </Button>
                )}
                {files.length > 0 && hasFilesWithoutCollection && (
                    <p className="text-[10px] text-destructive">
                        Defina uma collection para todos os arquivos antes de enviar.
                    </p>
                )}
            </div>
        </div>
    );
};

export default UploadTab;
