import { RegisterForm } from "@/components/auth/RegisterForm";

export const metadata = {
    title: "Criar Conta | ThingIndex",
    description: "Crie sua conta no ThingIndex para configurar seu pipeline de document intelligence.",
};

export default function RegisterPage() {
    return (
        <div className="relative flex min-h-screen items-center justify-center bg-background bg-dots p-4">
            <RegisterForm />
        </div>
    );
}
