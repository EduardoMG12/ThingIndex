import { Navbar } from "@/components/home/Navbar";
import { Hero } from "@/components/home/Hero";
import { Metrics } from "@/components/home/Metrics";
import { Features } from "@/components/home/Features";
import { HowItWorks } from "@/components/home/HowItWorks";
import { CTA } from "@/components/home/CTA";
import { Footer } from "@/components/home/Footer";

export const metadata = {
    title: "ThingIndex | Document Intelligence Pipeline",
    description: "Automate your legal document processing with an AI-driven async pipeline.",
};

export default function LandingPage() {
    return (
        <main className="min-h-screen bg-background">
            <Navbar />
            <div className="flex flex-col gap-20 pb-20">
                <Hero />
                <Metrics />
                <Features />
                <HowItWorks />
                <CTA />
            </div>
            <Footer />
        </main>
    );
}
