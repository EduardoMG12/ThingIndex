import { NotFoundContent } from "@/components/ui/NotFoundContent";

export const metadata = {
    title: "404 - Página Não Encontrada | ThingIndex",
    description: "Oops! A página que você está procurando não existe no ThingIndex.",
};

export default function NotFound() {
    return <NotFoundContent />;
}
