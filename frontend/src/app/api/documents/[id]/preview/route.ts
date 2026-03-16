import { cookies } from "next/headers";
import { NextResponse } from "next/server";

import { unauthorizedJson } from "@/app/api/_utils/auth";
import { createForwardHeaders } from "@/app/api/_utils/locale";

const API_URL = process.env.API_INTERNAL_URL || "http://api:8080/api/v1";

type RouteContext = {
    params: Promise<{
        id: string;
    }>;
};

export async function GET(request: Request, context: RouteContext) {
    const token = (await cookies()).get("token")?.value;

    if (!token) {
        return unauthorizedJson();
    }

    const { id } = await context.params;

    try {
        const response = await fetch(`${API_URL}/documents/${id}/content`, {
            method: "GET",
            headers: await createForwardHeaders(request, { token }),
            cache: "no-store",
        });

        if (response.status === 401) {
            return unauthorizedJson();
        }

        if (!response.ok || !response.body) {
            const payload = await response.json().catch(() => ({
                message: "Nao foi possivel carregar o preview do documento.",
            }));

            return NextResponse.json(payload, { status: response.status });
        }

        return new NextResponse(response.body, {
            status: response.status,
            headers: {
                "Content-Type": response.headers.get("content-type") || "application/octet-stream",
                "Content-Disposition": response.headers.get("content-disposition") || 'inline; filename="documento"',
                "Content-Length": response.headers.get("content-length") || "",
                "Cache-Control": "no-store",
                "X-Content-Type-Options": "nosniff",
            },
        });
    } catch {
        return NextResponse.json(
            { message: "Erro ao carregar o preview do documento." },
            { status: 502 }
        );
    }
}
