import { cookies } from "next/headers";
import { NextResponse } from "next/server";

import { proxyJsonWithAuthCleanup, unauthorizedJson } from "@/app/api/_utils/auth";
import { createForwardHeaders } from "@/app/api/_utils/locale";
import { fetchApiUpstream } from "@/app/api/_utils/upstream";

// Escopo do enriquecimento por LLM na indexacao (tres toggles por usuario).
async function encaminhar(request: Request, method: "GET" | "PUT") {
    const token = (await cookies()).get("token")?.value;
    if (!token) {
        return unauthorizedJson();
    }
    const body = method === "PUT" ? await request.json().catch(() => null) : undefined;
    try {
        const response = await fetchApiUpstream("/user/indexing-preferences", {
            method,
            headers: await createForwardHeaders(request, method === "PUT" ? { token, contentType: "application/json" } : { token }),
            ...(body !== undefined ? { body: JSON.stringify(body ?? {}) } : {}),
            cache: "no-store",
        });
        const payload = await response.json().catch(() => ({ message: "Nao foi possivel processar a resposta da API." }));
        return proxyJsonWithAuthCleanup(payload, response.status);
    } catch {
        return NextResponse.json({ message: "Erro ao acessar o escopo da indexacao." }, { status: 502 });
    }
}

export async function GET(request: Request) {
    return encaminhar(request, "GET");
}

export async function PUT(request: Request) {
    return encaminhar(request, "PUT");
}
