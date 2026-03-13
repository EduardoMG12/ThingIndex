import { NextResponse } from "next/server";

export function unauthorizedJson(message = "Usuario nao autenticado.") {
    const response = NextResponse.json({ message }, { status: 401 });
    response.cookies.delete("token");
    return response;
}

export function proxyJsonWithAuthCleanup(payload: unknown, status: number) {
    const response = NextResponse.json(payload, { status });

    if (status === 401) {
        response.cookies.delete("token");
    }

    return response;
}
