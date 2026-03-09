import { Navbar } from "@/components/home/Navbar";
import { Hero } from "@/components/home/Hero";
import { Metrics } from "@/components/home/Metrics";
import { Features } from "@/components/home/Features";
import { HowItWorks } from "@/components/home/HowItWorks";
import { CTA } from "@/components/home/CTA";
import { Footer } from "@/components/home/Footer";

export const metadata = {
    title: "ThingIndex | Document Intelligence Platform",
    description: "Pipeline RAG adaptativo com cadeia de 3 prompts. Indexe, estruture e consulte qualquer documento com precisão superior.",
};

export default function Home() {
    return (
        <div className="min-h-screen bg-background">
            <Navbar />
            <main>
                <Hero />
                <Metrics />
                <Features />
                <HowItWorks />
                <CTA />
            </main>
            <Footer />
        </div>
    );
}
