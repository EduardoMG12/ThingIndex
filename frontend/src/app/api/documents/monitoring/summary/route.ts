import { cookies } from "next/headers";

import { proxyJsonWithAuthCleanup, unauthorizedJson } from "@/app/api/_utils/auth";
import { localizedRouteError, parseProxyJsonWithFallback } from "@/app/api/_utils/messages";
import { createForwardHeaders } from "@/app/api/_utils/locale";

const API_URL = process.env.API_INTERNAL_URL || "http://api:8080/api/v1";

export async function GET(request: Request) {
    const token = (await cookies()).get("token")?.value;

    if (!token) {
        return unauthorizedJson();
    }

    try {
        const response = await fetch(`${API_URL}/documents/monitoring/summary`, {
            method: "GET",
            headers: await createForwardHeaders(request, { token }),
            cache: "no-store",
        });

        const payload = await parseProxyJsonWithFallback(response, request);

        return proxyJsonWithAuthCleanup(payload, response.status);
    } catch {
        return localizedRouteError(request, "apiRouteErrors.documents.monitoringSummary", 502);
    }
}
