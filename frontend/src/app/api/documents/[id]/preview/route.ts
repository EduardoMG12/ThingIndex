import { proxyDocumentContent, type DocumentContentRouteContext } from "@/app/api/documents/_utils/content-proxy";

export async function GET(request: Request, context: DocumentContentRouteContext) {
    return proxyDocumentContent(request, context, "preview");
}
