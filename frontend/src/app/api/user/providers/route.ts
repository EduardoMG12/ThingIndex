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
        const response = await fetch(`${API_URL}/user/providers`, {
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
            { message: "Erro ao consultar providers." },
            { status: 502 }
        );
    }
}

export async function POST(request: Request) {
    const token = (await cookies()).get("token")?.value;

    if (!token) {
        return unauthorizedJson();
    }

    const body = await request.json().catch(() => null);

    try {
        const response = await fetch(`${API_URL}/user/providers`, {
            method: "POST",
            headers: await createForwardHeaders(request, { token, contentType: "application/json" }),
            body: JSON.stringify(body ?? {}),
            cache: "no-store",
        });

        const payload = await response.json().catch(() => ({
            message: "Nao foi possivel processar a resposta da API.",
        }));

        return proxyJsonWithAuthCleanup(payload, response.status);
    } catch {
        return NextResponse.json(
            { message: "Erro ao salvar provider." },
            { status: 502 }
        );
    }
}
