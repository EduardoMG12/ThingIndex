"use client";

import * as React from "react";
import { Plus, Trash2, Link } from "lucide-react";
import { Card, CardHeader, CardContent } from "@/components/ui/Card";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";

export interface UrlEntry {
    id: string;
    url: string;
    depth: number;
}

interface UrlPanelProps {
    urls: UrlEntry[];
    onAdd: (url: string, depth: number) => void;
    onRemove: (id: string) => void;
}

export default function UrlPanel({ urls, onAdd, onRemove }: UrlPanelProps) {
    const [url, setUrl] = React.useState("");
    const [depth, setDepth] = React.useState(1);

    const handleAdd = () => {
        if (!url) return;
        onAdd(url, depth);
        setUrl("");
    };

    return (
        <Card>
            <CardHeader>
                <h3 className="text-lg font-semibold flex items-center gap-2">
                    <Link className="w-5 h-5 text-primary" />
                    Fontes de Dados
                </h3>
            </CardHeader>
            <CardContent className="space-y-4">
                <div className="flex gap-2">
                    <Input
                        placeholder="URL (ex: https://dominio.legal/...)"
                        value={url}
                        onChange={(e) => setUrl(e.target.value)}
                    />
                    <div className="w-24">
                        <Input
                            type="number"
                            min={1}
                            max={5}
                            value={depth}
                            onChange={(e) => setDepth(parseInt(e.target.value))}
                        />
                    </div>
                    <Button onClick={handleAdd}>
                        <Plus className="w-5 h-5" />
                    </Button>
                </div>

                <div className="space-y-2">
                    {urls.length === 0 ? (
                        <p className="text-sm text-muted-foreground text-center py-4">
                            Nenhuma URL adicionada ainda.
                        </p>
                    ) : (
                        urls.map((entry) => (
                            <div
                                key={entry.id}
                                className="flex items-center justify-between p-3 rounded-md bg-secondary/20 border border-border"
                            >
                                <div className="overflow-hidden">
                                    <p className="text-sm font-medium truncate">{entry.url}</p>
                                    <p className="text-xs text-muted-foreground">Profundidade: {entry.depth}</p>
                                </div>
                                <Button
                                    variant="ghost"
                                    size="sm"
                                    className="text-destructive hover:text-destructive hover:bg-destructive/10"
                                    onClick={() => onRemove(entry.id)}
                                >
                                    <Trash2 className="w-4 h-4" />
                                </Button>
                            </div>
                        ))
                    )}
                </div>
            </CardContent>
        </Card>
    );
}
