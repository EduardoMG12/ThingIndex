const DEFAULT_CONTAINER_API_URL = "http://api:8080/api/v1";
const DEFAULT_HOST_API_URL = "http://localhost:8000/api/v1";

function normalizeApiBaseUrl(value?: string | null) {
    const trimmedValue = value?.trim();
    if (!trimmedValue) {
        return null;
    }

    return trimmedValue.replace(/\/+$/, "");
}

function normalizeApiPath(pathname: string) {
    return pathname.startsWith("/") ? pathname : `/${pathname}`;
}

export function getApiBaseCandidates() {
    const candidates = [
        normalizeApiBaseUrl(process.env.API_INTERNAL_URL),
        normalizeApiBaseUrl(process.env.NEXT_PUBLIC_API_URL),
        DEFAULT_CONTAINER_API_URL,
        DEFAULT_HOST_API_URL,
    ];

    return candidates.filter((value, index, items): value is string => Boolean(value) && items.indexOf(value) === index);
}

export function buildApiUpstreamUrl(pathname: string, baseUrl: string) {
    return `${baseUrl}${normalizeApiPath(pathname)}`;
}

export async function fetchApiUpstream(pathname: string, init?: RequestInit) {
    const candidates = getApiBaseCandidates();
    const errors: string[] = [];

    for (const baseUrl of candidates) {
        try {
            return await fetch(buildApiUpstreamUrl(pathname, baseUrl), init);
        } catch (error) {
            errors.push(error instanceof Error ? error.message : String(error));
        }
    }

    const attemptedUrls = candidates.map((baseUrl) => buildApiUpstreamUrl(pathname, baseUrl)).join(", ");
    throw new Error(`Failed to reach API upstream. Attempted: ${attemptedUrls}. Errors: ${errors.join(" | ")}`);
}
