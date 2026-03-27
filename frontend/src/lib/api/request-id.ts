export const REQUEST_ID_HEADER_NAME = "X-Request-ID";

export function createRequestId() {
    if (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function") {
        return crypto.randomUUID();
    }

    return `req-${Date.now().toString(36)}`;
}
