import {
    hasResponseData,
    type ApiErrorResponse,
    type ApiSuccessResponse,
} from "@/lib/api/types";
import { fetchWithAuthRedirect } from "@/lib/api/fetch-with-auth-redirect";

type CurrentUserApiRecord = {
    id?: number;
    name?: string;
    email?: string;
    avatar_url?: string;
    createdAt?: string;
    updatedAt?: string;
};

export type CurrentUser = {
    id: number;
    name: string;
    email: string;
    avatarUrl?: string;
    createdAt?: string;
    updatedAt?: string;
};

function normalizeCurrentUser(record: CurrentUserApiRecord): CurrentUser | null {
    if (typeof record.id !== "number" || typeof record.email !== "string") {
        return null;
    }

    return {
        id: record.id,
        name: typeof record.name === "string" ? record.name : "",
        email: record.email,
        avatarUrl: typeof record.avatar_url === "string" ? record.avatar_url : undefined,
        createdAt: typeof record.createdAt === "string" ? record.createdAt : undefined,
        updatedAt: typeof record.updatedAt === "string" ? record.updatedAt : undefined,
    };
}

export async function fetchCurrentUser(signal?: AbortSignal): Promise<CurrentUser> {
    const response = await fetchWithAuthRedirect("/api/user/me", {
        method: "GET",
        cache: "no-store",
        signal,
    });

    const payload = (await response.json().catch(() => null)) as ApiSuccessResponse<CurrentUserApiRecord> | ApiErrorResponse | null;

    if (!response.ok) {
        throw new Error(payload?.message || "Nao foi possivel carregar o usuario autenticado.");
    }

    if (!hasResponseData(payload) || !payload.data) {
        throw new Error("Nao foi possivel carregar o usuario autenticado.");
    }

    const currentUser = normalizeCurrentUser(payload.data);
    if (!currentUser) {
        throw new Error("Resposta invalida para usuario autenticado.");
    }

    return currentUser;
}

export async function updateCurrentUserProfile(input: { name?: string; avatar?: File | null }): Promise<CurrentUser> {
    const formData = new FormData();

    if (typeof input.name === "string") {
        formData.set("name", input.name);
    }

    if (input.avatar) {
        formData.set("avatar", input.avatar);
    }

    const response = await fetchWithAuthRedirect("/api/user/me", {
        method: "PUT",
        body: formData,
    });

    const payload = (await response.json().catch(() => null)) as ApiSuccessResponse<CurrentUserApiRecord> | ApiErrorResponse | null;

    if (!response.ok) {
        throw new Error(payload?.message || "Nao foi possivel atualizar o usuario autenticado.");
    }

    if (!hasResponseData(payload) || !payload.data) {
        throw new Error("Nao foi possivel atualizar o usuario autenticado.");
    }

    const currentUser = normalizeCurrentUser(payload.data);
    if (!currentUser) {
        throw new Error("Resposta invalida para usuario autenticado.");
    }

    return currentUser;
}
