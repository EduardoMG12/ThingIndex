"use client";

import { useState } from "react";
import { Send, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/Button";
import { Textarea } from "@/components/ui/Textarea";

type Props = {
    onSubmit: (prompt: string) => void;
    isLoading?: boolean;
};

export default function PromptPanel({ onSubmit, isLoading }: Props) {
    const [prompt, setPrompt] = useState("");

    const handleSubmit = () => {
        if (!prompt.trim() || isLoading) return;
        onSubmit(prompt.trim());
        setPrompt("");
    };

    return (
        <div className="surface-card rounded-lg p-5">
            <div className="mb-4 flex items-center gap-2">
                <Send className="h-4 w-4 text-primary" />
                <h3 className="text-sm font-semibold">Prompt</h3>
            </div>

            <Textarea
                placeholder="Escreva seu prompt aqui... Ex: Analise os artigos sobre responsabilidade civil no CDC"
                value={prompt}
                onChange={(e) => setPrompt(e.target.value)}
                className="min-h-[120px] resize-none bg-background text-sm"
                onKeyDown={(e) => {
                    if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
                        e.preventDefault();
                        handleSubmit();
                    }
                }}
            />

            <div className="mt-3 flex items-center justify-between">
                <span className="text-[11px] text-muted-foreground">⌘/Ctrl + Enter para enviar</span>
                <Button onClick={handleSubmit} disabled={isLoading || !prompt.trim()} className="gap-2">
                    {isLoading ? (
                        <Loader2 className="h-4 w-4 animate-spin" />
                    ) : (
                        <Send className="h-4 w-4" />
                    )}
                    Enviar
                </Button>
            </div>
        </div>
    );
}
