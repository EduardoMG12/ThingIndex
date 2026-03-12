import { cookies } from "next/headers";
import { NextResponse } from "next/server";

const API_URL = process.env.API_INTERNAL_URL || "http://api:8080/api/v1";

type RouteContext = {
    params: Promise<{
        id: string;
    }>;
};

export async function PATCH(request: Request, context: RouteContext) {
    const token = (await cookies()).get("token")?.value;

    if (!token) {
        return NextResponse.json(
            { message: "Usuario nao autenticado." },
            { status: 401 }
        );
    }

    const { id } = await context.params;
    const body = await request.json().catch(() => null);

    try {
        const response = await fetch(`${API_URL}/documents/categories/${id}`, {
            method: "PATCH",
            headers: {
                "Content-Type": "application/json",
                Authorization: `Bearer ${token}`,
            },
            body: JSON.stringify(body ?? {}),
            cache: "no-store",
        });

        const payload = await response.json().catch(() => ({
            message: "Nao foi possivel processar a resposta da API.",
        }));

        return NextResponse.json(payload, { status: response.status });
    } catch {
        return NextResponse.json(
            { message: "Erro ao atualizar categoria." },
            { status: 502 }
        );
    }
}
