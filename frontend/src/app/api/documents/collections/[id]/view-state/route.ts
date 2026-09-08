import { proxyCollectionJson } from "@/app/api/documents/collections/[id]/_proxy";

type RouteContext = { params: Promise<{ id: string }> };

export async function GET(request: Request, context: RouteContext) {
    const { id } = await context.params;
    return proxyCollectionJson(request, `${id}/view-state`, { method: "GET" }, "Erro ao buscar o estado de visualizacao.");
}

export async function PUT(request: Request, context: RouteContext) {
    const { id } = await context.params;
    const body = await request.json().catch(() => ({}));
    return proxyCollectionJson(request, `${id}/view-state`, { method: "PUT", body }, "Erro ao salvar o estado de visualizacao.");
}
