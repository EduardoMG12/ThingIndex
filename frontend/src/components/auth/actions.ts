"use server";

import { redirect } from "next/navigation";
import { cookies } from "next/headers";
import { createForwardHeaders } from "@/app/api/_utils/locale";
import { fetchApiUpstream } from "@/app/api/_utils/upstream";
import { DEFAULT_LOCALE, LOCALE_COOKIE_NAME, normalizeLocale } from "@/lib/i18n/config";

export async function login(formData: FormData) {
  const cookieStore = await cookies();
  const locale = normalizeLocale(cookieStore.get(LOCALE_COOKIE_NAME)?.value ?? DEFAULT_LOCALE);
  const email = formData.get("email");
  const password = formData.get("password");

  if (!email || !password) {
    return { error: "Email e senha são obrigatórios", type: "error" };
  }

  try {
    const res = await fetchApiUpstream("/auth/login", {
      method: "POST",
      headers: await createForwardHeaders(undefined, { contentType: "application/json" }),
      body: JSON.stringify({ email, password }),
    });

    const result = await res.json().catch(() => ({ message: "Erro ao processar resposta" }));

    if (!res.ok) {
      return { 
        error: result.message || "Credenciais inválidas",
        type: result.type || "error",
        status: res.status,
        errors: result.errors
      };
    }

    const token = result.data?.token;
    
    if (!token) {
      return { error: "Token não recebido do servidor" };
    }

    // Configura o cookie com o token JWT
    cookieStore.set("token", token, {
      httpOnly: true,
      secure: process.env.NODE_ENV === "production",
      sameSite: "lax",
      maxAge: 60 * 60 * 24, // 1 dia
      path: "/",
    });
  } catch (error: unknown) {
    console.error("Login error:", error);
    return { error: "Erro de conexão com o servidor", type: "error" };
  }

  redirect(`/${locale}/workspace`);
}

export async function register(formData: FormData) {
  const cookieStore = await cookies();
  const locale = normalizeLocale(cookieStore.get(LOCALE_COOKIE_NAME)?.value ?? DEFAULT_LOCALE);
  const name = formData.get("name");
  const email = formData.get("email");
  const password = formData.get("password");

  if (!name || !email || !password) {
    return { error: "Todos os campos são obrigatórios", type: "error" };
  }

  try {
    const res = await fetchApiUpstream("/auth/register", {
      method: "POST",
      headers: await createForwardHeaders(undefined, { contentType: "application/json" }),
      body: JSON.stringify({ name, email, password }),
    });

    const result = await res.json().catch(() => ({ message: "Erro ao processar resposta" }));

    if (!res.ok) {
      return { 
        error: result.message || "Erro ao criar conta",
        type: result.type || "error",
        status: res.status,
        errors: result.errors
      };
    }

    const token = result.data?.token;

    if (!token) {
      return { error: "Token não recebido do servidor" };
    }

    // Configura o cookie com o token JWT
    cookieStore.set("token", token, {
      httpOnly: true,
      secure: process.env.NODE_ENV === "production",
      sameSite: "lax",
      maxAge: 60 * 60 * 24, // 1 dia
      path: "/",
    });
  } catch (error: unknown) {
    console.error("Registration error:", error);
    return { error: "Erro de conexão com o servidor", type: "error" };
  }

  redirect(`/${locale}/workspace`);
}

export async function logout() {
  const cookieStore = await cookies();
  const locale = normalizeLocale(cookieStore.get(LOCALE_COOKIE_NAME)?.value ?? DEFAULT_LOCALE);
  cookieStore.delete("token");
  redirect(`/${locale}`);
}
