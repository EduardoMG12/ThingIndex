import { cookies } from "next/headers";
import { NextResponse } from "next/server";

import { proxyJsonWithAuthCleanup, unauthorizedJson } from "@/app/api/_utils/auth";
import { createForwardHeaders } from "@/app/api/_utils/locale";
import { fetchApiUpstream } from "@/app/api/_utils/upstream";

export async function GET(request: Request) {
    const token = (await cookies()).get("token")?.value;

    if (!token) {
        return unauthorizedJson();
    }

    try {
        const response = await fetchApiUpstream("/user/me", {
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
            { message: "Erro ao consultar o usuario autenticado." },
            { status: 502 }
        );
    }
}

export async function PUT(request: Request) {
    const token = (await cookies()).get("token")?.value;

    if (!token) {
        return unauthorizedJson();
    }

    const body = await request.formData().catch(() => null);

    try {
        const response = await fetchApiUpstream("/user/me", {
            method: "PUT",
            headers: await createForwardHeaders(request, { token }),
            body: body ?? undefined,
            cache: "no-store",
        });

        const payload = await response.json().catch(() => ({
            message: "Nao foi possivel processar a resposta da API.",
        }));

        return proxyJsonWithAuthCleanup(payload, response.status);
    } catch {
        return NextResponse.json(
            { message: "Erro ao atualizar o usuario autenticado." },
            { status: 502 }
        );
    }
}
