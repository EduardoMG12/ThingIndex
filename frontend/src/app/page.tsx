import { Navbar } from "@/components/home/Navbar";
import { Hero } from "@/components/home/Hero";
import { Features } from "@/components/home/Features";
import { HowItWorks } from "@/components/home/HowItWorks";
import { CTA } from "@/components/home/CTA";
import { Footer } from "@/components/home/Footer";
import { ProductFocusStrip } from "@/components/home/ProductFocusStrip";

export const metadata = {
    title: "ThingIndex | Relational Document Intelligence",
    description: "Organize, index, and recover document context with a relational architecture.",
};

export default function LandingPage() {
    return (
        <main className="min-h-screen bg-background">
            <Navbar />
            <div className="flex flex-col gap-20 pb-20">
                <Hero />
                <ProductFocusStrip />
                <Features />
                <HowItWorks />
                <CTA />
            </div>
            <Footer />
        </main>
    );
}
