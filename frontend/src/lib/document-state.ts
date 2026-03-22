import type { Document, DocumentProcessingAttempt } from "@/lib/api/documents";
import type { AppLocale } from "@/lib/i18n/config";
import { translate } from "@/lib/i18n/translate";

export const PROCESSING_STALLED_THRESHOLD_MINUTES = 30;
export const QUEUED_STALLED_THRESHOLD_MINUTES = 20;
export const RECENT_FAILURE_WINDOW_HOURS = 24;

export type DocumentOperationalKind =
    | "orphaned"
    | "retrying"
    | "dlq"
    | "failed"
    | "stalledProcessing"
    | "stalledQueue";

export type DocumentOperationalIssue = {
    kind: DocumentOperationalKind;
    title: string;
    description: string;
    detail?: string;
    hint: string;
    attempt?: DocumentProcessingAttempt;
    ageMinutes?: number;
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

export function isDocumentProcessingStalled(document: Document, now = new Date()) {
    if (document.status !== "PROCESSING") {
        return false;
    }

    const ageMinutes = getMinutesSince(document.index_started_at || document.updated_at || document.created_at, now);
    return ageMinutes !== null && ageMinutes >= PROCESSING_STALLED_THRESHOLD_MINUTES;
}

export function isDocumentQueueStalled(document: Document, now = new Date()) {
    if (document.status !== "QUEUED" || document.processing_stage === "RETRY_SCHEDULED") {
        return false;
    }

    const ageMinutes = getMinutesSince(document.updated_at || document.created_at, now);
    return ageMinutes !== null && ageMinutes >= QUEUED_STALLED_THRESHOLD_MINUTES;
}

export function getDocumentOperationalKind(document: Document, now = new Date()): DocumentOperationalKind | null {
    if (isOrphanedDocument(document)) {
        return "orphaned";
    }

    if (document.processing_stage === "MOVED_TO_DLQ") {
        return "dlq";
    }

    if (document.status === "FAILED") {
        return "failed";
    }

    if (isDocumentProcessingStalled(document, now)) {
        return "stalledProcessing";
    }

    if (isDocumentQueueStalled(document, now)) {
        return "stalledQueue";
    }

    if (document.processing_stage === "RETRY_SCHEDULED") {
        return "retrying";
    }

    return null;
}

export function getDocumentOperationalAgeMinutes(document: Document, now = new Date()) {
    const kind = getDocumentOperationalKind(document, now);

    switch (kind) {
        case "stalledProcessing":
            return getMinutesSince(document.index_started_at || document.updated_at || document.created_at, now);
        case "retrying":
        case "stalledQueue":
        case "orphaned":
        case "dlq":
        case "failed":
            return getMinutesSince(document.updated_at || document.created_at, now);
        default:
            return null;
    }
}

export function isRecentOperationalFailure(document: Document, now = new Date()) {
    const kind = getDocumentOperationalKind(document, now);
    if (!kind || !["orphaned", "dlq", "failed", "retrying"].includes(kind)) {
        return false;
    }

    const ageMinutes = getMinutesSince(document.updated_at || document.created_at, now);
    return ageMinutes !== null && ageMinutes <= RECENT_FAILURE_WINDOW_HOURS * 60;
}

export function requiresManualIntervention(document: Document, now = new Date()) {
    const kind = getDocumentOperationalKind(document, now);
    return kind === "orphaned" || kind === "dlq" || kind === "failed";
}

export function getDocumentOperationalIssue(document: Document, locale: AppLocale, now = new Date()): DocumentOperationalIssue | null {
    const latestAttempt = getLatestProcessingAttempt(document);
    const kind = getDocumentOperationalKind(document, now);
    if (!kind) {
        return null;
    }

    const detail = normalizeDetail(latestAttempt?.error_message || document.last_processing_error || document.status_detail);
    const ageMinutes = getDocumentOperationalAgeMinutes(document, now) ?? undefined;

    switch (kind) {
        case "orphaned":
            return {
                kind,
                title: translate(locale, "documentsPage.issues.orphaned.title"),
                description: translate(locale, "documentsPage.issues.orphaned.description"),
                detail,
                hint: translate(locale, "documentsPage.issues.orphaned.hint"),
                attempt: latestAttempt ?? undefined,
                ageMinutes,
            };
        case "retrying":
            return {
                kind,
                title: translate(locale, "documentsPage.issues.retrying.title"),
                description: translate(locale, "documentsPage.issues.retrying.description"),
                detail,
                hint: translate(locale, "documentsPage.issues.retrying.hint"),
                attempt: latestAttempt ?? undefined,
                ageMinutes,
            };
        case "dlq":
            return {
                kind,
                title: translate(locale, "documentsPage.issues.dlq.title"),
                description: translate(locale, "documentsPage.issues.dlq.description"),
                detail,
                hint: translate(locale, "documentsPage.issues.dlq.hint"),
                attempt: latestAttempt ?? undefined,
                ageMinutes,
            };
        case "failed":
            return {
                kind,
                title: translate(locale, "documentsPage.issues.failed.title"),
                description: translate(locale, "documentsPage.issues.failed.description"),
                detail,
                hint: translate(locale, "documentsPage.issues.failed.hint"),
                attempt: latestAttempt ?? undefined,
                ageMinutes,
            };
        case "stalledProcessing":
            return {
                kind,
                title: translate(locale, "documentsPage.issues.stalledProcessing.title"),
                description: translate(locale, "documentsPage.issues.stalledProcessing.description", {
                    minutes: PROCESSING_STALLED_THRESHOLD_MINUTES,
                }),
                detail,
                hint: translate(locale, "documentsPage.issues.stalledProcessing.hint"),
                attempt: latestAttempt ?? undefined,
                ageMinutes,
            };
        case "stalledQueue":
            return {
                kind,
                title: translate(locale, "documentsPage.issues.stalledQueue.title"),
                description: translate(locale, "documentsPage.issues.stalledQueue.description", {
                    minutes: QUEUED_STALLED_THRESHOLD_MINUTES,
                }),
                detail,
                hint: translate(locale, "documentsPage.issues.stalledQueue.hint"),
                attempt: latestAttempt ?? undefined,
                ageMinutes,
            };
    }
}

function normalizeDetail(value?: string) {
    const trimmed = value?.trim();
    return trimmed ? trimmed : undefined;
}

function getMinutesSince(value: string | undefined, now: Date) {
    if (!value) {
        return null;
    }

    const timestamp = new Date(value);
    if (Number.isNaN(timestamp.getTime())) {
        return null;
    }

    const diff = now.getTime() - timestamp.getTime();
    if (diff < 0) {
        return 0;
    }

    return Math.round(diff / 60_000);
}
