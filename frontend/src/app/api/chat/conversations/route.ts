import { cookies } from "next/headers";
import { NextResponse } from "next/server";

import { proxyJsonWithAuthCleanup, unauthorizedJson } from "@/app/api/_utils/auth";
import { createForwardHeaders } from "@/app/api/_utils/locale";

const API_URL = process.env.API_INTERNAL_URL || "http://api:8080/api/v1";

export async function GET(request: Request) {
    const token = (await cookies()).get("token")?.value;

    if (!token) {
        return unauthorizedJson();
    }

    try {
        const url = new URL(request.url);
        const query = url.searchParams.toString();
        const response = await fetch(`${API_URL}/chat/conversations${query ? `?${query}` : ""}`, {
            method: "GET",
            headers: await createForwardHeaders(request, { token }),
            cache: "no-store",
        });

        const payload = await response.json().catch(() => ({
            message: "Nao foi possivel processar a resposta da API.",
        }));

        return proxyJsonWithAuthCleanup(payload, response.status);
    } catch {
        return NextResponse.json(
            { message: "Erro ao consultar conversas do chat." },
            { status: 502 }
        );
    }
}

export async function POST(request: Request) {
    const token = (await cookies()).get("token")?.value;

    if (!token) {
        return unauthorizedJson();
    }

    try {
        const payload = await request.text();
        const response = await fetch(`${API_URL}/chat/conversations`, {
            method: "POST",
            headers: await createForwardHeaders(request, {
                token,
                contentType: "application/json",
            }),
            body: payload,
            cache: "no-store",
        });

        const responsePayload = await response.json().catch(() => ({
            message: "Nao foi possivel processar a resposta da API.",
        }));

        return proxyJsonWithAuthCleanup(responsePayload, response.status);
    } catch {
        return NextResponse.json(
            { message: "Erro ao criar conversa do chat." },
            { status: 502 }
        );
    }
}
