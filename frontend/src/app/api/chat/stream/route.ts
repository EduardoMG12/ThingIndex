import { cookies } from "next/headers";
import { NextResponse } from "next/server";

import { unauthorizedJson } from "@/app/api/_utils/auth";
import { createForwardHeaders } from "@/app/api/_utils/locale";

const API_URL = process.env.API_INTERNAL_URL || "http://api:8080/api/v1";

export const runtime = "nodejs";
export const dynamic = "force-dynamic";

export async function POST(request: Request) {
    const token = (await cookies()).get("token")?.value;

    if (!token) {
        return unauthorizedJson();
    }

    try {
        const payload = await request.text();
        const response = await fetch(`${API_URL}/chat/stream`, {
            method: "POST",
            headers: await createForwardHeaders(request, {
                token,
                contentType: "application/json",
            }),
            body: payload,
            cache: "no-store",
            signal: request.signal,
        });

        if (response.status === 401) {
            return unauthorizedJson();
        }

        if (!response.ok || !response.body) {
            const errorPayload = await response.json().catch(() => ({
                message: "Nao foi possivel processar a resposta do chat.",
            }));
            return NextResponse.json(errorPayload, { status: response.status });
        }

        // Body do upstream direto, para o abort do cliente chegar ate o LLM
        // (com pipe manual o cancel se perdia).
        return new NextResponse(response.body, {
            status: response.status,
            headers: {
                "Content-Type": "text/event-stream; charset=utf-8",
                "Cache-Control": "no-store",
                "Connection": "keep-alive",
                "Content-Encoding": "none",
                "X-Accel-Buffering": "no",
            },
        });
    } catch {
        return NextResponse.json(
            { message: "Erro ao iniciar o streaming do chat." },
            { status: 502 }
        );
    }
}
