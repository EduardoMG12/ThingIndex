import { cookies } from "next/headers";
import { NextResponse } from "next/server";

import { proxyJsonWithAuthCleanup, unauthorizedJson } from "@/app/api/_utils/auth";
import { createForwardHeaders } from "@/app/api/_utils/locale";

const API_URL = process.env.API_INTERNAL_URL || "http://api:8080/api/v1";

/**
 * Proxy JSON autenticado para /documents/collections/{id}/... da API. As
 * rotas de anotacao do workspace (marcas e legenda) sao todas iguais fora o
 * caminho e o metodo; o que varia fica no chamador.
 */
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
        // createForwardHeaders devolve um Headers: espalhar com {...} perde tudo
        // (inclusive o Authorization) -- foi assim que o primeiro PUT deu 401.
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
