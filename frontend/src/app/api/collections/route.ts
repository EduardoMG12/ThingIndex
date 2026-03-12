import { cookies } from "next/headers";
import { NextResponse } from "next/server";

const API_URL = process.env.API_INTERNAL_URL || "http://api:8080/api/v1";

export async function GET() {
    const token = (await cookies()).get("token")?.value;

    if (!token) {
        return NextResponse.json(
            { message: "Usuario nao autenticado." },
            { status: 401 }
        );
    }

    try {
        const response = await fetch(`${API_URL}/documents/collections`, {
            method: "GET",
            headers: {
                Authorization: `Bearer ${token}`,
            },
            cache: "no-store",
        });

        const payload = await response.json().catch(() => ({
            message: "Nao foi possivel processar a resposta da API.",
        }));

        return NextResponse.json(payload, { status: response.status });
    } catch {
        return NextResponse.json(
            { message: "Erro ao consultar collections." },
            { status: 502 }
        );
    }
}
