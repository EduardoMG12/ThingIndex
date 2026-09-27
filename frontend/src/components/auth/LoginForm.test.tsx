import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

const { loginMock } = vi.hoisted(() => ({
    loginMock: vi.fn(async (formData: FormData) => {
        void formData;
        return undefined;
    }),
}));

vi.mock("@/components/auth/actions", () => ({
    login: loginMock,
}));

import { LoginForm } from "@/components/auth/LoginForm";
import { createTestWrapper } from "@/test/test-utils";

describe("LoginForm repassa o destino ?next= ao server action (F-04)", () => {
    afterEach(() => {
        window.history.replaceState(null, "", "/");
        loginMock.mockClear();
    });

    it("envia o next da URL junto com as credenciais", async () => {
        window.history.replaceState(null, "", "/en/login?next=%2Fen%2Fdocuments");
        render(<LoginForm />, { wrapper: createTestWrapper("en") });

        fireEvent.change(screen.getByLabelText("Email"), { target: { value: "a@b.c" } });
        fireEvent.submit(screen.getByRole("button", { name: /sign in/i }).closest("form")!);

        await waitFor(() => expect(loginMock).toHaveBeenCalledTimes(1));
        const enviado = loginMock.mock.calls[0]![0];
        expect(enviado.get("next")).toBe("/en/documents");
    });

    it("nao envia next quando a URL nao tem o parametro", async () => {
        window.history.replaceState(null, "", "/en/login");
        render(<LoginForm />, { wrapper: createTestWrapper("en") });

        fireEvent.submit(screen.getByRole("button", { name: /sign in/i }).closest("form")!);

        await waitFor(() => expect(loginMock).toHaveBeenCalledTimes(1));
        expect(loginMock.mock.calls[0]![0].get("next")).toBeNull();
    });
});
