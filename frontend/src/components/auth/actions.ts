"use server";

import { redirect } from "next/navigation";
import { cookies } from "next/headers";

const API_URL = process.env.API_INTERNAL_URL || "http://tcc-api:8000/api/v1";

export async function login(formData: FormData) {
  const email = formData.get("email");
  const password = formData.get("password");

  if (!email || !password) {
    return { error: "Email e senha são obrigatórios" };
  }

  try {
    const res = await fetch(`${API_URL}/auth/login`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email, password }),
    });

    if (!res.ok) {
      const errorData = await res.json().catch(() => ({ error: "Erro ao fazer login" }));
      return { error: errorData.error || "Credenciais inválidas" };
    }

    const { token } = await res.json();
    
    // Configura o cookie com o token JWT
    (await cookies()).set("token", token, {
      httpOnly: true,
      secure: process.env.NODE_ENV === "production",
      sameSite: "lax",
      maxAge: 60 * 60 * 24, // 1 dia
      path: "/",
    });
  } catch (error: any) {
    console.error("Login error:", error);
    return { error: "Erro de conexão com o servidor" };
  }

  redirect("/dashboard");
}

export async function register(formData: FormData) {
  const name = formData.get("name");
  const email = formData.get("email");
  const password = formData.get("password");

  if (!name || !email || !password) {
    return { error: "Todos os campos são obrigatórios" };
  }

  try {
    const res = await fetch(`${API_URL}/auth/register`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name, email, password }),
    });

    if (!res.ok) {
      const errorData = await res.json().catch(() => ({ error: "Erro ao registrar usuário" }));
      return { error: errorData.error || "Erro ao criar conta" };
    }

    const { token } = await res.json();

    // Configura o cookie com o token JWT
    (await cookies()).set("token", token, {
      httpOnly: true,
      secure: process.env.NODE_ENV === "production",
      sameSite: "lax",
      maxAge: 60 * 60 * 24, // 1 dia
      path: "/",
    });
  } catch (error: any) {
    console.error("Registration error:", error);
    return { error: "Erro de conexão com o servidor" };
  }

  redirect("/dashboard");
}
