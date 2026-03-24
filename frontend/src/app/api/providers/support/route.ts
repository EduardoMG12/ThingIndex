import { NextResponse } from "next/server";

import { createForwardHeaders } from "@/app/api/_utils/locale";
import { fetchApiUpstream } from "@/app/api/_utils/upstream";

export async function GET(request: Request) {
    try {
        const response = await fetchApiUpstream(`/providers/support${new URL(request.url).search}`, {
            method: "GET",
            headers: await createForwardHeaders(request),
            cache: "no-store",
        });

        const payload = await response.json().catch(() => ({
            message: "Nao foi possivel processar a resposta da API.",
        }));

        return NextResponse.json(payload, { status: response.status });
    } catch {
        return NextResponse.json(
            { message: "Erro ao consultar a matriz de suporte de providers." },
            { status: 502 }
        );
    }
}
