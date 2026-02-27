import Header from "@/components/Header";
import DocumentationContent from "@/components/documentacao/DocumentationContent";
import { Metadata } from "next";

// Metadados estáticos e SEO (Server-side)
export const metadata: Metadata = {
    title: "Documentação | LegisGraph",
    description: "Estrutura do projeto, arquitetura e documentação técnica.",
};

export default function DocumentationPage() {
    return (
        <div className="min-h-screen bg-background">
            <Header />
            <main id="main-content">
                <DocumentationContent />
            </main>
        </div>
    );
}
