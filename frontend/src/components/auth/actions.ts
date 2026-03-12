"use server";

import { redirect } from "next/navigation";
import { cookies } from "next/headers";

const API_URL = process.env.API_INTERNAL_URL || "http://api:8080/api/v1";

export async function login(formData: FormData) {
  const email = formData.get("email");
  const password = formData.get("password");

  if (!email || !password) {
    return { error: "Email e senha são obrigatórios", type: "error" };
  }

  try {
    const res = await fetch(`${API_URL}/auth/login`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
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
    (await cookies()).set("token", token, {
      httpOnly: true,
      secure: process.env.NODE_ENV === "production",
      sameSite: "lax",
      maxAge: 60 * 60 * 24, // 1 dia
      path: "/",
    });
  } catch (error: any) {
    console.error("Login error:", error);
    return { error: "Erro de conexão com o servidor", type: "error" };
  }

  redirect("/dashboard");
}

export async function register(formData: FormData) {
  const name = formData.get("name");
  const email = formData.get("email");
  const password = formData.get("password");

  if (!name || !email || !password) {
    return { error: "Todos os campos são obrigatórios", type: "error" };
  }

  try {
    const res = await fetch(`${API_URL}/auth/register`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
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
    (await cookies()).set("token", token, {
      httpOnly: true,
      secure: process.env.NODE_ENV === "production",
      sameSite: "lax",
      maxAge: 60 * 60 * 24, // 1 dia
      path: "/",
    });
  } catch (error: any) {
    console.error("Registration error:", error);
    return { error: "Erro de conexão com o servidor", type: "error" };
  }

  redirect("/dashboard");
}

export async function logout() {
  (await cookies()).delete("token");
  redirect("/");
}
