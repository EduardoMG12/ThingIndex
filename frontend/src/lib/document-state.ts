import type { Document } from "@/lib/api/documents";

export type DocumentOperationalIssue =
    | {
        kind: "orphaned";
        title: string;
        description: string;
        detail?: string;
        hint: string;
      }
    | {
        kind: "retrying";
        title: string;
        description: string;
        detail?: string;
        hint: string;
      }
    | {
        kind: "dlq";
        title: string;
        description: string;
        detail?: string;
        hint: string;
      }
    | {
        kind: "failed";
        title: string;
        description: string;
        detail?: string;
        hint: string;
      };

export function isPreviewableDocument(document: Document) {
    return ["application/pdf", "text/plain"].includes(document.mime_type);
}

export function isOrphanedDocument(document: Document) {
    return document.status === "ORPHANED";
}

export function getDocumentOperationalIssue(document: Document): DocumentOperationalIssue | null {
    if (isOrphanedDocument(document)) {
        const detail = normalizeDetail(document.status_detail || document.last_processing_error);

        return {
            kind: "orphaned",
            title: "Arquivo original ausente",
            description: "O registro do documento permanece no catalogo, mas o binario original nao foi encontrado no storage configurado.",
            detail,
            hint: "Reenvie o arquivo para restaurar o original, ou exclua o registro se ele nao deve mais existir.",
        };
    }

    if (document.processing_stage === "RETRY_SCHEDULED") {
        const detail = normalizeDetail(document.status_detail || document.last_processing_error);

        return {
            kind: "retrying",
            title: "Nova tentativa automatica agendada",
            description: "A ultima tentativa falhou, mas o documento voltou para a fila e sera reprocessado automaticamente.",
            detail,
            hint: "Acompanhe o status antes de reenfileirar manualmente para evitar duplicidade operacional.",
        };
    }

    if (document.processing_stage === "MOVED_TO_DLQ") {
        const detail = normalizeDetail(document.status_detail || document.last_processing_error);

        return {
            kind: "dlq",
            title: "Falha esgotou retries e foi para a DLQ",
            description: "As tentativas automaticas foram esgotadas e este documento agora exige intervencao manual.",
            detail,
            hint: "Corrija a causa raiz antes de reenfileirar a indexacao.",
        };
    }

    if (document.status === "FAILED") {
        const detail = normalizeDetail(document.last_processing_error || document.status_detail);

        return {
            kind: "failed",
            title: "Falha de processamento",
            description: "A indexacao nao concluiu e este documento precisa de nova tentativa ou correcao operacional.",
            detail,
            hint: "Revise o erro abaixo, corrija a causa e reenfileire a indexacao quando fizer sentido.",
        };
    }

    return null;
}

function normalizeDetail(value?: string) {
    const trimmed = value?.trim();
    return trimmed ? trimmed : undefined;
}
