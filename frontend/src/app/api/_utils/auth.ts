import { NextResponse } from "next/server";

import { getApiRouteMessage } from "@/app/api/_utils/messages";
import { REQUEST_ID_HEADER_NAME } from "@/lib/api/request-id";

export async function unauthorizedJson(request?: Request, message?: string) {
    const resolvedMessage = message ?? await getApiRouteMessage(request, "apiRouteErrors.unauthorized");
    const response = NextResponse.json({ message: resolvedMessage }, { status: 401 });
    response.cookies.delete("token");
    return response;
}

export function proxyJsonWithAuthCleanup(payload: unknown, status: number, requestId?: string) {
    const response = NextResponse.json(payload, { status });
    if (requestId) {
        response.headers.set(REQUEST_ID_HEADER_NAME, requestId);
    }

    if (status === 401) {
        response.cookies.delete("token");
    }

    return response;
}
