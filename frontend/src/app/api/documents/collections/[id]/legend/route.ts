import { proxyCollectionJson } from "@/app/api/documents/collections/[id]/_proxy";

type RouteContext = { params: Promise<{ id: string }> };

export async function GET(request: Request, context: RouteContext) {
    const { id } = await context.params;
    return proxyCollectionJson(request, `${id}/legend`, { method: "GET" }, "Erro ao buscar a legenda da collection.");
}

export async function PUT(request: Request, context: RouteContext) {
    const { id } = await context.params;
    const body = await request.json().catch(() => ({}));
    return proxyCollectionJson(request, `${id}/legend`, { method: "PUT", body }, "Erro ao salvar a legenda.");
}
