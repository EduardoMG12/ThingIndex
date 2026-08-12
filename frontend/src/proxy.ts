import { NextResponse, type NextRequest } from "next/server";

import {
    getLocaleFromPathname,
    getPreferredLocale,
    localizePath,
    LOCALE_COOKIE_NAME,
    stripLocaleFromPathname,
} from "@/lib/i18n/config";

const PROTECTED_PATHS = ["/dashboard", "/documents", "/cost-visibility", "/settings"];
const AUTH_PATHS = ["/login", "/register"];

function isStaticAsset(pathname: string) {
    return pathname.startsWith("/_next") || pathname.startsWith("/api") || /\.[a-z0-9]+$/i.test(pathname);
}

export function proxy(request: NextRequest) {
    const pathname = request.nextUrl.pathname;

    if (isStaticAsset(pathname)) {
        return NextResponse.next();
    }

    const token = request.cookies.get("token")?.value;
    const pathnameLocale = getLocaleFromPathname(pathname);
    const preferredLocale =
        pathnameLocale ??
        getPreferredLocale(
            request.cookies.get(LOCALE_COOKIE_NAME)?.value ?? request.headers.get("accept-language")
        );
    const normalizedPath = pathnameLocale ? stripLocaleFromPathname(pathname) : pathname;

    if (!pathnameLocale) {
        const redirectUrl = request.nextUrl.clone();
        redirectUrl.pathname = localizePath(pathname, preferredLocale);
        const redirectResponse = NextResponse.redirect(redirectUrl);
        redirectResponse.cookies.set(LOCALE_COOKIE_NAME, preferredLocale, {
            path: "/",
            sameSite: "lax",
            maxAge: 60 * 60 * 24 * 365,
        });
        return redirectResponse;
    }

    const isAuthPage = AUTH_PATHS.some((route) => normalizedPath === route || normalizedPath.startsWith(`${route}/`));
    const isProtectedPage = PROTECTED_PATHS.some(
        (route) => normalizedPath === route || normalizedPath.startsWith(`${route}/`)
    );

    if (!token && !isAuthPage && isProtectedPage) {
        const redirectUrl = request.nextUrl.clone();
        redirectUrl.pathname = `/${preferredLocale}/login`;
        redirectUrl.searchParams.set("next", pathname);
        const response = NextResponse.redirect(redirectUrl);
        response.cookies.set(LOCALE_COOKIE_NAME, preferredLocale, {
            path: "/",
            sameSite: "lax",
            maxAge: 60 * 60 * 24 * 365,
        });
        return response;
    }

    if (token && isAuthPage) {
        const redirectUrl = request.nextUrl.clone();
        redirectUrl.pathname = `/${preferredLocale}/dashboard`;
        const response = NextResponse.redirect(redirectUrl);
        response.cookies.set(LOCALE_COOKIE_NAME, preferredLocale, {
            path: "/",
            sameSite: "lax",
            maxAge: 60 * 60 * 24 * 365,
        });
        return response;
    }

    const response = NextResponse.next();
    response.cookies.set(LOCALE_COOKIE_NAME, preferredLocale, {
        path: "/",
        sameSite: "lax",
        maxAge: 60 * 60 * 24 * 365,
    });
    return response;
}

export const config = {
    matcher: ["/((?!_next|api|favicon.ico|robots.txt|sitemap.xml).*)"],
};
