"use client";

import * as React from "react";
import { SendHorizontal, Loader2 } from "lucide-react";
import { Card, CardHeader, CardContent } from "@/components/ui/Card";
import { Button } from "@/components/ui/Button";

interface PromptPanelProps {
    onSubmit: (prompt: string) => void;
    isLoading: boolean;
}

export default function PromptPanel({ onSubmit, isLoading }: PromptPanelProps) {
    const [prompt, setPrompt] = React.useState("");

    const handleSubmit = (e: React.FormEvent) => {
        e.preventDefault();
        if (!prompt.trim() || isLoading) return;
        onSubmit(prompt);
        setPrompt("");
    };

    return (
        <Card glow>
            <CardHeader>
                <h3 className="text-lg font-semibold">Análise Legislativa</h3>
            </CardHeader>
            <CardContent>
                <form onSubmit={handleSubmit} className="flex gap-2">
                    <textarea
                        className="flex-1 bg-secondary/30 border border-border rounded-md p-3 text-sm focus:outline-none focus:ring-2 focus:ring-ring resize-none min-h-[80px]"
                        placeholder="Descreva o caso ou solicite uma análise dos documentos..."
                        value={prompt}
                        onChange={(e) => setPrompt(e.target.value)}
                    />
                    <Button
                        type="submit"
                        className="self-end h-10"
                        disabled={!prompt.trim() || isLoading}
                    >
                        {isLoading ? (
                            <Loader2 className="w-5 h-5 animate-spin" />
                        ) : (
                            <SendHorizontal className="w-5 h-5" />
                        )}
                    </Button>
                </form>
            </CardContent>
        </Card>
    );
}
