"use client";

import type { DropzoneInputProps, DropzoneRootProps } from "react-dropzone";
import { Upload } from "lucide-react";

type UploadDropzoneProps = {
    getRootProps: (
        props?: DropzoneRootProps | undefined
    ) => DropzoneRootProps;
    getInputProps: (
        props?: DropzoneInputProps | undefined
    ) => DropzoneInputProps;
    isDragActive: boolean;
    disabled: boolean;
};

export default function UploadDropzone({
    getRootProps,
    getInputProps,
    isDragActive,
    disabled,
}: UploadDropzoneProps) {
    const highlightClasses = isDragActive
        ? "border-primary bg-primary/5 shadow-inner"
        : "border-border hover:border-primary/40 hover:bg-muted/30";

    const rootProps = getRootProps({
        className: [
            "rounded-lg border-2 border-dashed p-8 text-center transition-all",
            disabled ? "cursor-not-allowed opacity-60" : "cursor-pointer",
            highlightClasses,
        ].join(" "),
    });

    return (
        <div {...rootProps}>
            <input {...getInputProps()} />
            <Upload className="mx-auto mb-3 h-8 w-8 text-muted-foreground" />
            <p className="text-sm font-medium">Arraste seus documentos jurídicos</p>
            <p className="mt-1 text-xs text-muted-foreground">PDF, HTML ou Excel (Máx 20MB)</p>
        </div>
    );
}
