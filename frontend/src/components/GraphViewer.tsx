"use client";

import * as React from "react";
import { Expand, Share2, Layers } from "lucide-react";
import { Card, CardHeader, CardContent } from "@/components/ui/Card";
import { Button } from "@/components/ui/Button";

interface Node {
    id: string;
    label: string;
    group: number;
}

interface Link {
    source: string;
    target: string;
}

interface GraphViewerProps {
    nodes: Node[];
    links: Link[];
}

export default function GraphViewer({ nodes, links }: GraphViewerProps) {
    return (
        <Card className="h-full flex flex-col relative overflow-hidden group">
            <CardHeader className="absolute top-0 left-0 right-0 z-10 bg-gradient-to-b from-card/80 to-transparent flex flex-row items-center justify-between">
                <h3 className="text-lg font-semibold flex items-center gap-2">
                    <Layers className="w-5 h-5 text-primary" />
                    Visualização de Grafo
                </h3>
                <div className="flex gap-2">
                    <Button variant="secondary" size="sm" className="bg-card/50 backdrop-blur-sm">
                        <Share2 className="w-4 h-4" />
                    </Button>
                    <Button variant="secondary" size="sm" className="bg-card/50 backdrop-blur-sm">
                        <Expand className="w-4 h-4" />
                    </Button>
                </div>
            </CardHeader>

            <CardContent className="flex-1 p-0 flex items-center justify-center bg-[#0a0c10] min-h-[600px] relative">
                {/* Placeholder for actual graph visualization (D3/Sigma/React-Force-Graph) */}
                <div className="absolute inset-0 opacity-20 pointer-events-none overflow-hidden">
                    {/* Simple visual background representation of a graph */}
                    <svg className="w-full h-full" xmlns="http://www.w3.org/2000/svg">
                        <defs>
                            <pattern id="grid" width="40" height="40" patternUnits="userSpaceOnUse">
                                <path d="M 40 0 L 0 0 0 40" fill="none" stroke="currentColor" strokeWidth="0.5" strokeOpacity="0.1" />
                            </pattern>
                        </defs>
                        <rect width="100%" height="100%" fill="url(#grid)" />
                    </svg>
                </div>

                <div className="text-center space-y-4 max-w-xs z-10">
                    <div className="inline-block p-4 rounded-full bg-primary/10 border border-primary/20 animate-pulse">
                        <Layers className="w-12 h-12 text-primary" />
                    </div>
                    <p className="text-sm text-muted-foreground">
                        Renderizando grafo de conhecimento com {nodes.length} nós e {links.length} conexões...
                    </p>
                </div>

                <div className="absolute bottom-6 left-1/2 -translate-x-1/2 flex gap-4 p-2 rounded-full bg-card/50 backdrop-blur-md border border-border">
                    <div className="flex items-center gap-2 px-3">
                        <div className="w-3 h-3 rounded-full bg-primary" />
                        <span className="text-xs">Documentos</span>
                    </div>
                    <div className="flex items-center gap-2 px-3 border-l border-border">
                        <div className="w-3 h-3 rounded-full bg-blue-500" />
                        <span className="text-xs">Artigos</span>
                    </div>
                    <div className="flex items-center gap-2 px-3 border-l border-border">
                        <div className="w-3 h-3 rounded-full bg-green-500" />
                        <span className="text-xs">Tópicos</span>
                    </div>
                </div>
            </CardContent>
        </Card>
    );
}
