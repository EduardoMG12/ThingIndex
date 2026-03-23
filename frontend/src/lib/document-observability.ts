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
export type DocumentOperationalActionKey = "manual" | "stalled" | "recentFailures" | "retrying" | "nominal";

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

export type DocumentOperationalActionSummary = {
    key: DocumentOperationalActionKey;
    filter: DocumentOperationalFilter;
    severity: DocumentOperationalHealth;
    count: number;
    documentIds: number[];
    sampleTitles: string[];
};

export type DocumentOperationalAnomalySummary = {
    kind: DocumentOperationalKind;
    filter: DocumentOperationalFilter;
    severity: DocumentOperationalHealth;
    count: number;
    oldestAgeMinutes?: number;
    documentIds: number[];
    sampleTitles: string[];
};

export type DocumentOperationalOverview = {
    snapshot: DocumentOperationalSnapshot;
    actions: DocumentOperationalActionSummary[];
    anomalies: DocumentOperationalAnomalySummary[];
    primaryAction: DocumentOperationalActionSummary;
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

export function createDocumentOperationalOverview(documents: Document[], now = new Date()): DocumentOperationalOverview {
    const snapshot = createDocumentOperationalSnapshot(documents, now);
    const anomalyGroups = new Map<DocumentOperationalKind, Document[]>();
    const anomalyOldestAgeMinutes = new Map<DocumentOperationalKind, number>();
    const actionGroups: Record<DocumentOperationalActionKey, Document[]> = {
        manual: [],
        stalled: [],
        recentFailures: [],
        retrying: [],
        nominal: [],
    };

    for (const document of documents) {
        const kind = getDocumentOperationalKind(document, now);
        if (!kind) {
            actionGroups.nominal.push(document);
            continue;
        }

        const groupedDocuments = anomalyGroups.get(kind);
        if (groupedDocuments) {
            groupedDocuments.push(document);
        } else {
            anomalyGroups.set(kind, [document]);
        }

        const ageMinutes = getDocumentOperationalAgeMinutes(document, now);
        if (ageMinutes !== null) {
            anomalyOldestAgeMinutes.set(kind, Math.max(anomalyOldestAgeMinutes.get(kind) ?? 0, ageMinutes));
        }

        if (requiresManualIntervention(document, now)) {
            actionGroups.manual.push(document);
        }
        if (kind === "stalledProcessing" || kind === "stalledQueue") {
            actionGroups.stalled.push(document);
        }
        if (isRecentOperationalFailure(document, now)) {
            actionGroups.recentFailures.push(document);
        }
        if (kind === "retrying") {
            actionGroups.retrying.push(document);
        }
    }

    const actions = [
        createActionSummary("manual", actionGroups.manual, now),
        createActionSummary("stalled", actionGroups.stalled, now),
        createActionSummary("recentFailures", actionGroups.recentFailures, now),
        createActionSummary("retrying", actionGroups.retrying, now),
    ].filter((value): value is DocumentOperationalActionSummary => value !== null);

    if (actions.length === 0) {
        actions.push(createNominalActionSummary(actionGroups.nominal, now));
    }

    const anomalies = Array.from(anomalyGroups.entries())
        .map(([kind, items]) => createAnomalySummary(kind, items, anomalyOldestAgeMinutes.get(kind), now))
        .sort(compareAnomalies);

    return {
        snapshot,
        actions,
        anomalies,
        primaryAction: actions[0],
    };
}

export function getDocumentOperationalFilterForKind(kind: DocumentOperationalKind): DocumentOperationalFilter {
    switch (kind) {
        case "orphaned":
        case "dlq":
        case "failed":
            return "manual";
        case "stalledProcessing":
        case "stalledQueue":
            return "stalled";
        case "retrying":
            return "retrying";
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

function createActionSummary(
    key: Exclude<DocumentOperationalActionKey, "nominal">,
    documents: Document[],
    now: Date
): DocumentOperationalActionSummary | null {
    if (documents.length === 0) {
        return null;
    }

    return {
        key,
        filter: key,
        severity: key === "retrying" || key === "recentFailures" ? "attention" : "critical",
        count: documents.length,
        documentIds: documents.map((document) => document.id),
        sampleTitles: createSampleTitles(documents, now),
    };
}

function createNominalActionSummary(documents: Document[], now: Date): DocumentOperationalActionSummary {
    return {
        key: "nominal",
        filter: "nominal",
        severity: "nominal",
        count: documents.length,
        documentIds: documents.map((document) => document.id),
        sampleTitles: createSampleTitles(documents, now),
    };
}

function createAnomalySummary(
    kind: DocumentOperationalKind,
    documents: Document[],
    oldestAgeMinutes: number | undefined,
    now: Date
): DocumentOperationalAnomalySummary {
    return {
        kind,
        filter: getDocumentOperationalFilterForKind(kind),
        severity: kind === "retrying" ? "attention" : "critical",
        count: documents.length,
        oldestAgeMinutes,
        documentIds: documents.map((document) => document.id),
        sampleTitles: createSampleTitles(documents, now),
    };
}

function createSampleTitles(documents: Document[], now: Date) {
    return [...documents]
        .sort(compareDocumentsByOperationalPriority(now))
        .slice(0, 3)
        .map((document) => document.title);
}

function compareAnomalies(left: DocumentOperationalAnomalySummary, right: DocumentOperationalAnomalySummary) {
    const severityDiff = getKindSeverityWeight(left.kind) - getKindSeverityWeight(right.kind);
    if (severityDiff !== 0) {
        return severityDiff;
    }

    if (left.count !== right.count) {
        return right.count - left.count;
    }

    return (right.oldestAgeMinutes ?? 0) - (left.oldestAgeMinutes ?? 0);
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
