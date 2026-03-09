import Link from "next/link";
import { MoveLeft } from "lucide-react";
import { Button } from "@/components/ui/Button";

export const metadata = {
    title: "Criar Conta | ThingIndex",
};

export default function RegisterPage() {
    return (
        <div className="relative flex min-h-screen items-center justify-center bg-background bg-dots p-4">
            <div className="w-full max-w-sm rounded-xl border border-border bg-secondary/20 p-8 text-center backdrop-blur-sm">
                <h2 className="text-xl font-bold">Criar Conta</h2>
                <p className="mt-2 text-sm text-muted-foreground">
                    A criação de contas está desativada temporariamente.
                </p>
                <Link href="/login" className="mt-6 inline-block">
                    <Button variant="outline" size="sm" className="gap-2">
                        <MoveLeft className="h-4 w-4" />
                        Voltar para o Login
                    </Button>
                </Link>
            </div>
        </div>
    );
}
