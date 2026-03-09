"use server";

import { redirect } from "next/navigation";

export async function login(formData: FormData) {
  const email = formData.get("email");
  const password = formData.get("password");

  await new Promise((resolve) => setTimeout(resolve, 1000));

  console.log("Login attempt:", { email, password });

  redirect("/dashboard");
}
