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
        const response = await fetchApiUpstream("/documents/categories", {
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
            { message: "Erro ao consultar categorias." },
            { status: 502 }
        );
    }
}

export async function POST(request: Request) {
    const token = (await cookies()).get("token")?.value;

    if (!token) {
        return unauthorizedJson();
    }

    const body = await request.json().catch(() => null) as { name?: string; description?: string } | null;
    const name = body?.name?.trim();

    if (!name) {
        return NextResponse.json(
            { message: "Nome da categoria e obrigatorio." },
            { status: 400 }
        );
    }

    try {
        const response = await fetchApiUpstream("/documents/categories", {
            method: "POST",
            headers: await createForwardHeaders(request, { token, contentType: "application/json" }),
            body: JSON.stringify({
                name,
                description: body?.description ?? "",
            }),
            cache: "no-store",
        });

        const payload = await response.json().catch(() => ({
            message: "Nao foi possivel processar a resposta da API.",
        }));

        return proxyJsonWithAuthCleanup(payload, response.status);
    } catch {
        return NextResponse.json(
            { message: "Erro ao criar categoria." },
            { status: 502 }
        );
    }
}
