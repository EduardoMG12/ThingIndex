import { cookies } from "next/headers";
import { NextResponse } from "next/server";

import { unauthorizedJson } from "@/app/api/_utils/auth";
import { createForwardHeaders } from "@/app/api/_utils/locale";
import { fetchApiUpstream } from "@/app/api/_utils/upstream";

export async function GET(request: Request) {
    const token = (await cookies()).get("token")?.value;

    if (!token) {
        return unauthorizedJson();
    }

    try {
        const response = await fetchApiUpstream("/user/me/avatar", {
            method: "GET",
            headers: await createForwardHeaders(request, { token }),
            cache: "no-store",
        });

        if (response.status === 401) {
            return unauthorizedJson(request);
        }

        if (!response.ok || !response.body) {
            const payload = await response.json().catch(() => ({
                message: "Nao foi possivel carregar o avatar.",
            }));

            return NextResponse.json(payload, { status: response.status });
        }

        const headers = new Headers();
        headers.set("Content-Type", response.headers.get("content-type") || "application/octet-stream");
        headers.set("Cache-Control", "no-store");
        headers.set("X-Content-Type-Options", "nosniff");

        return new NextResponse(response.body, {
            status: response.status,
            headers,
        });
    } catch {
        return NextResponse.json(
            { message: "Erro ao carregar o avatar." },
            { status: 502 }
        );
    }
}
