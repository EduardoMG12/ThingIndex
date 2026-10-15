
export type TranslateFn = (key: string, variables?: Record<string, string | number | boolean>) => string;

export function formatDocumentStatus(status: string | undefined, t: TranslateFn) {
    switch (status) {
        case "PENDING":
            return t("dashboard.upload.status.pending");
        case "UPLOADING":
            return t("dashboard.upload.status.uploading");
        case "QUEUED":
            return t("dashboard.upload.status.queued");
        case "PROCESSING":
            return t("dashboard.upload.status.processing");
        case "INDEXED":
            return t("dashboard.upload.status.indexed");
        case "UPLOAD_ERROR":
            return t("dashboard.upload.status.uploadError");
        case "FAILED":
            return t("dashboard.upload.status.failed");
        case "ORPHANED":
            return t("dashboard.upload.status.orphaned");
        default:
            return status || t("documentsPage.operational.statusUnknown");
    }
}

/** "ok" indexado, "erro" qualquer falha, "andamento" o resto (fila, processando...). */
export function tomDoStatus(status: string | undefined): "ok" | "erro" | "andamento" {
    if (status === "INDEXED") {
        return "ok";
    }
    if (!status || status.startsWith("FAILED") || status === "UPLOAD_ERROR" || status === "ORPHANED") {
        return "erro";
    }
    return "andamento";
}

/** Verdadeiro enquanto a indexacao ainda pode mudar de status sozinha. */
export function aindaIndexando(status: string | undefined): boolean {
    return tomDoStatus(status) === "andamento";
}
