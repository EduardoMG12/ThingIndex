import * as React from "react";
import { Search } from "lucide-react";

export default function Header() {
    return (
        <header className="border-b border-border bg-card/50 backdrop-blur-md sticky top-0 z-50">
            <div className="container h-16 flex items-center justify-between">
                <div className="flex items-center gap-2">
                    <div className="w-8 h-8 rounded-lg bg-primary shadow-gold flex items-center justify-center">
                        <Search className="w-5 h-5 text-primary-foreground" />
                    </div>
                    <span className="text-xl font-bold tracking-tight">
                        Legis<span className="text-gradient-gold">Graph</span>
                    </span>
                </div>

                <nav className="hidden md:flex items-center gap-6">
                    <a href="#" className="text-sm font-medium text-muted-foreground hover:text-foreground transition-colors">Workspace</a>
                    <a href="#" className="text-sm font-medium text-muted-foreground hover:text-foreground transition-colors">Documentation</a>
                    <a href="#" className="text-sm font-medium text-muted-foreground hover:text-foreground transition-colors">Settings</a>
                </nav>

                <div className="flex items-center gap-4">
                    <div className="h-8 w-8 rounded-full bg-secondary border border-border" />
                </div>
            </div>
        </header>
    );
}
