# Auditoria do frontend ThingIndex (Next.js 16 / React 19 / App Router)

## Estado das verificações automáticas

| Comando | Resultado |
|---|---|
| `npm run lint` | **exit 0** — 0 erros, **3 warnings** (detalhados em P1-6 e P1-7) |
| `npm run typecheck` | **exit 0** — nenhum erro |
| `npm run test:unit` | **50 arquivos / 134 testes, todos passando** (15,3s) |
| `npm run build` | **exit 0** — 61 páginas estáticas geradas |

Não há falha de suíte para reportar. Os problemas abaixo são de comportamento em runtime, acessibilidade, performance e i18n — coisas que lint/tsc/vitest não pegam. Foram validados lendo o build de produção (`.next/server/app/**/*.html`, `.next/static/chunks`) e calculando contraste WCAG a partir dos tokens reais de `globals.css`.

---

## P0 — quebra experiência real em produção

### 1. `<html lang>` é sempre `"en"`, inclusive nas páginas pt-BR e es
**Arquivo:** `src/app/layout.tsx:56`

```tsx
<html lang="en" suppressHydrationWarning>
```

**Evidência (build de produção, não inferência):**
```
$ grep -o 'lang="[^"]*"' .next/server/app/pt-BR/documentation.html | head -1
lang="en"
$ grep -o 'lang="[^"]*"' .next/server/app/es/login.html | head -1
lang="en"
$ grep -o "Contraseña" .next/server/app/es/login.html | head -1
Contraseña          # conteúdo em espanhol, atributo diz inglês
```
Nenhum código escreve `documentElement.lang` — só lê (`src/lib/i18n/translate.ts:72`, `src/lib/api/fetch-with-auth-redirect.ts:16`). O valor nunca é corrigido no cliente.

**Impacto no usuário:** leitores de tela pronunciam texto português/espanhol com fonética inglesa ("Contraseña", "Não foi possível") — falha WCAG 3.1.1 (Nível A). Google indexa 61 páginas SSG como inglês, quebrando o `hreflang` que `src/lib/seo.ts` monta corretamente. Corretor ortográfico e tradução automática do navegador também erram. Efeito colateral extra: `fetch-with-auth-redirect.ts:16` usa `documentElement.lang` como fallback de locale, então requisições podem sair com `Accept-Language: en` num usuário pt-BR.

**Correção:** transformar o root layout num Server Component que lê o locale (via `params` no segmento `[locale]` ou o cookie `thingindex-locale` que o proxy já grava em `src/proxy.ts:38`) e emitir `<html lang={locale}>`.

---

### 2. Splash screen bloqueia a tela por 3,5 s a cada carregamento — a supressão foi comentada
**Arquivo:** `src/components/ui/SplashScreen.tsx:99`

```tsx
const timer = setTimeout(() => {
    // localStorage.setItem(SPLASH_STORAGE_KEY, Date.now().toString());   // <-- comentado
    setIsDismissed(true);
}, 3500);
```
`shouldShowSplash()` (linha 42) lê exatamente essa chave e retorna `true` quando ela não existe. Como nada nunca a escreve (`grep -rn "setItem" src` retorna só esta linha comentada), a condição `!lastShow` é sempre verdadeira. O intervalo de 3 dias (`SHOW_INTERVAL_MS`, linha 8) está morto.

O overlay é `className="fixed inset-0 z-[9999] ..."` (linha 114) e **não** tem `pointer-events-none` — só o `<div>` interno das partículas tem (linha 123). Ou seja, captura todos os cliques.

**Impacto no usuário:** todo hard refresh / primeira visita / aba nova custa 3,5 s de tela cheia inclicável. Em 3 idiomas e em todas as rotas. A própria suíte E2E precisa contornar isso (`tests/e2e/public-smoke.spec.ts:7` injeta a chave no `localStorage` antes de cada teste) — o teste documenta o bug em vez de pegá-lo.

**Correção:** descomentar o `setItem` (e considerar `sessionStorage` ou reduzir para ~1200 ms). Se a intenção era desabilitar o splash, remover o componente do `layout.tsx:62`.

---

### 3. Tema claro reprova em contraste WCAG AA no token primário
**Arquivo:** `src/app/globals.css:45-46, 54`

Razões calculadas a partir dos tokens HSL reais:

| Combinação | Onde | Razão | Exigido |
|---|---|---|---|
| `--primary: 38 92% 48%` sobre `--background: 42 45% 96%` (`text-primary`) | 151 ocorrências de `text-primary` em `.tsx` | **2.16:1** | 4.5:1 |
| `--primary-foreground: 0 0% 100%` sobre `bg-primary` | `Button variant="primary"` (`src/components/ui/button.tsx:16`), `badge.tsx:12`, skip-link (`layout.tsx:80`) | **2.32:1** | 4.5:1 |
| `--accent: 32 82% 52%` sobre background | `text-accent` | **2.39:1** | 4.5:1 |

Para comparação, o mesmo par no tema escuro dá **9.89:1** — o problema é exclusivo do `:root[data-theme="light"]`. O tema claro é alcançável: `ThemeToggle` está montado no header do `AppShell` (`src/components/layout/AppShell.tsx:612`), e o script inline de `layout.tsx:8-14` já aplica `light` automaticamente para quem tem `prefers-color-scheme: light`.

**Impacto no usuário:** no tema claro, **todo botão primário** (Entrar, Criar conta, Processar upload, Enviar mensagem) tem rótulo branco sobre laranja quase ilegível, além de todos os eyebrows/ícones `text-primary`. Usuários com baixa visão ou tela sob luz forte não conseguem ler as ações principais. O próprio skip-link de acessibilidade herda o problema.

**Correção:** escurecer `--primary` no tema claro para ~`38 92% 34%` (≈4.6:1 sobre o background) e usar `--primary-foreground` escuro (ex.: `220 24% 14%`) sobre o laranja em vez de branco. Adicionar um teste de contraste no CI (o `@storybook/addon-a11y` já está instalado).

---

### 4. Escolha de tema não é persistida — o toggle se desfaz sozinho no reload
**Arquivo:** `src/components/ui/ThemeToggle.tsx:37-41`

```tsx
function applyTheme(theme: ThemeMode) {
    document.documentElement.dataset.theme = theme;
    document.documentElement.style.colorScheme = theme;
    window.dispatchEvent(new Event(THEME_EVENT));
}   // nenhum localStorage / cookie
```

E o script de boot (`src/app/layout.tsx:9-14`) sempre recalcula a partir do sistema, ignorando qualquer preferência anterior:
```js
const preferredTheme = window.matchMedia("(prefers-color-scheme: light)").matches ? "light" : "dark";
document.documentElement.dataset.theme = preferredTheme;
```
`grep -rn "setItem|document.cookie *=" src` (excluindo testes) retorna **zero** escritas de tema.

**Impacto no usuário:** o botão de tema é decorativo entre navegações com recarga. Quem está no sistema em modo claro e escolhe escuro (justamente para escapar do problema P0-3) volta ao claro em cada F5. Como um dos temas reprova em contraste, isso força repetidamente o usuário para o modo inacessível.

**Correção:** gravar em `localStorage` (ou no mesmo cookie que o proxy já usa) dentro de `applyTheme`, e no script inline priorizar o valor salvo antes de cair no `matchMedia`.

---

## P1 — degradação séria de performance, corretude ou qualidade

### 5. Os três dicionários completos vão para o cliente em 37 das 38 páginas
**Arquivos:** `src/lib/i18n/messages.ts` (3.666 linhas / **208 KB** de fonte), `src/lib/i18n/provider.tsx:8`

`messages.ts` exporta um único objeto com `pt-BR`, `en` e `es` juntos, e `provider.tsx` (um `"use client"`) importa `getMessages` — o que arrasta o módulo inteiro para o bundle de cliente. Confirmado no build:

```
$ for f in .next/static/chunks/*.js; do grep -q "No se pudieron cargar los modelos" $f && echo $f; done
.next/static/chunks/2c96c818df619b57.js    # 144 KB minificado, 44,7 KB gzip

$ n=0; hit=0; for f in $(find .next/server/app -name "*.html"); do n=$((n+1)); grep -q "2c96c818df619b57" $f && hit=$((hit+1)); done; echo "$hit / $n"
37 / 38
```
O chunk carrega strings em espanhol mesmo na página `/en/login`. Total de first-load JS bruto de `/es/login`: 915 KB.

**Impacto no usuário:** ~30 KB gzip desperdiçados em toda navegação — para um idioma (`es`) que a UI nem oferece, já que `LanguageSwitcher` o filtra explicitamente (`src/components/ui/LanguageSwitcher.tsx:14-15`). Penaliza LCP/TTI em 3G e mobile, e cresce a cada chave nova.

**Correção:** quebrar `messages.ts` em `messages.pt-BR.ts` / `messages.en.ts` / `messages.es.ts` e carregar só o ativo (`import()` dinâmico no provider, ou passar as mensagens já resolvidas do Server Component `[locale]/layout.tsx` como prop). Ganho imediato: ~2/3 do chunk.

---

### 6. `useDocuments`: efeito com dependências ausentes + closure obsoleta no polling
**Arquivo:** `src/hooks/use-documents.ts:100-109` (+ `62`)

Saída literal do lint:
```
/src/hooks/use-documents.ts
   62:22  warning  'err' is defined but never used
  109:8   warning  React Hook useEffect has missing dependencies: 'loadDocuments' and 'stopPolling'
```
`loadDocuments` é declarada como `async function` solta (linha 73) — recriada a cada render e nunca memoizada — e o efeito de montagem tem `[]`. Ela é exportada como `reloadDocuments` (linha 117) e chamada de fora em `src/app/documents/page.tsx:1743` (`refreshAfterMutation`) e `:1723`.

Pior, o `catch` do poller (linha 62-67) descarta `err` e **reagenda incondicionalmente** a cada `POLL_INTERVAL_MS * 2`:
```ts
} catch (err) {
    if (controller.signal.aborted) { return; }
    pollTimerRef.current = setTimeout(poll, POLL_INTERVAL_MS * 2);
}
```
Não há contador de tentativas, backoff crescente nem `setError`.

**Impacto no usuário:** com a API fora do ar ou o token expirado, a página de documentos fica batendo em `/api/documents` a cada 6 s indefinidamente, em silêncio — sem mensagem de erro na tela, drenando bateria e enchendo o log do servidor. E como o erro é engolido, o estado `error` do hook nunca reflete a falha durante o polling.

**Correção:** envolver `loadDocuments` em `useCallback`, declarar as deps corretas e adicionar backoff exponencial com teto (ex.: 5 tentativas) que faça `setError` e pare.

---

### 7. `use-chat-workspace`: sincronização de URL lê `searchParams` fora das dependências
**Arquivo:** `src/hooks/use-chat-workspace.ts:717-736`

Lint:
```
736:8  warning  React Hook useEffect has a missing dependency: 'searchParams'
```
O corpo compara contra o valor atual da query e só então navega:
```ts
const next = params.toString();
const current = searchParams.toString();   // lido, mas não é dependência
if (next !== current) {
    router.replace(next ? `${pathname}?${next}` : pathname, { scroll: false });
}
}, [selectedCollectionId, activeConversationId, pathname, router]);
```

**Impacto no usuário:** quando a URL muda por fora deste efeito (o próprio `AppShell` faz `router.push` com `?new=<timestamp>` em `AppShell.tsx:265` e `:283`), o `searchParams` capturado fica velho e a comparação `next !== current` pode dar falso-negativo — a URL deixa de refletir a conversa aberta, e um refresh/compartilhamento de link abre a conversa errada. Também é a origem plausível de `router.replace` redundante em loop com os efeitos vizinhos (`:375-388`, `:390-398`) que leem o mesmo `searchParams`.

**Correção:** incluir `searchParams` nas deps (e derivar o `current` de um `useRef` se isso introduzir re-execução excessiva), ou usar `window.location.search` na hora da comparação.

---

### 8. `DocumentsPage` (2.243 linhas) recalcula todo o pipeline de dados a cada render — zero `useMemo`
**Arquivo:** `src/app/documents/page.tsx:1571-1578`

```tsx
const operationalNow = new Date();                                        // 1571 — novo objeto todo render
const operationalOverview = createDocumentOperationalOverview(documents, operationalNow);
const scopedDocuments = ... filterDocumentsByOperationalView(documents, operationalFilter, operationalSearch, operationalNow);
const groupedDocuments = groupDocuments(scopedDocuments, locale, t("documentsPage.collectionFallback"));
const categoriesCount = new Set(documents.map(...)).size;
const collectionsCount = new Set(documents.map(...)).size;
```

```
$ grep -c "useMemo" src/app/documents/page.tsx
0
$ sed -n '4p' src/app/documents/page.tsx
import { useEffect, useState } from "react";      # useMemo nem é importado
```
`groupDocuments` (linha 690-750) constrói `Map` aninhados, faz spread de arrays e **ordena com `localeCompare` em três níveis**; `createDocumentOperationalOverview` percorre a lista inteira (`src/lib/document-observability.ts:173`, 384 linhas).

O componente tem 15 `useState` (linhas 1556-1570) e o `useDocuments` faz polling a cada 3 s (`use-documents.ts:7`), então cada tick e **cada tecla digitada** no campo de busca (`operationalSearch`, sem debounce) refaz todo esse pipeline.

**Impacto no usuário:** digitação travada no filtro de documentos e stutter visível no polling quando a biblioteca cresce. Como `operationalNow = new Date()` muda a cada render, nem uma memoização ingênua funcionaria — o timestamp precisa ser estabilizado antes.

**Correção:** estabilizar `operationalNow` (`useRef`/`useState` renovado por intervalo, não por render), envolver `operationalOverview`/`scopedDocuments`/`groupedDocuments`/os dois `Set` em `useMemo` com deps explícitas, e aplicar debounce (~200 ms) em `operationalSearch`. Extrair as subárvores de lista para componentes com `React.memo` (hoje `grep "React.memo\|memo("` em todo `src` retorna zero).

---

### 9. Mais código morto: dois componentes e um módulo i18n sem nenhum import
Mapeei os 239 módulos de `src/`, resolvi todo `import`/`import()` (incluindo o alias `@/`) e contei referências. Além do `ChatConversationRail`/`ChatWorkspaceHero` já conhecidos:

| Arquivo | Linhas | Referências |
|---|---|---|
| `src/components/documentation/PricingTable.tsx` | 73 | **0** (nem teste, nem story) |
| `src/components/documentation/ComparisonTable.tsx` | 28 | **0** |
| `src/lib/i18n/server.ts` | 28 | **0** — exporta `getServerLocale`, `getServerMessages`, `resolveLocaleFromHeaders`, nenhuma usada |

```
$ grep -rn "PricingTable\|ComparisonTable" src | grep -v "^src/components/documentation/PricingTable.tsx\|^src/components/documentation/ComparisonTable.tsx"
(vazio)
```
`src/test/test-utils.tsx` também só é referenciado por testes, mas isso é correto por design.

**Impacto no usuário:** nenhum direto — mas `server.ts` é justamente a peça que resolveria o P0-1 (locale no servidor) e está abandonada, o que sugere uma migração pela metade. Manutenção paga o custo de ler/atualizar código que nada renderiza.

**Correção:** deletar `PricingTable.tsx` e `ComparisonTable.tsx`; **antes de deletar** `i18n/server.ts`, avaliar reaproveitá-lo para corrigir o `<html lang>`.

---

### 10. 14 chaves de i18n órfãs (custo de 42 entradas mantidas nos 3 idiomas)
**Arquivo:** `src/lib/i18n/messages.ts`

**Primeiro, a boa notícia:** achatei as três árvores e comparei — **nenhuma chave faltando em nenhum idioma**. `pt-BR: 873 / en: 873 / es: 873`, união = 873, zero divergências. A UI não vai cair em fallback de chave crua.

Órfãs confirmadas (busquei cada uma como literal e como prefixo dinâmico em todo `src` + `tests`, excluindo o próprio dicionário):

```
common.appName          common.loading          common.language      common.skipToContent
shell.user.role         auth.login.pageDescription                   auth.register.pageDescription
landing.metrics.precision  landing.metrics.latency  landing.metrics.throughput  landing.metrics.uptime
chatPage.panelTitle     chatPage.panelDescription  chatPage.collectionsTitle
```

As de `seo.pages.*` e `providerRuntime.{chat,indexing}.*` **não** são órfãs — são montadas dinamicamente (`src/lib/seo.ts:20` e `src/components/runtime/ProviderRuntimeCallout.tsx:15`) e por isso ficaram fora da lista.

Dois casos merecem atenção porque revelam funcionalidade perdida, não só lixo:
- **`common.skipToContent`** existe traduzido nos 3 idiomas, mas o skip-link renderiza o texto fixo `"Skip to main content"` (`src/app/layout.tsx:82`) — o único atalho de teclado para pular a navegação fica em inglês para todos.
- **`auth.login.pageDescription` / `auth.register.pageDescription`** existem traduzidas, mas `src/app/(auth)/login/page.tsx:4-7` e `register/page.tsx:4-7` exportam `metadata` com descrição hardcoded em português.

Oito valores também são idênticos nos 3 idiomas — mas todos são legitimamente neutros (`"Dashboard | ThingIndex"`, `"{tokens} tokens · {cost}"`); não é tradução faltando.

**Correção:** usar `common.skipToContent` e os `pageDescription` (fecha meio caminho do P0-1); remover as 11 restantes × 3 idiomas. Adicionar um teste de unidade que falhe em chave órfã e em divergência entre locales — o parity está perfeito hoje e vale proteger.

---

### 11. Strings pt-BR hardcoded no caminho do cliente vazam para usuários en/es
**Arquivos:** 15 ocorrências fora de `messages.ts` e de testes.

Os casos que o usuário realmente vê:

| Arquivo:linha | String |
|---|---|
| `src/components/ui/NotFoundContent.tsx:56,72` | `"Página não encontrada"`, `"Voltar para o Início"` — **a página 404 inteira** |
| `src/components/auth/actions.ts:16,53,67,104` | `"Email e senha são obrigatórios"`, `"Erro de conexão com o servidor"`, `"Todos os campos são obrigatórios"` |
| `src/lib/api/current-user.ts:59,63,68,93,97,102` | `"Nao foi possivel carregar o usuario autenticado."` etc. |
| `src/hooks/use-documents.ts:91`, `use-collections.ts:34,88`, `use-categories.ts:26`, `use-provider-runtime-overview.ts:35` | fallbacks `"Nao foi possivel carregar ..."` |
| `src/components/documentation/{IndexingBenchmarkSection:113, ProductFocusSection:25, ProviderSupportSection:38}.tsx` | `fallbackError` em pt |

O agravante estrutural: `src/app/not-found.tsx:12` e `src/app/(auth)/layout.tsx:9` fixam `<I18nProvider locale="pt-BR">`, então até o que *é* traduzido nessas rotas sai em português. (Nas rotas `/[locale]/*` o provider se auto-corrige via `getLocaleFromPathname` — `provider.tsx:26` — mas essas duas não passam por lá.)

A infra correta já existe e é usada em 53 pontos: `apiErrorMessage()` (`src/lib/api/error.ts`) resolvendo as 43 chaves `apiErrors.*` — os arquivos acima simplesmente não a adotaram.

**Impacto no usuário:** um usuário en-US que erra a senha vê `"Email e senha são obrigatórios"`; que acessa URL inválida vê uma 404 100% em português; que perde a sessão vê `"Nao foi possivel carregar o usuario autenticado."` (sem acentos, ainda por cima). Quebra a promessa de produto trilíngue exatamente nos momentos de erro.

**Correção:** migrar essas mensagens para `apiErrorMessage()`/`t()`, e trocar os `locale="pt-BR"` fixos de `not-found.tsx` e `(auth)/layout.tsx` pelo locale resolvido.

---

## P2 — melhorias relevantes de robustez e acessibilidade

### 12. Modais sem gestão de foco; painel de preview sem Escape e com botão de fechar sem rótulo
**Arquivos:** `src/components/ui/danger-confirm-modal.tsx`, `src/app/documents/page.tsx:1068`

O `DangerConfirmModal` acerta o básico — `role="dialog"`, `aria-modal`, `aria-labelledby`/`aria-describedby` (linhas 87-91), Escape (48-62) e `<label>` associado ao input (110-117). Mas:
- `grep -n "focus()\|autoFocus\|trap"` no arquivo → **vazio**: nenhum foco inicial, nenhum focus trap, nenhuma restauração de foco ao fechar. Com Tab dá para sair do diálogo e navegar a página "atrás" do overlay.

O painel lateral de preview de documento é pior:
- Botão de fechar **sem `aria-label`** e só com ícone (`documents/page.tsx:1068-1073`):
  ```tsx
  <button type="button" onClick={onClose} className="rounded-lg border ... p-2 ...">
      <X className="h-4 w-4" />
  </button>
  ```
  Um leitor de tela anuncia apenas "botão". (O overlay clicável logo acima, linha 1049-1054, *tem* `aria-label` — a inconsistência é dentro do mesmo componente.)
- `grep -n "Escape\|keydown" src/app/documents/page.tsx` → **exit 1, nenhuma ocorrência**: o painel não fecha com Escape.
- O contêiner é um `<div className="fixed inset-0 z-50 flex">` sem `role="dialog"`/`aria-modal`.

Mesmo padrão em `AppShell.tsx:372` (botão "cancelar edição" só com `<X>`, sem `aria-label` — enquanto os botões irmãos de renomear/excluir, linhas 400 e 410, têm).

**Impacto no usuário:** usuário de teclado que abre o preview de um documento fica preso (não há Escape) e navega às cegas por trás do overlay; usuário de leitor de tela não sabe o que os botões `X` fazem. Falha WCAG 2.4.3 e 4.1.2.

**Correção:** adicionar `aria-label={t("...")}` nos dois botões-ícone; extrair um hook `useModalA11y` (foco inicial no primeiro elemento focável, ciclo de Tab contido, retorno do foco ao gatilho, Escape) e aplicar ao `DangerConfirmModal` e ao painel de preview, que também deve receber `role="dialog" aria-modal="true"`.

---

### 13. Nenhum error boundary em toda a aplicação
**Diretório:** `src/app/`

```
$ find src/app -name "error.tsx" -o -name "global-error.tsx" -o -name "loading.tsx"
(vazio)
$ grep -rn "ErrorBoundary\|componentDidCatch" src
(vazio)
```
Os hooks tratam falha de *fetch* com competência (`AbortController`, `Promise.allSettled` em `use-cost-visibility.ts:57` e `settings/page.tsx:655`, estados de loading/erro/vazio bem cobertos) — o buraco é o erro de *renderização*.

Superfície concreta de risco: `ChatRichText` (`src/components/chat/ChatRichText.tsx`) parseia markdown vindo do LLM em runtime; `IndexStructureView` (`documents/page.tsx:772+`) faz encadeamentos longos sobre payload da API (`documentIndex?.paragraph_nodes ?? context?.paragraphNodes.filter(...)`); `ChatMessageCard.tsx:30` faz `new Intl.DateTimeFormat(locale).format(new Date(message.createdAt))` — uma data inválida vindo da API lança `RangeError` e derruba a árvore.

**Impacto no usuário:** qualquer um desses erros troca a aplicação inteira pela tela de erro padrão do Next, perdendo a conversa em andamento ou a fila de upload, sem caminho de recuperação além do refresh.

**Correção:** adicionar `src/app/error.tsx` (com botão `reset()`) e `src/app/global-error.tsx`; opcionalmente boundaries locais em torno do chat e do painel de índice para degradar só a região afetada.

---

### 14. `sitemap.xml` publica rotas protegidas que só redirecionam para o login
**Arquivo:** `src/app/sitemap.ts:6`

```ts
const localizedPaths = ["/", "/login", "/register", "/dashboard", "/chat", "/documents", "/cost-visibility", "/documentation", "/settings"];
```
Mas o proxy barra quatro delas sem sessão (`src/proxy.ts:11`):
```ts
const PROTECTED_PATHS = ["/dashboard", "/documents", "/cost-visibility", "/settings"];
```
São 4 rotas × 3 locales = **12 das 27 URLs do sitemap** que devolvem 307 para `/login?next=...` para qualquer crawler. `src/app/robots.ts` faz `allow: "/"`, sem exclusão.

**Impacto no usuário:** desperdício de crawl budget e resultados de busca que levam à tela de login em vez de conteúdo. `buildLocalizedMetadata` ainda declara `robots: { index: true, follow: true }` (`src/lib/seo.ts:47-50`) nessas páginas, reforçando o sinal errado.

**Correção:** derivar `localizedPaths` excluindo `PROTECTED_PATHS` (importar a constante do `proxy.ts` para as duas listas não divergirem) e marcar `robots: { index: false }` no metadata das rotas autenticadas.

---

### 15. Erros de renomear/excluir conversa são engolidos; o delete gera unhandled rejection
**Arquivo:** `src/components/layout/AppShell.tsx:239-241` e `:244-277`

```tsx
// saveConversationTitle — catch vazio, sem toast, sem estado de erro
} catch {
    setIsSavingConversationTitle(false);
}

// confirmDeleteConversation — try/finally SEM catch
try {
    await deleteChatConversation(conversationPendingDelete.id);
    ...
} finally {
    setIsDeletingConversation(false);
}
```
A chamada é `onConfirm={() => void confirmDeleteConversation()}` (linha ~578): se `deleteChatConversation` rejeitar, a promise vai para `unhandledrejection` sem tratamento. Contraste direto: o *mesmo* fluxo no `use-chat-workspace.ts:661-667` faz certo (`setStreamError(message)` + rethrow), e o resto do app usa `toast.error(...)` consistentemente (ex.: `AppShell.tsx:194`, `documents/page.tsx:1710`).

**Impacto no usuário:** ao renomear uma conversa com a API fora do ar, a caixa de edição simplesmente para de responder — sem mensagem. Ao excluir, o modal de confirmação fica aberto, o spinner some (por causa do `finally`) e a conversa continua na lista: o usuário conclui que apagou algo que não foi apagado, e a única pista fica no console do navegador.

**Correção:** adicionar `catch (error) { toast.error(error instanceof Error ? error.message : t("apiErrors.chat.delete")) }` nos dois handlers — as chaves `apiErrors.chat.delete` e `apiErrors.chat.update` já existem nos três idiomas.

---

## Resumo

| Prioridade | Itens | Tema dominante |
|---|---|---|
| **P0** | 4 | `lang` incorreto, splash bloqueante, contraste do tema claro, tema não persistido |
| **P1** | 7 | i18n de 3 idiomas no bundle, 2 warnings de deps do lint, ausência total de memoização em página de 2k linhas, código morto, chaves órfãs, strings pt-BR vazando |
| **P2** | 4 | foco/teclado em modais, error boundaries, sitemap, erros silenciosos |

**Pontos genuinamente fortes** (verificados, não presumidos, e por isso ausentes da lista): paridade de chaves i18n perfeita nos 3 idiomas (873/873/873); `AbortController` disciplinado em todos os hooks de fetch; `Promise.allSettled`/`Promise.all` corretamente paralelos em settings, cost-visibility e no painel de índice (nenhum waterfall encontrado); o único `dangerouslySetInnerHTML` é um script de tema estático sem input de usuário; token JWT em cookie `httpOnly` + `sameSite: lax` + `secure` em produção (`src/components/auth/actions.ts:44-49`), nunca em `localStorage`; nenhum dado sensível logado; proxy de conteúdo com allowlist de content-type, `nosniff`, CSP e sandbox no iframe de preview; nenhum `<img>` cru — a única imagem usa `next/image`; e `prefers-reduced-motion` respeitado em `globals.css:105`.
