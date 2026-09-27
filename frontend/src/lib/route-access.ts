// Rotas que exigem sessao (o proxy manda para /login) e rotas de auth.
// Fonte unica para o proxy, o sitemap e o robots.
export const PROTECTED_PATHS = ["/workspace", "/documents", "/cost-visibility", "/settings"] as const;
export const AUTH_PATHS = ["/login", "/register"] as const;

// Paginas publicas indexaveis (sem o prefixo de locale).
export const PUBLIC_INDEXABLE_PATHS = ["/", "/login", "/register", "/documentation"] as const;
