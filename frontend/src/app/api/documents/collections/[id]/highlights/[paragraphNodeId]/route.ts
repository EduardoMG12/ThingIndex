import { proxyCollectionJson } from "@/app/api/documents/collections/[id]/_proxy";

type RouteContext = { params: Promise<{ id: string; paragraphNodeId: string }> };

export async function PUT(request: Request, context: RouteContext) {
    const { id, paragraphNodeId } = await context.params;
    const body = await request.json().catch(() => ({}));
    return proxyCollectionJson(
        request,
        `${id}/highlights/${paragraphNodeId}`,
        { method: "PUT", body },
        "Erro ao salvar a marca."
    );
}

export async function DELETE(request: Request, context: RouteContext) {
    const { id, paragraphNodeId } = await context.params;
    return proxyCollectionJson(request, `${id}/highlights/${paragraphNodeId}`, { method: "DELETE" }, "Erro ao remover a marca.");
}
