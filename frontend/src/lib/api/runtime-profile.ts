import { apiErrorMessage } from "@/lib/api/error";
import { hasResponseData, type ApiErrorResponse, type ApiSuccessResponse } from "@/lib/api/types";

type RuntimeProfileAudienceApiRecord = {
    name?: string;
    description?: string;
};

type RuntimeProfileArtifactApiRecord = {
    label?: string;
    path?: string;
    kind?: string;
    summary?: string;
};

type RuntimeProfileApiRecord = {
    product_name?: string;
    tagline?: string;
    primary_use_case?: string;
    core_problem?: string;
    value_proposition?: string;
    target_users?: unknown;
    current_scope?: unknown;
    differentiators?: unknown;
    non_goals?: unknown;
    research_tracks?: unknown;
    artifact_refs?: unknown;
    last_reviewed_on?: string;
};

export type RuntimeProfileAudience = {
    name: string;
    description: string;
};

export type RuntimeProfileArtifact = {
    label: string;
    path: string;
    kind: string;
    summary: string;
};

export type RuntimeProfile = {
    product_name: string;
    tagline: string;
    primary_use_case: string;
    core_problem: string;
    value_proposition: string;
    target_users: RuntimeProfileAudience[];
    current_scope: string[];
    differentiators: string[];
    non_goals: string[];
    research_tracks: string[];
    artifact_refs: RuntimeProfileArtifact[];
    last_reviewed_on: string;
};

function normalizeStringList(value: unknown): string[] {
    if (!Array.isArray(value)) {
        return [];
    }

    return value.reduce<string[]>((items, item) => {
        if (typeof item === "string" && item.trim()) {
            items.push(item.trim());
        }
        return items;
    }, []);
}

function normalizeAudience(record: unknown): RuntimeProfileAudience | null {
    if (!record || typeof record !== "object") {
        return null;
    }

    const audience = record as RuntimeProfileAudienceApiRecord;
    if (!audience.name || !audience.description) {
        return null;
    }

    return {
        name: audience.name,
        description: audience.description,
    };
}

function normalizeArtifact(record: unknown): RuntimeProfileArtifact | null {
    if (!record || typeof record !== "object") {
        return null;
    }

    const artifact = record as RuntimeProfileArtifactApiRecord;
    if (!artifact.label || !artifact.path || !artifact.kind || !artifact.summary) {
        return null;
    }

    return {
        label: artifact.label,
        path: artifact.path,
        kind: artifact.kind,
        summary: artifact.summary,
    };
}

function normalizeRuntimeProfile(record: unknown): RuntimeProfile | null {
    if (!record || typeof record !== "object") {
        return null;
    }

    const profile = record as RuntimeProfileApiRecord;
    if (
        !profile.product_name ||
        !profile.tagline ||
        !profile.primary_use_case ||
        !profile.core_problem ||
        !profile.value_proposition ||
        !profile.last_reviewed_on
    ) {
        return null;
    }

    const target_users = Array.isArray(profile.target_users)
        ? profile.target_users.map(normalizeAudience).filter((item): item is RuntimeProfileAudience => item !== null)
        : [];

    const artifact_refs = Array.isArray(profile.artifact_refs)
        ? profile.artifact_refs.map(normalizeArtifact).filter((item): item is RuntimeProfileArtifact => item !== null)
        : [];

    return {
        product_name: profile.product_name,
        tagline: profile.tagline,
        primary_use_case: profile.primary_use_case,
        core_problem: profile.core_problem,
        value_proposition: profile.value_proposition,
        target_users,
        current_scope: normalizeStringList(profile.current_scope),
        differentiators: normalizeStringList(profile.differentiators),
        non_goals: normalizeStringList(profile.non_goals),
        research_tracks: normalizeStringList(profile.research_tracks),
        artifact_refs,
        last_reviewed_on: profile.last_reviewed_on,
    };
}

export async function fetchRuntimeProfile(signal?: AbortSignal): Promise<RuntimeProfile | null> {
    const response = await fetch("/api/runtime/profile", {
        method: "GET",
        cache: "no-store",
        signal,
    });

    const payload = (await response.json().catch(() => null)) as ApiSuccessResponse<unknown> | ApiErrorResponse | null;

    if (!response.ok) {
        throw new Error(payload?.message || apiErrorMessage("runtimeProfile"));
    }

    return normalizeRuntimeProfile(hasResponseData(payload) ? payload.data : null);
}
