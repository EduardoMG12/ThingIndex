import { cookies } from "next/headers";
import { NextResponse } from "next/server";

import { proxyJsonWithAuthCleanup, unauthorizedJson } from "@/app/api/_utils/auth";
import { createForwardHeaders } from "@/app/api/_utils/locale";

const API_URL = process.env.API_INTERNAL_URL || "http://api:8080/api/v1";

type RouteContext = {
    params: Promise<{
        id: string;
    }>;
};

/**
 * Conexoes derivadas entre nos de documentos distintos da colecao.
 *
 * Repassa `node_id` e `min_score` para a API. O `min_score` viaja explicito em
 * vez de ficar fixo aqui porque o limiar e o que separa "existe evidencia" de
 * "devolvi o menos ruim" -- deixar o cliente escolher permite mostrar na tela
 * que subir o corte faz as conexoes desaparecerem, que e o comportamento
 * correto quando nao ha relacao real.
 */
export async function GET(request: Request, context: RouteContext) {
    const token = (await cookies()).get("token")?.value;

    if (!token) {
        return unauthorizedJson();
    }

    const { id } = await context.params;
    const requestUrl = new URL(request.url);

    const params = new URLSearchParams();
    const nodeId = requestUrl.searchParams.get("node_id");
    if (nodeId) {
        params.set("node_id", nodeId);
    }
    const minScore = requestUrl.searchParams.get("min_score");
    if (minScore) {
        params.set("min_score", minScore);
    }

    const query = params.toString();

    try {
        const response = await fetch(
            `${API_URL}/documents/collections/${id}/connections${query ? `?${query}` : ""}`,
            {
                headers: await createForwardHeaders(request, { token }),
                cache: "no-store",
            }
        );

        const payload = await response.json().catch(() => ({
            message: "Nao foi possivel processar a resposta da API.",
        }));

        return proxyJsonWithAuthCleanup(payload, response.status);
    } catch {
        return NextResponse.json(
            { message: "Erro ao buscar conexoes do no." },
            { status: 502 }
        );
    }
}
