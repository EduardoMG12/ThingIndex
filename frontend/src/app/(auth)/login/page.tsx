import { LoginForm } from "@/components/auth/LoginForm";

export const metadata = {
    title: "Login | ThingIndex",
    description: "Faça login no ThingIndex para acessar a orquestração de documentos e RAG.",
};

export default function LoginPage() {
    return (
        <div className="relative flex min-h-screen items-center justify-center bg-background bg-dots p-4">
            <LoginForm />
        </div>
    );
}
