"use client";

import { useState } from "react";
import { Send, Loader2, Key, Bot } from "lucide-react";
import { Button } from "@/components/ui/Button";
import { Textarea } from "@/components/ui/Textarea";

const AI_MODELS = [
    { value: "gemini", label: "Google Gemini" },
    // Future providers — just add here:
    // { value: "openai", label: "OpenAI GPT" },
    // { value: "grok", label: "xAI Grok" },
];

type Props = {
    onSubmit: (prompt: string) => void;
    isLoading?: boolean;
    aiModel: string;
    onAiModelChange: (model: string) => void;
    apiToken: string;
    onApiTokenChange: (token: string) => void;
};

export default function PromptPanel({
    onSubmit,
    isLoading,
    aiModel,
    onAiModelChange,
    apiToken,
    onApiTokenChange,
}: Props) {
    const [prompt, setPrompt] = useState("");

    const handleSubmit = () => {
        if (!prompt.trim() || isLoading || !apiToken.trim()) return;
        onSubmit(prompt.trim());
        setPrompt("");
    };

    return (
        <div className="surface-card rounded-lg p-5">
            <div className="mb-4 flex items-center gap-2">
                <Send className="h-4 w-4 text-primary" aria-hidden="true" />
                <h3 className="text-sm font-semibold" id="prompt-heading">Assistente Jurídico</h3>
            </div>

            {/* AI Model Selector */}
            <div className="mb-3 flex gap-3">
                <div className="flex-1">
                    <label className="mb-1 block text-[11px] font-medium text-muted-foreground">
                        <Bot className="mr-1 inline h-3 w-3" />
                        Modelo de IA
                    </label>
                    <select
                        value={aiModel}
                        onChange={(e) => onAiModelChange(e.target.value)}
                        className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-1 focus:ring-primary"
                        aria-label="Selecionar modelo de IA"
                    >
                        {AI_MODELS.map((m) => (
                            <option key={m.value} value={m.value}>
                                {m.label}
                            </option>
                        ))}
                    </select>
                </div>
                <div className="flex-1">
                    <label className="mb-1 block text-[11px] font-medium text-muted-foreground">
                        <Key className="mr-1 inline h-3 w-3" />
                        API Token
                    </label>
                    <input
                        type="password"
                        value={apiToken}
                        onChange={(e) => onApiTokenChange(e.target.value)}
                        placeholder="Cole seu token aqui"
                        className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-primary"
                        aria-label="API Token para o modelo de IA"
                    />
                </div>
            </div>

            <Textarea
                placeholder="Escreva sua pergunta jurídica... Ex: Qual a pena para lesão corporal no Código Penal?"
                value={prompt}
                onChange={(e) => setPrompt(e.target.value)}
                className="min-h-[120px] resize-none bg-background text-sm"
                aria-label="Pergunta para o assistente jurídico"
                onKeyDown={(e) => {
                    if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
                        e.preventDefault();
                        handleSubmit();
                    }
                }}
            />

            <div className="mt-3 flex items-center justify-between">
                <span className="text-[11px] text-muted-foreground">
                    {!apiToken.trim()
                        ? "⚠️ Insira seu API Token para enviar"
                        : "⌘/Ctrl + Enter para enviar"}
                </span>
                <Button
                    onClick={handleSubmit}
                    disabled={isLoading || !prompt.trim() || !apiToken.trim()}
                    className="gap-2"
                >
                    {isLoading ? (
                        <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
                    ) : (
                        <Send className="h-4 w-4" aria-hidden="true" />
                    )}
                    Enviar
                </Button>
            </div>
        </div>
    );
}
