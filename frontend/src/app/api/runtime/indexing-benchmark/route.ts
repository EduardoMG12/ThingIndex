import { NextResponse } from "next/server";

import { createForwardHeaders } from "@/app/api/_utils/locale";

const API_URL = process.env.API_INTERNAL_URL || "http://api:8080/api/v1";

export async function GET(request: Request) {
    try {
        const response = await fetch(`${API_URL}/meta/indexing-benchmark`, {
            method: "GET",
            headers: await createForwardHeaders(request),
            cache: "no-store",
        });

        const payload = await response.json().catch(() => ({
            message: "Nao foi possivel processar a resposta do benchmark publico.",
        }));

        return NextResponse.json(payload, { status: response.status });
    } catch {
        return NextResponse.json(
            { message: "Erro ao consultar o benchmark publico de indexacao." },
            { status: 502 }
        );
    }
}
