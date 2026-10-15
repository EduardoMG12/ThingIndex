import { cookies } from "next/headers";
import { NextResponse } from "next/server";

import { proxyJsonWithAuthCleanup, unauthorizedJson } from "@/app/api/_utils/auth";
import { createForwardHeaders } from "@/app/api/_utils/locale";

const API_URL = process.env.API_INTERNAL_URL || "http://api:8080/api/v1";

// Proxy JSON autenticado para /documents/collections/{id}/... da API.
export async function proxyCollectionJson(
    request: Request,
    subpath: string,
    init: { method: string; body?: unknown },
    fallbackMessage: string
) {
    const token = (await cookies()).get("token")?.value;
    if (!token) {
        return unauthorizedJson();
    }

    try {
        // Headers nao espalha com {...}: perde o Authorization.
        const headers = new Headers(await createForwardHeaders(request, { token }));
        if (init.body !== undefined) {
            headers.set("Content-Type", "application/json");
        }
        const response = await fetch(`${API_URL}/documents/collections/${subpath}`, {
            method: init.method,
            headers,
            body: init.body === undefined ? undefined : JSON.stringify(init.body),
            cache: "no-store",
        });
        const payload = await response.json().catch(() => ({
            message: "Nao foi possivel processar a resposta da API.",
        }));
        return proxyJsonWithAuthCleanup(payload, response.status);
    } catch {
        return NextResponse.json({ message: fallbackMessage }, { status: 502 });
    }
}
