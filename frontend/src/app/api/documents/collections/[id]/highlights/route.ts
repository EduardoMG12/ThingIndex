import { proxyCollectionJson } from "@/app/api/documents/collections/[id]/_proxy";

type RouteContext = { params: Promise<{ id: string }> };

export async function GET(request: Request, context: RouteContext) {
    const { id } = await context.params;
    return proxyCollectionJson(request, `${id}/highlights`, { method: "GET" }, "Erro ao buscar as marcas da collection.");
}
