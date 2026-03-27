import { NextResponse } from "next/server";

import { getRequestLocale } from "@/app/api/_utils/locale";
import { translate, type TranslationVariables } from "@/lib/i18n/translate";

export async function getApiRouteMessage(
    request: Request | undefined,
    key: string,
    variables?: TranslationVariables
) {
    const locale = await getRequestLocale(request);
    return translate(locale, key, variables);
}

export async function parseProxyJsonWithFallback(response: Response, request: Request, key = "apiRouteErrors.proxyResponse") {
    const fallbackMessage = await getApiRouteMessage(request, key);
    return response.json().catch(() => ({
        message: fallbackMessage,
    }));
}

export async function localizedRouteError(
    request: Request | undefined,
    key: string,
    status: number,
    variables?: TranslationVariables
) {
    return NextResponse.json(
        { message: await getApiRouteMessage(request, key, variables) },
        { status }
    );
}
