import { NextResponse } from "next/server";
import { REQUEST_ID_HEADER_NAME } from "@/lib/api/request-id";

export function unauthorizedJson(message = "Usuario nao autenticado.") {
    const response = NextResponse.json({ message }, { status: 401 });
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
