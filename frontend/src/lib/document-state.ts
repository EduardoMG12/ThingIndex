import type { Document, DocumentProcessingAttempt } from "@/lib/api/documents";
import type { AppLocale } from "@/lib/i18n/config";
import { translate } from "@/lib/i18n/translate";

export type DocumentOperationalIssue =
    | {
        kind: "orphaned";
        title: string;
        description: string;
        detail?: string;
        hint: string;
        attempt?: DocumentProcessingAttempt;
      }
    | {
        kind: "retrying";
        title: string;
        description: string;
        detail?: string;
        hint: string;
        attempt?: DocumentProcessingAttempt;
      }
    | {
        kind: "dlq";
        title: string;
        description: string;
        detail?: string;
        hint: string;
        attempt?: DocumentProcessingAttempt;
      }
    | {
        kind: "failed";
        title: string;
        description: string;
        detail?: string;
        hint: string;
        attempt?: DocumentProcessingAttempt;
      };

export function isPreviewableDocument(document: Document) {
    return ["application/pdf", "text/plain"].includes(document.mime_type);
}

export function isOrphanedDocument(document: Document) {
    return document.status === "ORPHANED";
}

export function getLatestProcessingAttempt(document: Document): DocumentProcessingAttempt | null {
    return Array.isArray(document.processing_attempts) && document.processing_attempts.length > 0
        ? document.processing_attempts[0]
        : null;
}

export function getDocumentOperationalIssue(document: Document, locale: AppLocale): DocumentOperationalIssue | null {
    const latestAttempt = getLatestProcessingAttempt(document);

    if (isOrphanedDocument(document)) {
        const detail = normalizeDetail(latestAttempt?.error_message || document.status_detail || document.last_processing_error);

        return {
            kind: "orphaned",
            title: translate(locale, "documentsPage.issues.orphaned.title"),
            description: translate(locale, "documentsPage.issues.orphaned.description"),
            detail,
            hint: translate(locale, "documentsPage.issues.orphaned.hint"),
            attempt: latestAttempt ?? undefined,
        };
    }

    if (document.processing_stage === "RETRY_SCHEDULED") {
        const detail = normalizeDetail(latestAttempt?.error_message || document.last_processing_error || document.status_detail);

        return {
            kind: "retrying",
            title: translate(locale, "documentsPage.issues.retrying.title"),
            description: translate(locale, "documentsPage.issues.retrying.description"),
            detail,
            hint: translate(locale, "documentsPage.issues.retrying.hint"),
            attempt: latestAttempt ?? undefined,
        };
    }

    if (document.processing_stage === "MOVED_TO_DLQ") {
        const detail = normalizeDetail(latestAttempt?.error_message || document.last_processing_error || document.status_detail);

        return {
            kind: "dlq",
            title: translate(locale, "documentsPage.issues.dlq.title"),
            description: translate(locale, "documentsPage.issues.dlq.description"),
            detail,
            hint: translate(locale, "documentsPage.issues.dlq.hint"),
            attempt: latestAttempt ?? undefined,
        };
    }

    if (document.status === "FAILED") {
        const detail = normalizeDetail(latestAttempt?.error_message || document.last_processing_error || document.status_detail);

        return {
            kind: "failed",
            title: translate(locale, "documentsPage.issues.failed.title"),
            description: translate(locale, "documentsPage.issues.failed.description"),
            detail,
            hint: translate(locale, "documentsPage.issues.failed.hint"),
            attempt: latestAttempt ?? undefined,
        };
    }

    return null;
}

function normalizeDetail(value?: string) {
    const trimmed = value?.trim();
    return trimmed ? trimmed : undefined;
}
