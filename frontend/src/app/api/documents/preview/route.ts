import { createHash, createHmac } from "crypto";
import { cookies } from "next/headers";
import { NextRequest, NextResponse } from "next/server";

import { unauthorizedJson } from "@/app/api/_utils/auth";

const DEFAULT_REGION = "us-east-1";
const DEFAULT_EXPIRATION_SECONDS = 300;

function sha256(value: string) {
    return createHash("sha256").update(value, "utf8").digest("hex");
}

function hmac(key: Buffer | string, value: string) {
    return createHmac("sha256", key).update(value, "utf8").digest();
}

function normalizeEndpoint(rawEndpoint: string) {
    const trimmed = rawEndpoint.trim();
    if (!trimmed) {
        throw new Error("MINIO_ENDPOINT nao configurado.");
    }

    const withProtocol = /^https?:\/\//i.test(trimmed) ? trimmed : `http://${trimmed}`;
    const parsed = new URL(withProtocol);

    if (parsed.hostname === "localhost" || parsed.hostname === "127.0.0.1") {
        parsed.hostname = "host.docker.internal";
    }

    return parsed;
}

function encodeObjectPath(path: string) {
    return path
        .split("/")
        .filter(Boolean)
        .map((segment) => encodeURIComponent(segment))
        .join("/");
}

function createPresignedObjectURL(objectPath: string) {
    const accessKey = process.env.MINIO_ACCESS_KEY?.trim();
    const secretKey = process.env.MINIO_SECRET_KEY?.trim();
    const rawEndpoint = process.env.MINIO_ENDPOINT?.trim();
    const useSSL = process.env.MINIO_USE_SSL?.trim().toLowerCase() === "true";

    if (!accessKey || !secretKey || !rawEndpoint) {
        throw new Error("Configuracao do MinIO incompleta no frontend.");
    }

    const endpoint = normalizeEndpoint(rawEndpoint);
    endpoint.protocol = useSSL ? "https:" : "http:";

    const now = new Date();
    const iso = now.toISOString().replace(/[:-]|\.\d{3}/g, "");
    const amzDate = `${iso.slice(0, 8)}T${iso.slice(8, 14)}Z`;
    const dateStamp = amzDate.slice(0, 8);
    const credentialScope = `${dateStamp}/${DEFAULT_REGION}/s3/aws4_request`;
    const canonicalUri = `/${encodeObjectPath(objectPath)}`;

    const queryEntries = [
        ["X-Amz-Algorithm", "AWS4-HMAC-SHA256"],
        ["X-Amz-Credential", `${accessKey}/${credentialScope}`],
        ["X-Amz-Date", amzDate],
        ["X-Amz-Expires", String(DEFAULT_EXPIRATION_SECONDS)],
        ["X-Amz-SignedHeaders", "host"],
    ] as const;

    const canonicalQuery = queryEntries
        .map(([key, value]) => `${encodeURIComponent(key)}=${encodeURIComponent(value)}`)
        .join("&");

    const canonicalHeaders = `host:${endpoint.host}\n`;
    const canonicalRequest = [
        "GET",
        canonicalUri,
        canonicalQuery,
        canonicalHeaders,
        "host",
        "UNSIGNED-PAYLOAD",
    ].join("\n");

    const stringToSign = [
        "AWS4-HMAC-SHA256",
        amzDate,
        credentialScope,
        sha256(canonicalRequest),
    ].join("\n");

    const signingKey = hmac(
        hmac(
            hmac(
                hmac(`AWS4${secretKey}`, dateStamp),
                DEFAULT_REGION
            ),
            "s3"
        ),
        "aws4_request"
    );

    const signature = createHmac("sha256", signingKey).update(stringToSign, "utf8").digest("hex");

    endpoint.pathname = canonicalUri;
    endpoint.search = `${canonicalQuery}&X-Amz-Signature=${signature}`;

    return endpoint.toString();
}

function canInlinePreview(contentType: string) {
    return ["application/pdf", "text/html", "text/plain"].includes(contentType);
}

export async function GET(request: NextRequest) {
    const token = (await cookies()).get("token")?.value;
    if (!token) {
        return unauthorizedJson();
    }

    const objectPath = request.nextUrl.searchParams.get("path")?.trim();
    const contentType = request.nextUrl.searchParams.get("contentType")?.trim() || "application/octet-stream";
    const title = request.nextUrl.searchParams.get("title")?.trim() || "documento";

    if (!objectPath) {
        return NextResponse.json({ message: "Path do documento nao informado." }, { status: 400 });
    }

    try {
        const signedURL = createPresignedObjectURL(objectPath);
        const upstream = await fetch(signedURL, { cache: "no-store" });

        if (!upstream.ok || !upstream.body) {
            return NextResponse.json(
                { message: "Nao foi possivel carregar o preview do documento." },
                { status: upstream.status || 502 }
            );
        }

        return new NextResponse(upstream.body, {
            status: upstream.status,
            headers: {
                "Content-Type": upstream.headers.get("content-type") || contentType,
                "Content-Disposition": `${canInlinePreview(contentType) ? "inline" : "attachment"}; filename="${encodeURIComponent(title)}"`,
                "Cache-Control": "no-store",
            },
        });
    } catch {
        return NextResponse.json(
            { message: "Erro ao gerar preview do documento." },
            { status: 502 }
        );
    }
}
