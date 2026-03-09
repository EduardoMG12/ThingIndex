"use server";

import { redirect } from "next/navigation";
import { cookies } from "next/headers";

const API_URL = process.env.API_INTERNAL_URL || "http://tcc-api:8000/api/v1";

export async function login(formData: FormData) {
  const email = formData.get("email");
  const password = formData.get("password");

  if (!email || !password) {
    throw new Error("Email e senha são obrigatórios");
  }

  try {
    const res = await fetch(`${API_URL}/auth/login`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email, password }),
    });

    if (!res.ok) {
      const errorData = await res.json().catch(() => ({ error: "Erro ao fazer login" }));
      throw new Error(errorData.error || "Erro desconhecido");
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
  } catch (error) {
    console.error("Login error:", error);
    // Em um cenário real com form data action, você retornaria um estado de erro.
    // Para simplificar, vamos apenas dar o redirect se o login falhar ou tratar no client.
    throw error;
  }

  redirect("/dashboard");
}

export async function register(formData: FormData) {
  const name = formData.get("name");
  const email = formData.get("email");
  const password = formData.get("password");

  if (!name || !email || !password) {
    throw new Error("Todos os campos são obrigatórios");
  }

  try {
    const res = await fetch(`${API_URL}/auth/register`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name, email, password }),
    });

    if (!res.ok) {
      const errorData = await res.json().catch(() => ({ error: "Erro ao registrar usuário" }));
      throw new Error(errorData.error || "Erro desconhecido");
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
  } catch (error) {
    console.error("Registration error:", error);
    throw error;
  }

  redirect("/dashboard");
}
