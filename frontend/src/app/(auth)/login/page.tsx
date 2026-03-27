import { LoginForm } from "@/components/auth/LoginForm";
import { AnimatedBackground } from "@/components/ui/AnimatedBackground";

export const metadata = {
    title: "Login | ThingIndex",
    description: "Faça login no ThingIndex para acessar a orquestração de documentos e RAG.",
};

export default function LoginPage() {
    return (
        <div className="relative flex min-h-screen items-center justify-center p-4 overflow-hidden">
            <AnimatedBackground />
            <LoginForm />
        </div>
    );
}
