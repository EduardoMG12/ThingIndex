import type { Document } from "@/lib/api/documents";
import {
    getDocumentOperationalAgeMinutes,
    getDocumentOperationalKind,
    isRecentOperationalFailure,
    requiresManualIntervention,
    type DocumentOperationalKind,
} from "@/lib/document-state";

export const DOCUMENT_OPERATIONAL_FILTERS = [
    "all",
    "attention",
    "manual",
    "stalled",
    "retrying",
    "recentFailures",
    "nominal",
] as const;

export type DocumentOperationalFilter = (typeof DOCUMENT_OPERATIONAL_FILTERS)[number];
export type DocumentOperationalHealth = "nominal" | "attention" | "critical";

export type DocumentOperationalHighlight = {
    document: Document;
    kind: DocumentOperationalKind;
    ageMinutes?: number;
};

export type DocumentOperationalSnapshot = {
    counts: {
        total: number;
        attention: number;
        manual: number;
        stalled: number;
        retrying: number;
        recentFailures: number;
        nominal: number;
    };
    health: DocumentOperationalHealth;
    highlights: DocumentOperationalHighlight[];
};

export function createDocumentOperationalSnapshot(documents: Document[], now = new Date()): DocumentOperationalSnapshot {
    let attention = 0;
    let manual = 0;
    let stalled = 0;
    let retrying = 0;
    let recentFailures = 0;
    const highlights: DocumentOperationalHighlight[] = [];

    for (const document of documents) {
        const kind = getDocumentOperationalKind(document, now);
        if (!kind) {
            continue;
        }

        attention += 1;
        if (requiresManualIntervention(document, now)) {
            manual += 1;
        }
        if (kind === "stalledProcessing" || kind === "stalledQueue") {
            stalled += 1;
        }
        if (kind === "retrying") {
            retrying += 1;
        }
        if (isRecentOperationalFailure(document, now)) {
            recentFailures += 1;
        }

        highlights.push({
            document,
            kind,
            ageMinutes: getDocumentOperationalAgeMinutes(document, now) ?? undefined,
        });
    }

    highlights.sort(compareOperationalHighlights);

    const nominal = Math.max(documents.length - attention, 0);

    return {
        counts: {
            total: documents.length,
            attention,
            manual,
            stalled,
            retrying,
            recentFailures,
            nominal,
        },
        health:
            manual > 0 || stalled > 0
                ? "critical"
                : retrying > 0 || recentFailures > 0 || attention > 0
                    ? "attention"
                    : "nominal",
        highlights: highlights.slice(0, 6),
    };
}

export function filterDocumentsByOperationalView(
    documents: Document[],
    filter: DocumentOperationalFilter,
    query: string,
    now = new Date()
) {
    const normalizedQuery = query.trim().toLowerCase();

    return documents
        .filter((document) => {
            if (normalizedQuery && !matchesDocumentQuery(document, normalizedQuery)) {
                return false;
            }

            return matchesDocumentOperationalFilter(document, filter, now);
        })
        .sort(compareDocumentsByOperationalPriority(now));
}

export function matchesDocumentOperationalFilter(
    document: Document,
    filter: DocumentOperationalFilter,
    now = new Date()
) {
    const kind = getDocumentOperationalKind(document, now);

    switch (filter) {
        case "attention":
            return kind !== null;
        case "manual":
            return requiresManualIntervention(document, now);
        case "stalled":
            return kind === "stalledProcessing" || kind === "stalledQueue";
        case "retrying":
            return kind === "retrying";
        case "recentFailures":
            return isRecentOperationalFailure(document, now);
        case "nominal":
            return kind === null;
        default:
            return true;
    }
}

function matchesDocumentQuery(document: Document, normalizedQuery: string) {
    const haystack = [
        document.title,
        document.collection,
        document.category?.name,
        document.file_path,
        document.mime_type,
        document.status,
        document.processing_stage,
    ]
        .filter(Boolean)
        .join(" ")
        .toLowerCase();

    return haystack.includes(normalizedQuery);
}

function compareDocumentsByOperationalPriority(now: Date) {
    return (left: Document, right: Document) => {
        const severityDiff = getOperationalSeverityWeight(left, now) - getOperationalSeverityWeight(right, now);
        if (severityDiff !== 0) {
            return severityDiff;
        }

        return getDocumentTimestamp(right) - getDocumentTimestamp(left);
    };
}

function compareOperationalHighlights(left: DocumentOperationalHighlight, right: DocumentOperationalHighlight) {
    const severityDiff = getKindSeverityWeight(left.kind) - getKindSeverityWeight(right.kind);
    if (severityDiff !== 0) {
        return severityDiff;
    }

    return getDocumentTimestamp(right.document) - getDocumentTimestamp(left.document);
}

function getOperationalSeverityWeight(document: Document, now: Date) {
    return getKindSeverityWeight(getDocumentOperationalKind(document, now));
}

function getKindSeverityWeight(kind: DocumentOperationalKind | null) {
    switch (kind) {
        case "orphaned":
        case "dlq":
            return 0;
        case "failed":
            return 1;
        case "stalledProcessing":
            return 2;
        case "stalledQueue":
            return 3;
        case "retrying":
            return 4;
        default:
            return 5;
    }
}

function getDocumentTimestamp(document: Document) {
    const timestamp = new Date(document.updated_at || document.created_at).getTime();
    return Number.isNaN(timestamp) ? 0 : timestamp;
}
