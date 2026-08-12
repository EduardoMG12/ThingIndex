import { cookies } from "next/headers";
import { NextResponse } from "next/server";

import { proxyJsonWithAuthCleanup, unauthorizedJson } from "@/app/api/_utils/auth";
import { createForwardHeaders } from "@/app/api/_utils/locale";
import { fetchApiUpstream } from "@/app/api/_utils/upstream";

type RouteContext = {
    params: Promise<{
        providerType: string;
    }>;
};

export async function PATCH(request: Request, context: RouteContext) {
    const token = (await cookies()).get("token")?.value;

    if (!token) {
        return unauthorizedJson();
    }

    const { providerType } = await context.params;
    const body = await request.json().catch(() => null);

    try {
        const response = await fetchApiUpstream(`/user/providers/${providerType}`, {
            method: "PATCH",
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
            { message: "Erro ao atualizar provider." },
            { status: 502 }
        );
    }
}
