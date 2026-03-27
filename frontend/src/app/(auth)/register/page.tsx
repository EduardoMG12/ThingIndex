import { RegisterForm } from "@/components/auth/RegisterForm";
import { AnimatedBackground } from "@/components/ui/AnimatedBackground";

export const metadata = {
    title: "Criar Conta | ThingIndex",
    description: "Crie sua conta no ThingIndex para organizar documentos e contexto relacional.",
};

export default function RegisterPage() {
    return (
        <div className="relative flex min-h-screen items-center justify-center p-4 overflow-hidden">
            <AnimatedBackground />
            <RegisterForm />
        </div>
    );
}
