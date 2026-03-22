import type { AuditCountBucket } from "@/lib/api/audit";

export function formatAuditDateTime(value: string | undefined, locale: string) {
    if (!value) {
        return "-";
    }

    return new Intl.DateTimeFormat(locale, {
        dateStyle: "short",
        timeStyle: "short",
    }).format(new Date(value));
}

export function formatDocumentAuditAction(action?: string) {
    switch (action) {
        case "DOCUMENT_UPLOAD":
            return "Upload";
        case "DOCUMENT_RETRY_INDEX":
            return "Retry / reindexacao";
        case "DOCUMENT_DELETE":
            return "Exclusao";
        default:
            return action || "Evento";
    }
}

export function formatProviderAuditAction(action?: string) {
    switch (action) {
        case "PROVIDER_CONFIG_SAVE":
            return "Salvar configuracao";
        case "PROVIDER_ACTIVATION_CHANGE":
            return "Alterar ativacao";
        default:
            return action || "Evento";
    }
}

export function formatAuditStatus(status?: string) {
    switch (status) {
        case "SUCCESS":
            return "Sucesso";
        case "FAILED":
            return "Falha";
        default:
            return status || "Desconhecido";
    }
}

export function getAuditBucketCount(buckets: AuditCountBucket[] | undefined, key: string) {
    return buckets?.find((bucket) => bucket.key === key)?.count ?? 0;
}

export function formatAuditActorType(actorType?: string) {
    switch (actorType) {
        case "USER":
            return "Usuario";
        case "SYSTEM":
            return "Sistema";
        case "SERVICE":
            return "Servico";
        case "WORKER":
            return "Worker";
        default:
            return actorType || "-";
    }
}

export function formatAuditActor(actorType?: string, actorId?: number) {
    if (!actorType && typeof actorId !== "number") {
        return "-";
    }

    if (typeof actorId === "number") {
        return `${formatAuditActorType(actorType)} #${actorId}`;
    }

    return formatAuditActorType(actorType);
}

function stringifyMetadataValue(value: unknown) {
    if (value === null || value === undefined) {
        return null;
    }
    if (typeof value === "string" || typeof value === "number" || typeof value === "boolean") {
        return String(value);
    }
    return null;
}

export function getAuditMetadataEntries(metadata: Record<string, unknown>, maxItems = 4) {
    return Object.entries(metadata)
        .map(([key, value]) => {
            const normalized = stringifyMetadataValue(value);
            if (normalized === null || normalized === "") {
                return null;
            }

            return {
                key,
                value: normalized,
            };
        })
        .filter((entry): entry is { key: string; value: string } => entry !== null)
        .slice(0, maxItems);
}

export function formatDateTimeLocalValue(value: string | undefined) {
    if (!value) {
        return "";
    }

    const date = new Date(value);
    if (Number.isNaN(date.getTime())) {
        return "";
    }

    const year = date.getFullYear();
    const month = String(date.getMonth() + 1).padStart(2, "0");
    const day = String(date.getDate()).padStart(2, "0");
    const hours = String(date.getHours()).padStart(2, "0");
    const minutes = String(date.getMinutes()).padStart(2, "0");

    return `${year}-${month}-${day}T${hours}:${minutes}`;
}

export function toRFC3339FromDateTimeLocal(value: string) {
    const trimmed = value.trim();
    if (!trimmed) {
        return undefined;
    }

    const date = new Date(trimmed);
    if (Number.isNaN(date.getTime())) {
        return undefined;
    }

    return date.toISOString();
}

export function parseAuditActorId(value: string) {
    const trimmed = value.trim();
    if (!trimmed) {
        return undefined;
    }

    const parsed = Number.parseInt(trimmed, 10);
    if (!Number.isInteger(parsed) || parsed <= 0) {
        return undefined;
    }

    return parsed;
}

export function normalizeAuditFilterValue(value: string) {
    const trimmed = value.trim().toUpperCase();
    if (!trimmed || trimmed === "ALL") {
        return "";
    }

    return trimmed;
}
