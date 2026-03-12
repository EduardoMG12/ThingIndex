const LOGIN_PATH = "/login";

export class UnauthorizedRequestError extends Error {
    constructor(message = "Usuario nao autenticado.") {
        super(message);
        this.name = "UnauthorizedRequestError";
    }
}

function redirectToLogin() {
    if (typeof window === "undefined") {
        return;
    }

    if (window.location.pathname === LOGIN_PATH) {
        return;
    }

    window.location.assign(LOGIN_PATH);
}

export async function fetchWithAuthRedirect(
    input: RequestInfo | URL,
    init?: RequestInit
) {
    const response = await fetch(input, init);

    if (response.status === 401) {
        redirectToLogin();
        throw new UnauthorizedRequestError();
    }

    return response;
}
