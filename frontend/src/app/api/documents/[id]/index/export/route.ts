import { cookies } from "next/headers";
import { NextResponse } from "next/server";

import { proxyJsonWithAuthCleanup, unauthorizedJson } from "@/app/api/_utils/auth";
import { createForwardHeaders } from "@/app/api/_utils/locale";
import { localizedRouteError, parseProxyJsonWithFallback } from "@/app/api/_utils/messages";
import {
    buildContentDispositionHeader,
    normalizeContentType,
} from "@/lib/http/content-disposition";

const API_URL = process.env.API_INTERNAL_URL || "http://api:8080/api/v1";
const DOCUMENT_INDEX_EXPORT_ORIGIN_HEADER = "X-ThingIndex-Export-Origin";

type RouteContext = {
    params: Promise<{
        id: string;
    }>;
};

export async function GET(request: Request, context: RouteContext) {
    const token = (await cookies()).get("token")?.value;

    if (!token) {
        return unauthorizedJson(request);
    }

    const { id } = await context.params;
    const requestOrigin = new URL(request.url).origin;

    try {
        const response = await fetch(`${API_URL}/documents/${id}/index/export`, {
            method: "GET",
            headers: await createForwardHeaders(request, {
                token,
                headers: {
                    [DOCUMENT_INDEX_EXPORT_ORIGIN_HEADER]: requestOrigin,
                },
            }),
            cache: "no-store",
        });

        if (response.status === 401) {
            return unauthorizedJson(request);
        }

        if (!response.ok || !response.body) {
            const payload = await parseProxyJsonWithFallback(response, request, "apiRouteErrors.documents.indexExport");
            return proxyJsonWithAuthCleanup(payload, response.status);
        }

        const headers = new Headers();
        headers.set("Content-Type", normalizeContentType(response.headers.get("content-type"), "application/json"));
        headers.set(
            "Content-Disposition",
            buildContentDispositionHeader(
                "attachment",
                response.headers.get("content-disposition"),
                `document-${id}.thingindex-index.json`
            )
        );
        headers.set("Cache-Control", "no-store");
        headers.set("X-Content-Type-Options", "nosniff");
        headers.set("Referrer-Policy", "no-referrer");

        return new NextResponse(response.body, {
            status: response.status,
            headers,
        });
    } catch {
        return localizedRouteError(request, "apiRouteErrors.documents.indexExportLoad", 502);
    }
}
