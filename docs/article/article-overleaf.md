\documentclass[12pt]{article}

\newif\ifthingindexsbc
\IfFileExists{sbc-template.sty}{\thingindexsbctrue}{\thingindexsbcfalse}

\ifthingindexsbc
  \usepackage{sbc-template}
\else
  \usepackage[a4paper,top=3.5cm,bottom=2.5cm,left=3.0cm,right=3.0cm]{geometry}
\fi

\usepackage[utf8]{inputenc}
\usepackage[T1]{fontenc}
\usepackage[brazil,english]{babel}
\usepackage{url}
\usepackage{enumitem}
\usepackage{graphicx}
\usepackage{booktabs}
\usepackage{float}
\usepackage{array}
\usepackage{placeins}

\ifthingindexsbc\else
  \makeatletter
  \newcommand{\inst}[1]{}
  \newcommand{\thingindex@address}{}
  \newcommand{\address}[1]{\gdef\thingindex@address{#1}}
  \newcommand{\email}[1]{\texttt{#1}}
  \renewcommand{\maketitle}{
    \begin{center}
      {\bfseries\LARGE \@title\par}
      \vspace{12pt}
      {\normalsize\bfseries \@author\par}
      \vspace{12pt}
      {\normalsize \thingindex@address\par}
    \end{center}
    \vspace{12pt}
  }
  \newenvironment{resumo}
    {\begin{quote}\noindent\textbf{Resumo. }}
    {\end{quote}}
  \renewenvironment{abstract}
    {\begin{quote}\noindent\textbf{Abstract. }}
    {\end{quote}}
  \makeatother
\fi

\sloppy
\setlist[itemize]{leftmargin=1.5em}

\title{ThingIndex: Plataforma Open Source para Inteligência Documental com Baseline Relacional}

\author{Charles Eduardo Mello Guimarães\inst{1}}

\address{
  Instituto Federal do Paraná (IFPR) -- Campus Palmas\\
  Curso de Sistemas de Informação\\
  Palmas -- PR -- Brasil\\
  \email{charles.eduardo.mello.guimaraes@hotmail.com}
}

\begin{document}

\maketitle

\selectlanguage{brazil}
\begin{resumo}
Este artigo apresenta o ThingIndex, uma plataforma open source voltada a inteligência documental, organização de coleções e recuperação de contexto com base em modelagem relacional. O trabalho parte do problema de que fluxos documentais frequentemente se apoiam em ferramentas fragmentadas, com baixa auditabilidade e pouca rastreabilidade do contexto reconstruído para busca, indexação e interação assistida por IA. O objetivo geral é propor, especificar e avaliar o ThingIndex como uma plataforma open source de inteligência documental baseada em modelagem relacional, capaz de organizar, processar, indexar e recuperar contexto documental de forma auditável. O sistema foi implementado e avaliado em um experimento comparativo com seis braços de recuperação (relacional, BM25, denso exato, denso aproximado e duas fusões por Reciprocal Rank Fusion) sobre corpus controlado com regras anti-circularidade. Os resultados indicam que a recuperação relacional não foi inferior à densa neste corpus (nDCG@10 de 0,869 contra 0,844) e que o índice vetorial aproximado, necessário para escalar a busca densa, torna o ranking instável sob fatores que o usuário não controla: variar apenas a semente ou a ordem de inserção dos documentos alterou o resultado em 100\% das consultas, com perda média de 48,4\% de recall. A recuperação relacional não paga nem o custo de varredura exata nem o preço de não-determinismo do índice aproximado.
\end{resumo}

\noindent\textbf{Palavras-chave}: inteligência documental, recuperação de informação, RAG relacional, baseline relacional, open source.

\selectlanguage{english}
\begin{abstract}
 This paper presents ThingIndex, an open-source platform focused on document intelligence, collection organization, and context retrieval based on relational modeling. The work starts from the problem that document workflows often rely on fragmented tools, with low auditability and limited traceability of the reconstructed context for search, indexing, and AI-assisted interaction. The general objective is to propose, specify, and evaluate ThingIndex as an open-source document intelligence platform based on relational modeling, capable of organizing, processing, indexing, and retrieving document context in an auditable manner. The system was implemented and evaluated in a comparative experiment with six retrieval arms (relational, BM25, exact dense, approximate dense, and two Reciprocal Rank Fusion combinations) over a controlled corpus with anti-circularity rules. The results indicate that relational retrieval was not inferior to dense retrieval on this corpus (nDCG@10 of 0.869 versus 0.844) and that the approximate vector index required to scale dense search makes the ranking unstable under factors the user does not control: varying only the seed or the document insertion order changed the result in 100\% of the queries, with an average recall loss of 48.4\%. Relational retrieval pays neither the cost of exact scanning nor the non-determinism price of the approximate index.
\end{abstract}

\noindent\textbf{Keywords}: document intelligence, information retrieval, relational RAG, relational baseline, open source.

\selectlanguage{brazil}

% =============================================================================
% 1. INTRODUCAO
% =============================================================================
\section{Introdução}

Em ambientes organizacionais, a produção de documentos, como relatórios técnicos, atas de reunião, contratos e manuais de processos, é contínua. Esses documentos são frequentemente armazenados em sistemas de arquivos, diretórios compartilhados ou plataformas de gestão eletrônica de documentos sem qualquer forma de estruturação semântica de seu conteúdo. Quando um colaborador precisa localizar uma informação específica, como uma cláusula contratual, um parecer técnico ou uma decisão registrada em ata, a busca geralmente se limita a palavras-chave associadas ao nome do arquivo ou, em alguns casos, a metadados cadastrais. Estudos sobre recuperação de informação em ambientes corporativos indicam que a dependência de mecanismos baseados em palavras-chave e metadados pode dificultar a localização eficiente de documentos relevantes e restringir o acesso ao contexto associado às informações recuperadas \cite{schymik2015benefits}. Como consequência, a rastreabilidade da informação recuperada, sua origem documental e a preservação de seu contexto nem sempre são explicitamente suportadas pelos mecanismos tradicionais de busca.

Esse cenário compromete a auditabilidade dos processos, reduz a confiabilidade das informações recuperadas e limita a reutilização do conhecimento organizacional. Em setores como jurídico, compliance e gestão de contratos, nos quais a rastreabilidade constitui um requisito essencial, a ausência de mecanismos capazes de preservar a estrutura documental e a proveniência das informações pode representar riscos operacionais significativos.

Diante desse cenário, define-se a seguinte pergunta de pesquisa: é possível sustentar um sistema de inteligência documental com auditabilidade utilizando exclusivamente modelagem relacional, sem depender de bancos vetoriais ou grafos como camada primária de recuperação?

Para investigar essa questão, propõe-se o ThingIndex, um sistema de inteligência documental fundamentado em uma arquitetura relacional. O projeto organiza coleções, documentos, páginas, parágrafos, tabelas, metadados e artefatos derivados em uma estrutura persistente e rastreável, apoiada principalmente em PostgreSQL. Dessa forma, a relevância da proposta está associada à crescente demanda por soluções capazes de tratar documentos de forma estruturada, segura e auditável. 
% Em muitos cenários organizacionais, a recuperação de informações ocorre por meio de mecanismos que não preservam explicitamente a proveniência dos dados nem as transformações realizadas durante seu processamento.

Além de sua relevância prática, o projeto apresenta interesse acadêmico ao integrar conceitos de engenharia de software, arquitetura de sistemas, modelagem de dados, processamento assíncrono e recuperação de informação, passível de implementação, avaliação e evolução.

Portanto, este documento apresenta a pesquisa completa: são descritos nas próximas seções os objetivos, a fundamentação teórica, os trabalhos relacionados, a metodologia, a proposta arquitetural do sistema e, em seguida, os resultados da avaliação experimental da recuperação, a discussão desses resultados com suas ameaças à validade e a conclusão do trabalho.

% O ThingIndex será desenvolvido como um artefato executável, composto por frontend, backend, pipeline assíncrono de indexação, mecanismos de observabilidade, suporte à avaliação experimental de desempenho e integração com sistemas externos. Assim, este artigo apresenta a motivação da pesquisa, a arquitetura proposta, o planejamento de desenvolvimento e a consolidação do sistema como um artefato de engenharia para gestão e recuperação de conhecimento documental.


% Aldo: Coloquei a justificativa no texto da introdução para não ficar redundante. 
% =============================================================================
% 2. JUSTIFICATIVA
% =============================================================================
% \section{Justificativa}

% A relevância deste trabalho está associada à necessidade crescente de sistemas que tratem documentos de forma estruturada, segura e auditável. Ferramentas generalistas de armazenamento ou busca frequentemente não oferecem rastreabilidade suficiente sobre como o contexto foi montado, quais transformações foram realizadas durante o processamento e de que maneira os dados podem ser reutilizados em fluxos posteriores de consulta, indexação ou interação assistida.

% Do ponto de vista prático, o desenvolvimento do ThingIndex se justifica por propor uma plataforma capaz de integrar persistência, organização documental, processamento assíncrono e recuperação de contexto em um único fluxo. Isso reduz fragmentação técnica e favorece manutenção, observabilidade e evolução incremental do sistema. Do ponto de vista acadêmico, o projeto é relevante por aplicar conceitos de engenharia de software, arquitetura de sistemas, modelagem de dados, filas de processamento e integração com provedores externos em um artefato real, analisável e demonstrável.

% Há ainda uma justificativa metodológica importante: em vez de assumir desde o início abordagens cuja auditabilidade ainda não é completamente previsível, o projeto estabelece uma baseline relacional concreta como ponto de partida. Essa decisão fortalece a comparabilidade futura com outras abordagens e delimita o escopo do trabalho ao que será efetivamente implementado.

% =============================================================================
% 3. OBJETIVOS
% =============================================================================
\section{Objetivos}

O presente estudo tem como objetivo geral propor e especificar o ThingIndex como uma plataforma open source de inteligência documental baseada em modelagem relacional, capaz de organizar, processar, indexar e recuperar contexto documental de forma auditável.

\subsection{Objetivos específicos}

\begin{itemize}
  \item identificar o problema de fragmentação e baixa rastreabilidade em fluxos de processamento documental;
  \item definir uma estrutura relacional para representar coleções, documentos, páginas, parágrafos, tabelas, metadados e artefatos derivados;
  \item projetar uma arquitetura full-stack;
  % \item especificar um pipeline assíncrono de indexação com mecanismos de fila, retries e observabilidade operacional;
  \item definir critérios objetivos de avaliação que permitam verificar auditabilidade, precisão na recuperação e custo operacional;
  \item estabelecer uma \textit{baseline} funcional e avaliável que sustente comparações futuras com abordagens vetoriais ou orientadas a grafo.
\end{itemize}

% =============================================================================
% 4. REFERENCIAL TEORICO
% =============================================================================
\section{Referencial Teórico}

Esta seção apresenta os fundamentos teóricos que sustentam as áreas de conhecimento envolvidas no projeto do ThingIndex. 
% Cada subseção aborda um conceito central, explica seu funcionamento e o relaciona com as decisões do sistema proposto.

\subsection{Inteligência Documental e Gestão do Conhecimento}

A gestão do conhecimento estuda como organizações criam, retêm e compartilham conhecimento. Nonaka e Takeuchi~\cite{nonaka1995knowledge} classificam o conhecimento em duas categorias: tácito, que reside na mente das pessoas e é difícil de articular; e explícito, registrado em documentos, manuais e bases de dados. A transição entre eles ocorre por meio da espiral do conhecimento, composta por quatro etapas: socialização, externalização, combinação e internalização. A externalização, conversão do conhecimento tácito em explícito, é a etapa mais relevante para sistemas documentais, pois é o momento em que a experiência individual se registra em um documento acessível a outros.

Davenport e Prusak~\cite{davenport1998working} argumentam que o valor do conhecimento está em sua capacidade de ser localizado e reutilizado. Sem mecanismos de organização e recuperação, o conhecimento explícito acumulado se perde. A inteligência documental emerge como a capacidade de um sistema de extrair, organizar e recuperar conhecimento a partir de documentos, preservando sua estrutura semântica, como parágrafos, tabelas, metadados e hierarquias, permitindo que o conhecimento seja localizado por conteúdo, não apenas por nome de arquivo.

Os estudos de Nonaka e Takeuchi~\cite{nonaka1995knowledge} e de Davenport e Prusak~\cite{davenport1998working} fundamentaram a decisão central do ThingIndex de tratar documentos não como arquivos monolíticos, mas como conjuntos de unidades de conhecimento extraíveis. Em vez de armazenar PDFs inteiros e buscar por nome de arquivo, o que ignoraria a espiral do conhecimento ao manter o conhecimento em estado bruto e não combinável, o sistema fragmenta cada documento em parágrafos, tabelas e metadados, armazenados em tabelas relacionais tipadas. Essa abordagem organiza o conhecimento explícito para que ele possa ser localizado, combinado e reutilizado, materializando na prática a etapa de combinação descrita por Nonaka e Takeuchi.

\subsection{Recuperação de Informação e Indexação Semântica}

Recuperação de Informação (RI) é a área que estuda como encontrar documentos relevantes a partir de uma necessidade de informação~\cite{manning2008ir}. Diferentemente de uma consulta a um banco de dados tradicional, a RI lida com texto não estruturado e relevância aproximada.

A área de RI pode ser dividida em duas grandes abordagens: os métodos clássicos, baseados em estatística textual, e os métodos neurais, baseados em vetores densos e representações contextuais.

O modelo clássico mais influente é o TF-IDF (Term Frequency: Inverse Document Frequency), proposto por Salton~\cite{salton1989automatic}. O TF-IDF calcula um peso para cada termo em cada documento: a frequência do termo no documento (TF) multiplicada pelo inverso da frequência do termo no corpus (IDF). Termos frequentes em um documento, mas raros no corpus geral, recebem peso alto. Robertson e Zaragoza~\cite{robertson2009bm25} aperfeiçoaram essa abordagem com o BM25, um modelo probabilístico que incorpora saturação de frequência de termo e normalização por tamanho do documento.

A limitação central dos métodos clássicos é o matching léxico: eles só encontram correspondência exata entre termos da consulta e termos do documento. Uma busca por ``automóvel'' não encontra documentos que mencionam ``carro''. Para superar essa limitação, surgiram os métodos neurais, que representam palavras como vetores densos em um espaço contínuo. Com o Word2Vec~\cite{mikolov2013efficient}, palavras com significado semelhante, como ``carro'' e ``automóvel'', passaram a ocupar posições próximas nesse espaço vetorial. O BERT~\cite{devlin2019bert} avançou ao gerar embeddings contextuais: a palavra ``banco'' tem vetores diferentes em ``banco de dados'' e ``banco do parque''. A similaridade de cosseno entre esses vetores permite buscar por sentido, não por palavra exata, formando a base de sistemas modernos de busca e de arquiteturas como o Retrieval-Augmented Generation (RAG), apresentada a seguir.

\subsection{Retrieval-Augmented Generation}

Modelos de linguagem de grande escala (Large Language Models - LLMs) como GPT, Llama e Gemini têm conhecimento limitado aos dados com os quais foram treinados. Quando confrontados com perguntas sobre documentos específicos, tendem a alucinar, isto é, gerar respostas confiantes porém incorretas~\cite{ji2023survey}.

O Retrieval-Augmented Generation (RAG) é uma arquitetura proposta por Lewis et al.~\cite{lewis2020rag}, pesquisadores do Facebook AI Research (FAIR), no artigo ``Retrieval-Augmented Generation for Knowledge-Intensive NLP Tasks''. A ideia central é combinar recuperação de informação com geração de texto: um retriever busca trechos relevantes em uma base de conhecimento externa, e um generator (LLM) recebe esses trechos como contexto adicional junto com a pergunta do usuário para gerar uma resposta fundamentada.

O pipeline típico do RAG funciona em cinco etapas:
\begin{enumerate}
  \item Ingestão: o documento original é recebido pelo sistema.
  \item Chunking: o documento é dividido em pedaços (chunks), geralmente de 256 a 1024 tokens (unidades mínimas de processamento de texto utilizadas pelos modelos de linguagem), com ou sem sobreposição.
  \item Indexação vetorial: cada chunk é convertido em vetor denso (embedding) e armazenado em um banco de dados vetorial (vector DB), como Pinecone, Weaviate ou pgvector.
  \item Consulta: a pergunta do usuário é convertida no mesmo espaço vetorial; o vector DB retorna os K chunks mais similares por similaridade de cosseno.
  \item Geração: os chunks recuperados são concatenados com a pergunta e enviados ao LLM, que gera uma resposta com base no contexto fornecido.
\end{enumerate}

A principal vantagem do RAG é que o conhecimento pode ser atualizado sem retreinar o modelo. No entanto, o RAG convencional apresenta limitações significativas: o chunking é geralmente cego (baseado em contagem de tokens, sem respeitar a estrutura semântica do documento); a recuperação por similaridade vetorial é uma caixa-preta, sem explicação do critério de matching; e a auditabilidade é limitada: rastrear de que documento, página e parágrafo veio cada informação exige mecanismos adicionais que a maioria das implementações não oferece.

O estudo do RAG, especialmente o artigo seminal de Lewis et al.~\cite{lewis2020rag} e das limitações documentadas por Ji et al.~\cite{ji2023survey} sobre alucinação, foi o principal motivador para a proposta do ThingIndex. Identificou-se que a arquitetura RAG resolve o problema de contexto, mas introduz um novo problema de opacidade. Dessa forma, a proposta é substituir o vector DB por uma baseline relacional auditável, isto é, em vez de fragmentar documentos em chunks genéricos e indexá-los por similaridade vetorial, o sistema preserva a estrutura semântica original em tabelas relacionais tipadas (detalhada na Seção 6.7).

% A recuperação de contexto ocorre por consultas SQL determinísticas sobre chaves estrangeiras, não por similaridade de cosseno. O uso de LLMs é mantido como componente opcional para enriquecimento e chat, mas a camada de recuperação é inteiramente transparente: cada parágrafo retornado sabe exatamente de que documento, página e posição veio. Em vez de um RAG clássico (vector DB + LLM), o ThingIndex propõe um RAG relacional: o contexto é montado por joins hierárquicos, e o LLM atua apenas como interface de consulta sobre esse contexto.

\subsection{Modelagem Relacional Aplicada a Sistemas Documentais}

O modelo relacional foi proposto por Codd~\cite{codd1970} no artigo ``A Relational Model of Data for Large Shared Data Banks''. Nele, os dados são organizados em relações (tabelas). Cada linha é uma tupla e cada coluna é um atributo. A consistência dos dados é garantida por três mecanismos:

\begin{itemize}
  \item Chaves e integridade referencial: cada tupla é identificada por uma chave primária. Referências entre tabelas são feitas por chaves estrangeiras, que apontam para chaves primárias de outras tabelas, garantindo que não existam referências órfãs.
  \item Normalização: para evitar redundância, os dados são organizados em formas normais. A primeira forma normal (1FN) exige colunas atômicas; a segunda (2FN) exige dependência total da chave; a terceira (3FN) elimina dependências transitivas entre colunas não-chave~\cite{elmasri2016fundamentals}.
  \item Propriedades ACID: transações em bancos relacionais garantem atomicidade (tudo ou nada), consistência (estado válido), isolamento (transações simultâneas não interferem) e durabilidade (dados persistidos mesmo em falha)~\cite{date2004introduction}.
\end{itemize}

Buneman et al.~\cite{buneman2001provenance} introduziram o conceito de proveniência de dados: a capacidade de rastrear a origem e as transformações de um dado dentro de um sistema. Em bancos relacionais, a proveniência pode ser implementada por meio de chaves estrangeiras, registros de auditoria e consultas determinísticas: cada dado pode ser rastreado até sua origem.

% O conceito de proveniência de Buneman será determinante para o design do modelo de dados do ThingIndex. Se o objetivo é oferecer auditabilidade, o modelo de dados precisava ser projetado para suportar rastreabilidade desde o início, não como uma camada adicional. Foi por isso que se optou por fragmentar o conteúdo dos documentos em tabelas tipadas (parágrafos, tabelas, metadados) em vez de usar um JSONB monolítico: a proveniência exige granularidade.

% O ThingIndex utiliza PostgreSQL como banco relacional principal. Cada tabela de conteúdo possui chaves estrangeiras que apontam para o documento de origem, além de índices compostos para preservar a ordenação original. Dessa forma, cada parágrafo recuperado pode ser rastreado até o documento, página e posição de origem por meio de consultas SQL. A proveniência, no ThingIndex, não é uma promessa: é uma constraint de esquema.

\subsection{Panorama Tecnológico}

Esta subseção apresenta as tecnologias escolhidas para o ThingIndex e como cada uma se relaciona com os fundamentos teóricos discutidos nas subseções anteriores.

Next.js. O frontend utiliza Next.js, framework React mantido pela Vercel. A escolha está alinhada com a necessidade de uma interface que permita ao usuário navegar, consultar e visualizar o conhecimento recuperado. O suporte nativo a Server Components permite renderizar páginas no servidor sem expor chaves de API, e o App Router simplifica a estrutura de rotas e layouts.

Go. A API e os workers de indexação são implementados em Go, linguagem criada na Google em 2009. A escolha foi influenciada pelo pipeline de processamento documental: o parsing e a extração de parágrafos e tabelas (descritos na Seção~6.7) precisam ocorrer de forma concorrente para múltiplos documentos. Go oferece goroutines para concorrência sem a complexidade de threads tradicionais, e a compilação para binário único reduz erros em tempo de execução.

PostgreSQL. O banco de dados principal é o PostgreSQL, sistema relacional maduro. A escolha é uma consequência direta dos fundamentos apresentados na Seção~3.4: o modelo relacional de Codd~\cite{codd1970}, as propriedades ACID e a proveniência de Buneman~\cite{buneman2001provenance} só são plenamente realizáveis com um banco que ofereça chaves estrangeiras, integridade referencial e índices avançados. O PostgreSQL oferece ainda a extensão pgvector para futuras comparações com abordagens semânticas.

Object Storage. Os arquivos brutos (PDFs) são armazenados em object storage compatível com a API S3. A separação entre armazenamento de objetos e banco relacional é intencional: permite acesso aos arquivos via URLs temporárias (presigned URLs) sem sobrecarregar a API, e mantém o banco com tamanho gerenciável.

Filas Assíncronas (RabbitMQ). O pipeline de indexação é orquestrado por filas de mensagens utilizando RabbitMQ. O padrão outbox é empregado para reduzir o risco de perda de eventos de indexação: quando um documento é persistido no banco, um evento de indexação é registrado na mesma transação. Um dispatcher lê os eventos da tabela de outbox e os publica no RabbitMQ. O RabbitMQ foi escolhido por sua maturidade, suporte a filas de retry e dead-letter queues (DLQ).

Docker Compose. Todo o ambiente de desenvolvimento é reproduzível via Docker Compose, com healthchecks entre serviços, garantindo que o sistema possa ser executado com um único comando independentemente do sistema operacional.

% =============================================================================
% 5. TRABALHOS RELACIONADOS
% =============================================================================
\section{Trabalhos Relacionados}

Esta seção analisa frameworks e plataformas que atuam em áreas próximas ao ThingIndex. Para cada ferramenta, são descritos sua origem, funcionamento e limitações, destacando as lacunas de pesquisa que o ThingIndex se propõe a preencher.

\subsection{Notion}

O Notion, criado pela Notion Labs Inc. (fundada por Ivan Zhao e Simon Last em 2013), é um workspace all-in-one que combina documentos, bancos de dados e gerenciamento de tarefas. Sua arquitetura é baseada em blocos aninhados (block-based editor), onde cada elemento (texto, imagem, tabela, banco de dados) é um bloco que pode conter outros blocos. O Notion oferece busca full-text e integração com IA para resumo e geração de texto.

A principal limitação do Notion é a ausência de uma API pública de indexação semântica e a dificuldade de auditar o contexto recuperado. O conhecimento fica preso na plataforma, e não há um pipeline de processamento documental que permita rastrear a origem de cada trecho recuperado.

\subsection{LangChain}

O LangChain, criado por Harrison Chase em 2022 (LangChain Inc.), é um framework para construção de aplicações baseadas em LLMs. Sua arquitetura é baseada em \textit{chains}: sequências de chamadas a modelos de linguagem, intercaladas com recuperação de informação (\textit{retrievers}), ferramentas externas (\textit{tools}) e agentes autônomos (\textit{agents}). O LangChain se tornou o framework mais popular para prototipação de sistemas RAG, oferecendo integrações prontas com dezenas de provedores de LLM e bancos vetoriais.

A limitação do LangChain está na opacidade do pipeline de recuperação: o desenvolvedor configura um retriever, mas tem pouca visibilidade sobre como o contexto foi segmentado, ordenado e montado. O custo operacional depende de serviços externos (API de embeddings + vector DB + LLM), e a auditabilidade do contexto recuperado é limitada.

\subsection{LlamaIndex}

O LlamaIndex (originalmente GPT Index), criado por Jerry Liu em 2022, é um framework focado em indexação e recuperação de dados para LLMs. Sua arquitetura suporta múltiplos tipos de índice (lista, vetor, árvore, documento) e permite conectar fontes de dados heterogêneas (PDFs, bancos de dados, APIs). O LlamaIndex se tornou referência para construção de sistemas RAG, oferecendo componentes prontos para ingestão, chunking, embedding e consulta.

A limitação do LlamaIndex está na opacidade do pipeline de recuperação: o desenvolvedor escolhe configurações de chunking e embedding, mas tem baixa visibilidade sobre como o contexto é montado. A dependência de bancos vetoriais torna o custo variável, e a auditabilidade do contexto recuperado é limitada: não há garantia de que cada trecho retornado possa ser rastreado até o documento e página de origem.

\subsection{PageIndex}

O PageIndex~\cite{pageindex2025} é o trabalho mais próximo desta proposta e, por isso, o mais relevante para delimitá-la. Trata-se de uma ferramenta de indexação de PDFs que extrai a hierarquia estrutural do documento --- seções, subseções e intervalos de página --- e a materializa como uma árvore navegável, deliberadamente sem embeddings e sem banco vetorial. A recuperação percorre essa árvore em vez de comparar vetores, e a resposta aponta para o nó e a página de origem.

A convergência é considerável: a premissa de que a estrutura já presente no documento é sinal suficiente para recuperação, e a rejeição do índice vetorial como camada primária, são as mesmas deste trabalho. Registrar a semelhança é mais honesto do que apresentar a abordagem como inédita.

Três diferenças delimitam a contribuição. A primeira é a unidade de persistência: o PageIndex produz um artefato de árvore por documento, enquanto o ThingIndex normaliza a estrutura em tabelas relacionais consultáveis por SQL, o que permite consulta transversal ao acervo sem carregar cada árvore em memória. A segunda é o escopo do sistema: o PageIndex resolve indexação e recuperação, sem tratar do ciclo operacional --- fila com retry e \emph{dead-letter}, catálogo de provedores, trilha de auditoria. A terceira, e a mais relevante para a questão de pesquisa, é o objeto da avaliação: o PageIndex reporta qualidade de recuperação, enquanto este trabalho mede também a estabilidade da ordenação sob reindexação e a completude da proveniência, dimensões que a comparação entre paradigmas exige e que raramente aparecem quantificadas.

O PageIndex é usado neste projeto como referência de qualidade estrutural. O benchmark de indexação compara o perfil produzido pelo indexador determinístico contra artefatos normalizados de três documentos processados pela ferramenta --- um relatório anual corporativo, um documento técnico e um processo judicial brasileiro --- em profundidade da árvore, aninhamento, cobertura de resumos e concisão de títulos. Cabe a ressalva registrada no próprio artefato: esses arquivos são \emph{barra de qualidade}, não gabarito perfeito. Um deles contém intervalo de página invertido, e a comparação existe para detectar regressão estrutural, não para afirmar superioridade.

\subsection{Lacunas de pesquisa}

Nenhuma das ferramentas analisadas oferece simultaneamente: (i) pipeline assíncrono de indexação com retry e DLQ; (ii) recuperação de contexto por consultas SQL determinísticas com rastreabilidade total da origem; (iii) desacoplamento de provedores de IA; (iv) custo previsível sem dependência obrigatória de vector DB.

A lacuna mais específica, contudo, não é de funcionalidade e sim de evidência. O PageIndex demonstra que recuperação estrutural sem vetores é viável, mas a literatura de comparação entre paradigmas de recuperação raramente reporta duas propriedades que este trabalho mede: a estabilidade da ordenação quando o índice é reconstruído, e a fração de itens recuperados cuja origem pode ser apontada com precisão de página. É nesse espaço --- medir o que costuma ser afirmado --- que a contribuição se posiciona.
% O ThingIndex se propõe a ocupar esse espaço, priorizando auditabilidade e determinismo sobre flexibilidade semântica, sem excluir a possibilidade de extensões futuras com embeddings ou grafos.

% =============================================================================
% 6. METODOLOGIA
% =============================================================================
\section{Metodologia}

Este trabalho caracteriza-se como uma pesquisa aplicada orientada pela Design Science Research (DSR)~\cite{hevner2004design}, que estabelece que a construção de um artefato de software e sua avaliação segundo critérios explícitos de utilidade e rigor são inseparáveis da investigação do problema. A pesquisa investiga o problema de fragmentação documental, propõe o artefato ThingIndex como solução, descreve sua arquitetura, implementa o sistema e o avalia experimentalmente segundo critérios definidos a priori.

Quanto aos procedimentos, o trabalho segue os ciclos da DSR propostos por Hevner et al.~\cite{hevner2004design}: (i) conscientização do problema, (ii) sugestão, (iii) desenvolvimento, (iv) avaliação e (v) conclusão.

A revisão da literatura de Recuperação de Informação (RI) e RAG, apresentada na Seção~3, evidenciou as limitações das abordagens puramente lexicais (matching exato) e a opacidade dos métodos neurais (embeddings e bancos vetoriais) no que diz respeito à auditabilidade do contexto recuperado. Essa constatação fundamentou a decisão metodológica de adotar a modelagem relacional como baseline primária, priorizando determinismo e rastreabilidade sobre flexibilidade semântica. O desenvolvimento incremental do artefato permitirá avaliar, a cada ciclo, se a abordagem relacional atende aos critérios de precisão e auditabilidade definidos na Proposta.

A classificação como exploratória justifica-se pela investigação de um problema pouco estudado na literatura: o uso de modelagem relacional pura como baseline para inteligência documental em contraste com abordagens vetoriais predominantes. A natureza descritiva se dará na apresentação detalhada da arquitetura, do modelo de dados, dos fluxos de processamento e da estratégia de validação.

As etapas metodológicas do trabalho serão organizadas da seguinte forma:

\begin{itemize}
  \item levantamento do problema e definição do escopo do sistema;
  \item revisão bibliográfica dos conceitos que sustentam a proposta;
  \item levantamento e organização dos requisitos funcionais e não funcionais;
  \item modelagem da arquitetura e das estruturas de dados;
  \item implementação incremental do software;
  \item execução de testes e validação da solução desenvolvida;
  \item análise dos resultados obtidos.
\end{itemize}

% =============================================================================
% 7. PROPOSTA
% =============================================================================
\section{Proposta}

Esta seção apresenta o ThingIndex, detalhando sua definição, arquitetura, modelo de dados, fluxo de processamento, tecnologias, protótipo das telas, diagramas, critérios de avaliação e cronograma de desenvolvimento.

\subsection{O que é o ThingIndex?}

O ThingIndex é uma plataforma open source de inteligência documental baseada em modelagem relacional. Seu objetivo é organizar coleções de documentos PDF, extrair seu conteúdo estrutural (parágrafos, tabelas, metadados) e permitir a recuperação auditável do contexto por meio de consultas SQL determinísticas.

Diferentemente de sistemas RAG convencionais, que dependem de bancos vetoriais e embeddings opacos, o ThingIndex prioriza a rastreabilidade, isto é, cada parágrafo recuperado pode ser rastreado até o documento, página e posição de origem. A arquitetura, detalhada na Seção~6, combina tecnologias para frontend, API, persistência, armazenamento e processamento assíncrono.

A construção de uma plataforma com tais características se justifica pelos fundamentos de inteligência documental e gestão do conhecimento discutidos na Seção~3.1: sem mecanismos de organização e recuperação, o conhecimento explícito acumulado em documentos acadêmicos e organizacionais se perde. O ThingIndex propõe preencher essa lacuna com uma abordagem que privilegia auditabilidade e determinismo sobre flexibilidade semântica, sem excluir a possibilidade de extensões futuras com embeddings ou grafos.

A opção pelo modelo open source não é apenas uma decisão de licenciamento, mas um compromisso com a replicabilidade e a evolução contínua do projeto. Ao tornar o código fonte acessível, a comunidade acadêmica e técnica poderá examinar, testar e contribuir com extensões, promovendo um ciclo de melhoria que beneficia não apenas o ThingIndex, mas o ecossistema de ferramentas de inteligência documental como um todo. O repositório será mantido público após a conclusão do TCC, permitindo que outros pesquisadores e desenvolvedores realizem forks, submetam contribuições e comparem a abordagem relacional com outras estratégias de recuperação de informação.

\subsection{Aplicação dos Fundamentos Teóricos ao ThingIndex}

O ThingIndex seguirá a abordagem de pesquisa aplicada descrita por Hevner et al.~\cite{hevner2004design}, na qual a construção do artefato de software será inseparável da investigação do problema. O projeto não se limitará a propor uma arquitetura, mas a implementará como monorepo executável, com testes, pipeline de integração e documentação versionada. A organização incremental do código, com autenticação primeiro, depois coleções, depois pipeline assíncrono, refletirá a preocupação com a entrega de valor funcional em cada etapa, coerente com o ciclo de vida iterativo descrito por Pressman e Maxim~\cite{pressman2014}.

A arquitetura do ThingIndex adotará separação em camadas com fronteiras explícitas entre interface, aplicação, domínio e infraestrutura. A propriedade mais importante será o desacoplamento: o módulo de autenticação não conhecerá detalhes de storage, o pipeline de indexação não dependerá de um provedor de LLM específico e a recuperação de contexto funcionará independentemente de existir ou não um chat configurado. A aplicação de DDD será pragmática: o domínio conterá contratos e erros de negócio, enquanto adaptadores para serviços externos serão mantidos separadamente. Isso garantirá que a troca de um provedor de armazenamento (de MinIO para GCS, por exemplo) ou de um modelo de linguagem não exija alteração na lógica de coleção ou de recuperação de contexto.

O modelo relacional será a base sobre a qual o ThingIndex construirá sua proposta de auditabilidade. A proveniência descrita por Buneman et al.~\cite{buneman2001provenance} será materializada literalmente: o contexto de uma coleção poderá ser reconstruído por uma sequência de comandos SQL determinísticos, sem depender de vetores opacos ou de caminhos de inferência não estruturados. A ordem semântica será preservada por índices de ordenação, os vínculos por chaves estrangeiras e o histórico de processamento por tabelas de eventos e tentativas. O sistema optará por consistência transacional (ACID) via PostgreSQL. Essa escolha será deliberadamente conservadora: o sistema aceitará o custo de schema rígido e normalização em troca de previsibilidade operacional, facilidade de auditoria e clareza para fins acadêmicos. Outras abordagens, vetoriais ou de grafo, não serão excluídas, mas relegadas a trilhas comparativas futuras.

% Aldo: Texto redundante, já foi justificado anteriormente ....
% O ThingIndex posiciona-se como uma alternativa complementar às abordagens puramente vetoriais: recuperação estruturada por modelo relacional com joins hierárquicos. Em vez de converter documentos em vetores e consultar por similaridade de cosseno, o sistema preservará a estrutura semântica original (parágrafos, tabelas, metadados) em tabelas tipadas e reconstruirá o contexto por agregação linear. Essa abordagem não competirá em semântica flexível com um sistema RAG, mas superará em determinismo, custo previsível e auditabilidade. A intuição de Xu et al.~\cite{xu2020layoutlm} sobre compreensão de layout será incorporada de forma prática ao separar parágrafos de tabelas e preservar a ordem de leitura, mesmo sem utilizar um modelo de layout como núcleo do pipeline.

A seguir são apresentados os requisitos funcionais e não funcionais do projeto. 

\subsection{Requisitos Funcionais}

\begin{itemize}
  \item RF01: o sistema deve permitir o cadastro, login, logout e gerenciamento de perfil de usuários;
  \item RF02: o sistema deve permitir a criação, edição, listagem e exclusão de categorias documentais;
  \item RF03: o sistema deve permitir a criação, edição, listagem e exclusão de coleções documentais;
  \item RF04: o sistema deve permitir o envio de documentos PDF com validação de tipo;
  \item RF05: o sistema deve armazenar documentos em object storage;
  \item RF06: o sistema deve processar documentos de forma assíncrona, extraindo parágrafos, tabelas, metadados, páginas, chunks e outline;
  \item RF07: o sistema deve permitir enriquecimento opcional por LLM durante a indexação;
  \item RF08: o sistema deve possibilitar a recuperação de contexto completa de uma coleção via API REST;
  \item RF09: o sistema deve oferecer chat grounded por coleção com transmissão via SSE e citação de fontes;
  \item RF10: o sistema deve permitir configuração de provedores externos (LLM e storage) com criptografia de credenciais;
  \item RF11: o sistema deve fornecer painel de monitoramento com telemetria por documento e agregada por usuário;
  \item RF12: o sistema deve expor benchmark público de indexação com corpus controlado;
  \item RF13: o sistema deve registrar trilha de auditoria para operações em documentos e provedores;
  \item RF14: o sistema deve permitir retry manual de documentos falhos e detecção de documentos órfãos;
  \item RF15: o sistema deve suportar internacionalização (PT-BR e EN-US) nas respostas da API.
\end{itemize}

\subsection{Requisitos Não Funcionais}

\begin{itemize}
  \item RNF01: o sistema deve apresentar interface responsiva e navegação clara;
  \item RNF02: o sistema deve garantir integridade referencial via chaves estrangeiras e consistência transacional (ACID) no PostgreSQL;
  \item RNF03: o sistema deve suportar documentos e coleções com centenas de documentos sem degradação significativa;
  \item RNF04: o sistema deve permitir manutenção e evolução incremental, com separação de camadas e fronteiras de domínio explícitas;
  \item RNF05: o sistema deve adotar mecanismos de segurança: hash para senhas, tokens para autenticação, criptografia para credenciais, rate limiting em rotas sensíveis e sanitização de entradas;
  \item RNF06: o sistema deve ser reproduzível via Docker Compose, com ambiente de desenvolvimento configurável em máquinas Linux, macOS e Windows (WSL);
  \item RNF07: o sistema deve possuir cobertura de testes automatizados em nível unitário, integração e end-to-end;
  \item RNF08: o sistema deve gerar documentação da API automaticamente via Swagger/OpenAPI a partir de anotações no código.
\end{itemize}


% *************************************
% ALDO: trouxe para cá as subseções da seção 7. Arquitetura e Modelagem do Sistema.
% Caso não fique legal, só excluir os textos abaixo e descomentar a seção 7.
% *************************************

\subsection{Visão Geral da Arquitetura}

O sistema será organizado em quatro camadas principais: interface de usuário, API, persistência e infraestrutura auxiliar (object storage, filas), cujas tecnologias são detalhadas na Seção~3.5. A Figura~\ref{fig:arquitetura} ilustrará essa composição. Richards~\cite{richards2015patterns} classifica essa organização como uma variação do padrão \textit{Layered Architecture}, no qual cada camada possui responsabilidade bem definida e a dependência flui em uma única direção.

A API será a peça central. Ela exporá rotas HTTP para autenticação, gerenciamento de coleções, upload de documentos, configuração de provedores, recuperação de contexto e chat. As rotas serão organizadas em \textit{handlers} finos, que apenas validarão entrada, autenticarão e serializarão respostas. A lógica de negócio residirá em \textit{use cases}, que orquestrarão chamadas a \textit{repositories} e serviços auxiliares.

A Figura~\ref{fig:visao-geral-sistema} apresenta a visão geral da arquitetura do ThingIndex, consolidando a integração entre frontend, API, banco de dados, filas e provedores de IA.

\begin{figure}[!ht]
  \centering
  \includegraphics[width=1\textwidth]{anexos/diagramas/visao-geral-sistema.png}
  \caption{Visão geral da arquitetura do sistema.}
  \label{fig:visao-geral-sistema}
\end{figure}

A Figura~\ref{fig:casos-uso} apresenta o diagrama de casos de uso do ThingIndex, com os atores e as funcionalidades disponíveis no sistema.

\begin{figure}[!ht]
  \centering
  \includegraphics[width=1\textwidth]{anexos/diagramas/casos-de-uso.png}
  \caption{Diagrama de casos de uso do ThingIndex.}
  \label{fig:casos-uso}
\end{figure}

O pipeline assíncrono será implementado por meio do padrão \textit{outbox}: quando um documento for enviado, um evento será gravado na mesma transação do banco que persistir o documento. Um processo \textit{dispatcher} realizará polling na tabela de outbox e publicará os eventos no RabbitMQ. Essa estratégia reduz o risco de perda de mensagens: se a API cair antes de publicar no broker, o evento ainda estará no banco e será reenviado na próxima rodada do dispatcher. Richardson~\cite{richardson2018microservices} descreve esse padrão como a solução mais robusta para o problema de dualidade de escrita (banco + broker) em arquiteturas baseadas em microsserviços.

O worker \textit{indexer} consumirá a fila, processará o documento e persistirá as estruturas extraídas. Se o processamento falhar, o evento será reenviado para uma fila de \textit{retry} com atraso configurável. Após esgotar as tentativas, ele será movido para uma \textit{dead-letter queue} (DLQ), onde poderá ser inspecionado manualmente. Hohpe e Woolf~\cite{hohpe2003eip} classificam esses mecanismos como \textit{Dead Letter Channel} e \textit{Resequencer}, padrões consolidados de integração empresarial que permitem entrega confiável mesmo em cenários de falha parcial.

\begin{figure}[!ht]
  \centering
  \includegraphics[width=0.65\textwidth]{anexos/diagramas/arquitetura.png}
  \caption{Arquitetura runtime do ThingIndex, com separação entre camada web, API, persistência e processamento assíncrono.}
  \label{fig:arquitetura}
\end{figure}

\FloatBarrier

\subsection{Modelagem de Dados}

O modelo de dados será projetado para que o contexto de uma coleção seja reconstruído por consultas SQL determinísticas. O esquema completo contempla 21 tabelas organizadas em três grupos lógicos: núcleo, estruturas extraídas e infraestrutura, conforme descrito a seguir e disponível na íntegra no Apêndice A. A Figura~\ref{fig:schema-core} apresenta o diagrama entidade-relacionamento do grupo principal (núcleo), com as entidades fundamentais do sistema.

A Figura~\ref{fig:schema-core} apresenta o diagrama entidade-relacionamento do núcleo do sistema, composto pelas entidades fundamentais: usuários, categorias, coleções e documentos. Esse núcleo estabelece as relações básicas sobre as quais todo o sistema se apoia. Um \texttt{user} pode possuir múltiplas \texttt{categories} e \texttt{collections}, e cada \texttt{collection} agrupa diversos \texttt{documents}. A \texttt{collection} pode opcionalmente pertencer a uma \texttt{category}, permitindo organização hierárquica. Cada \texttt{document} armazena metadados como caminho, tipo MIME, tamanho e status de processamento.

\begin{figure}[!ht]
  \centering
  \includegraphics[width=0.85\textwidth]{anexos/database/diagrama-database-principal.png}
  \caption{Diagrama entidade-relacionamento do núcleo: usuários, categorias, coleções e documentos.}
  \label{fig:schema-core}
\end{figure}

A partir do documento, as estruturas extraídas são fragmentadas em seis tipos de tabelas, cada uma responsável por um aspecto específico do conteúdo:
\begin{itemize}
  \item páginas: conteúdo textual de cada página, com resumo e estimativa de tokens;
  \item parágrafos: blocos textuais com posição sequencial e número da página;
  \item tabelas: conteúdo tabular com cabeçalhos e linhas;
  \item metadados: pares chave-valor associados ao documento;
  \item unidades de recuperação: trechos com tipo, título e vínculo aos parágrafos de origem;
  \item hierarquia de seções: estrutura de tópicos com níveis e referência ao pai.
\end{itemize}

Além das entidades de domínio, o esquema inclui tabelas de infraestrutura: eventos de outbox para o padrão de mensageria, tentativas de processamento para rastreabilidade de falhas, eventos de auditoria para trilha de operações, configurações de provedores com credenciais criptografadas, conversas e mensagens de chat, além de dados de precificação para rastreio de custos.

A fragmentação intencional do conteúdo em tabelas separadas evita colunas polimórficas ou JSONB monolítico, garantindo que cada tipo de dado tenha seu próprio esquema bem definido. Índices compostos e chaves estrangeiras preservam a ordenação original e a integridade referencial entre as tabelas.

A Figura~\ref{fig:classes} complementa essa visão com o diagrama de classes, que organiza as 14 entidades nos três grupos lógicos. O diagrama de classes permite visualizar não apenas as tabelas, mas também os métodos associados a cada entidade e os relacionamentos com suas cardinalidades, oferecendo uma perspectiva orientada a objetos do modelo de dados.

\begin{figure}[!ht]
  \centering
  \includegraphics[width=1\textwidth]{anexos/diagramas/classes.png}
  \caption{Diagrama de classes do ThingIndex, com 16 classes organizadas em núcleo (azul), estruturas (verde) e infraestrutura (roxo).}
  \label{fig:classes}
\end{figure}

\FloatBarrier

\subsection{Fluxo de Processamento}

O fluxo de processamento de um documento no ThingIndex pode ser dividido em três etapas principais, cada uma com responsabilidades bem definidas dentro da arquitetura do sistema.

A primeira etapa compreende o upload e a validação do documento. Quando o usuário envia um arquivo PDF, a API verifica o tipo MIME e a existência da coleção de destino. O arquivo é armazenado em object storage e registrado no banco de dados com status pendente. Nesse momento, um evento de indexação é inserido em uma tabela de outbox na mesma transação, garantindo que nenhum documento fique sem processamento mesmo em caso de falha da API.

A segunda etapa é a indexação assíncrona, executada por um worker dedicado. O dispatcher lê os eventos da tabela de outbox e os publica no RabbitMQ. O worker indexer consome a mensagem e realiza o parsing do PDF, extraindo parágrafos, tabelas e metadados. Opcionalmente, um LLM pode enriquecer o conteúdo extraído. Todas as estruturas são persistidas no banco relacional, e a telemetria do processamento é registrada para auditoria. Em caso de falha, o evento é reenviado para uma fila de retry e, após esgotar as tentativas, movido para uma dead-letter queue para inspeção manual.

A terceira etapa é a recuperação de contexto. Quando uma consulta é realizada, o sistema executa joins SQL sobre as tabelas tipadas, monta um JSON hierárquico com todo o contexto da coleção e disponibiliza o resultado para o chat com citação de fontes.

\begin{figure}[!ht]
  \centering
  \includegraphics[width=0.95\textwidth]{anexos/diagramas/fluxo-processamento.png}
  \caption{Fluxo completo: upload, enfileiramento, indexação e recuperação de contexto.}
  \label{fig:fluxo}
\end{figure}

\subsubsection{Upload e Indexação}

O usuário autenticado envia um arquivo PDF. A API valida o tipo MIME e a existência da coleção de destino. O arquivo é enviado ao object storage e registrado no PostgreSQL com status pendente. Um evento de indexação é inserido na tabela de outbox na mesma transação.

O dispatcher publica o evento no RabbitMQ. O worker indexer consome a mensagem e executa: leitura do arquivo, parsing, extração de parágrafos e tabelas, enriquecimento opcional por LLM, persistência das estruturas e registro de telemetria. Em caso de falha, o evento segue para retry ou dead-letter queue. A Figura~\ref{fig:fluxo} ilustra esse fluxo completo.

\subsubsection{Interação entre Componentes}

Para complementar a visão do fluxo de processamento, o diagrama de sequência apresentado a seguir mostra a ordem temporal das chamadas entre os componentes do sistema. Nele é possível visualizar como o frontend se comunica com a API, como a API interage com o banco de dados e o storage, como o evento de indexação é propagado para o RabbitMQ e consumido pelo worker, e finalmente como o contexto é recuperado e exibido ao usuário.

A Figura~\ref{fig:sequencia} detalha essa comunicação durante os três fluxos principais, abrangendo desde o upload do documento até a resposta do chat com citação de fontes.

\begin{figure}[!ht]
  \centering
  \includegraphics[width=0.85\textwidth]{anexos/diagramas/sequencia.png}
  \caption{Diagrama de sequência do ThingIndex, ilustrando a comunicação entre frontend, API, storage, banco, fila e worker.}
  \label{fig:sequencia}
\end{figure}


% *************************************
% FIM das subsções que eram da seção 7
% *************************************

\FloatBarrier

\subsection{Prototipação de Telas}

Esta subseção apresenta os protótipos de tela do ThingIndex, ilustrando a interface proposta para o sistema.

\subsubsection{Landing Page}

A landing page apresenta a proposta de valor do ThingIndex. A Figura~\ref{fig:ui-home} ilustra essa tela.

\begin{figure}[H]
  \centering
  \includegraphics[width=0.95\textwidth]{anexos/telas/pagina-inicial.png}
  \caption{Página inicial do ThingIndex.}
  \label{fig:ui-home}
\end{figure}

\subsubsection{Dashboard}

O dashboard operacional oferece gerenciamento de categorias e coleções, listagem de documentos com preview lateral e telemetria, upload por arrastar-e-soltar, painel de monitoramento agregado e tela de chat com histórico de conversas e fontes consultadas. A Figura~\ref{fig:ui-dashboard} ilustra essa tela.

\begin{figure}[H]
  \centering
  \includegraphics[width=0.95\textwidth]{anexos/telas/dashboard.png}
  \caption{Dashboard operacional.}
  \label{fig:ui-dashboard}
\end{figure}

\subsubsection{Documentos}

A tela de documentos permite visualizar e gerenciar os arquivos enviados, com informações de status de processamento e acesso ao conteúdo extraído. A Figura~\ref{fig:ui-documents} ilustra essa tela.

\begin{figure}[H]
  \centering
  \includegraphics[width=0.95\textwidth]{anexos/telas/documents.png}
  \caption{Tela de gerenciamento de documentos.}
  \label{fig:ui-documents}
\end{figure}

\subsubsection{Configurações}

A tela de configurações permite ao usuário gerenciar provedores de IA e storage, com criptografia de credenciais no backend. O usuário poderá definir provedores distintos para indexação e chat. A Figura~\ref{fig:ui-settings} ilustra essa tela.

\begin{figure}[H]
  \centering
  \includegraphics[width=0.95\textwidth]{anexos/telas/settings.png}
  \caption{Tela de configurações de provedores.}
  \label{fig:ui-settings}
\end{figure}

\FloatBarrier

\subsection{Avaliação da Qualidade}

Para validar se o ThingIndex atende aos objetivos propostos, foram aplicados critérios de avaliação divididos em três dimensões: precisão na recuperação, auditabilidade e custo operacional, apresentadas a seguir.

\subsubsection{Precisão na Recuperação de Contexto}

A precisão foi medida por meio de experimento controlado sobre o corpus C1, descrito no Protocolo Experimental. Adotou-se o painel padrão da área de Recuperação de Informação --- P@1, P@3, R@5, MRR e nDCG@10 --- em lugar de uma taxa de acerto binária. A razão é de validade de construto: uma taxa de acerto trata igualmente o sistema que devolve o documento correto em primeiro lugar e o que o devolve em quinto, e a ordenação é precisamente o que distingue uma recuperação útil de uma inútil quando o resultado alimenta um modelo de linguagem com janela de contexto limitada.

O experimento consistiu em:
\begin{enumerate}
  \item construir um corpus com documentos de estrutura conhecida e gabarito de relevância graduada, sob regras anti-circularidade travadas por teste;
  \item executar cada consulta em seis braços de recuperação, em delineamento pareado;
  \item registrar uma linha por par (consulta, braço) na saída bruta, preservando os valores individuais exigidos pelo teste estatístico pareado;
  \item comparar os braços por Wilcoxon com correção de Bonferroni, reportando tamanho de efeito e intervalos de confiança por \emph{bootstrap}.
\end{enumerate}

A medição foi conduzida em duas granularidades --- documento e seção ---, pelas razões de validade de construto detalhadas na Seção~\ref{sec:granularidade}.

\subsubsection{Auditabilidade}

A auditabilidade foi verificada rastreando a origem de cada item indexado. Para cada elemento do índice, o sistema deve ser capaz de informar documento de origem, número da página, posição sequencial e tipo de estrutura. Duas métricas operacionalizam essa verificação --- a taxa de proveniência completa (TPC) e a taxa de citação verificável (TCV), definidas e medidas na Seção~\ref{sec:auditabilidade} ---, reportadas por tipo estrutural para que uma média global não oculte um tipo com proveniência ausente.

\subsubsection{Custo Operacional}

O custo operacional é a dimensão menos desenvolvida deste ciclo, e é declarada aqui como lacuna parcial em vez de omitida. A instrumentação foi implementada: o pipeline registra tempo por etapa, e o consumo de tokens reportado pelos provedores (campo \texttt{usage} das respostas) passou a ser capturado e exposto ao chamador, fechando o circuito com a tabela de preços que o sistema já sincronizava. O que não foi feito a tempo é a agregação desses valores em custo por documento sobre um corpus representativo, com variação por provedor.

A distinção importa para a leitura dos resultados: o que existe hoje é a capacidade de medir, não a medição. Duas decisões de projeto preservam a honestidade dessa métrica quando ela for produzida. Primeiro, provedores que não reportam consumo --- como instâncias locais de Ollama --- são registrados como \emph{não informado}, categoria distinta de \emph{zero}: agregar os dois no mesmo balde subestimaria o custo de qualquer configuração híbrida. Segundo, o consumo é lido do que o provedor devolve, e não estimado por contagem local de caracteres; estimativa de tokens a partir do texto enviado ignora tokens de sistema e de formatação, e o erro cresce justamente nos prompts longos que dominam o custo real.

As estimativas de custo apresentadas em trabalhos anteriores do projeto permanecem rotuladas como estimativas, e não como medição.

\subsection{Cronograma}

O desenvolvimento do ThingIndex foi conduzido ao longo de 2026, conforme ilustrado na Figura~\ref{fig:cronograma}.

\begin{figure}[!ht]
  \centering
  \includegraphics[width=1\textwidth]{anexos/diagramas/cronograma.png}
  \caption{Cronograma de desenvolvimento do ThingIndex.}
  \label{fig:cronograma}
\end{figure}

\FloatBarrier

% =============================================================================
% RESULTADOS
% =============================================================================
\section{Resultados}

Esta seção apresenta os resultados do ciclo de avaliação da DSR~\cite{hevner2004design}. O artefato descrito na Seção anterior foi integralmente implementado como monorepo executável (API e workers em Go, frontend em Next.js, PostgreSQL com as 21 tabelas do Apêndice A, pipeline assíncrono com outbox, retry e DLQ), com testes automatizados nos níveis unitário, de integração e end-to-end. Sobre esse artefato foi conduzido um experimento comparativo de recuperação de informação, cujo protocolo e resultados são descritos a seguir. Todos os números reportados são gerados por scripts a partir da saída bruta do experimento — nenhuma tabela foi transcrita manualmente — e o experimento completo é reproduzível com dois comandos a partir do repositório do projeto.

\subsection{Protocolo Experimental}

\textbf{Corpus.} A avaliação utilizou o corpus C1, composto por 5 documentos em português (domínios jurídico, licitatório, contratual, normativo e operacional) e 18 consultas, das quais 16 possuem resposta no corpus e 2 são casos negativos explícitos, cuja resposta correta é não recuperar nada. As consultas positivas são estratificadas em três variantes: \emph{literal} (a consulta reutiliza termos do trecho-alvo, servindo de controle de teto), \emph{paráfrase} (a mesma necessidade de informação expressa com outras palavras) e \emph{ruidosa} (consulta com termos irrelevantes adicionados). A relevância é graduada de 0 a 3.

Três regras anti-circularidade foram aplicadas na construção do corpus e estão travadas por testes automatizados que falham o build caso sejam violadas: (i) o gabarito referencia documentos por identificador, nunca por título produzido pelo próprio pipeline; (ii) as consultas não literais não podem compartilhar termos com o trecho-alvo além de um limiar verificado automaticamente; (iii) os casos negativos são marcados explicitamente, distinguindo ``nada é relevante'' de ``ninguém anotou''.

\textbf{Unidade de recuperação.} O experimento é executado em duas granularidades sobre o mesmo corpus. Na granularidade de \emph{documento}, cada um dos 5 documentos é uma unidade recuperável --- configuração que corresponde à avaliação clássica de RI, mas que descasa da tese deste trabalho, cuja contribuição é proveniência de parágrafo. Na granularidade de \emph{seção}, o corpus é fragmentado nas 19 seções de conteúdo declaradas em seu gabarito, e o ranking passa a operar sobre elas.

A distinção não é cosmética. Com 5 documentos e um único relevante por consulta, R@5 vale 1,000 por aritmética --- recuperar cinco de cinco sempre encontra o alvo --- e a linha de base aleatória de P@1 é 0,200. Com 19 seções, a linha de base cai para 0,053 e a métrica volta a discriminar. O gabarito de seção segue as mesmas regras anti-circularidade: cada seção recebe identificador autoral estável e uma âncora textual que deve existir no documento-fonte, de modo que o rótulo dependa do texto e nunca da saída do extrator estrutural.

Os braços densos são avaliados apenas na granularidade de documento. O artefato de embeddings contém um vetor por documento, e produzir vetores por seção exigiria chamada de rede a um serviço de inferência --- incompatível com a hermeticidade exigida do harness. A ausência é declarada, não silenciosa: o harness a anuncia em cada execução.

\textbf{Braços comparados.} Seis estratégias de recuperação foram avaliadas sobre o mesmo corpus, em delineamento pareado por consulta:

\begin{itemize}
  \item \textbf{A — relacional}: o scorer de produção do ThingIndex (casamento de substring sobre as estruturas relacionais, sem IDF), reproduzido fielmente, defeitos inclusive — o experimento mede o sistema como ele é, não uma versão idealizada;
  \item \textbf{B — BM25}: a baseline lexical clássica~\cite{robertson2009bm25}, com $k_1{=}1{,}2$ e $b{=}0{,}75$;
  \item \textbf{C — denso (exato)}: embeddings do modelo \texttt{multilingual-e5-base}~\cite{wang2022e5} (768 dimensões), com varredura exata por similaridade de cosseno;
  \item \textbf{C′ — denso-IVF}: os mesmos embeddings sob índice aproximado IVF-Flat, reimplementado com os nomes e valores padrão do pgvector (\texttt{lists}${=}\sqrt{N}$, \texttt{probes}${=}1$), que é o regime em que a busca densa efetivamente escala;
  \item \textbf{D — RRF}: fusão dos braços A e B por Reciprocal Rank Fusion~\cite{cormack2009rrf}, com $k{=}60$;
  \item \textbf{E — RRF denso}: fusão de B e C, o híbrido léxico-denso do estado da prática.
\end{itemize}

Para preservar a hermeticidade do harness (execução em CI sem GPU, sem rede e sem dependência da versão local de bibliotecas de inferência), os embeddings são computados uma única vez e versionados como artefato. Dois guardas independentes, baseados em hash SHA-256 do texto submetido ao modelo, impedem que o artefato dessincronize silenciosamente do corpus: um teste de CI falha o build e o próprio harness aborta a medição antes de emitir qualquer número.

\textbf{Métricas.} Foi adotado o painel padrão da área de RI~\cite{manning2008ir}: P@1, P@3, R@5, MRR e nDCG@10 com ganho exponencial~\cite{jarvelin2002cumulated}. Cada implementação de métrica possui teste automatizado cujo valor esperado foi calculado manualmente a partir da definição publicada — nunca a partir da saída da própria implementação.

\textbf{Análise estatística.} As comparações entre braços usam o teste pareado de postos sinalizados de Wilcoxon~\cite{wilcoxon1945individual}, adequado a métricas limitadas ao intervalo $[0,1]$, com correção de Bonferroni para as 75 comparações realizadas. O tamanho de efeito é reportado por $\delta$ de Cliff, com os limiares de interpretação de Romano et al.~\cite{romano2006appropriate}. Os intervalos de confiança de 95\% são obtidos por bootstrap sobre as consultas (10.000 reamostragens, semente fixa). A implementação do teste de Wilcoxon foi validada contra a biblioteca de referência \texttt{scipy}, com divergência máxima de $1{,}4\times10^{-16}$.

\subsection{Desempenho de Recuperação}

A Tabela~\ref{tab:retrieval-summary} apresenta o desempenho dos seis braços nas consultas positivas.

\input{tables/retrieval-summary}

Os braços de fusão e o BM25 obtiveram os melhores valores agregados (nDCG@10 entre 0,890 e 0,900), seguidos do relacional (0,869), do denso exato (0,844) e do denso aproximado (0,797). O resultado central para a pergunta de pesquisa é de não-inferioridade: a recuperação relacional não foi inferior à densa neste corpus, com intervalos de confiança amplamente sobrepostos em todas as métricas.

A Tabela~\ref{tab:retrieval-variants} decompõe a P@1 por variante de consulta, medindo a dependência de casamento lexical de cada braço.

\input{tables/retrieval-variants}

Os braços lexicais acertam 100\% quando a consulta reutiliza os termos do alvo e caem para 57--71\% quando a mesma necessidade de informação é parafraseada. A queda é maior no relacional (0,429) do que no BM25 (0,286), o que é coerente com a ausência de ponderação IDF no scorer de produção~\cite{salton1989automatic}. O braço denso inverte o padrão: sua precisão praticamente não depende de a consulta reutilizar as palavras do documento (queda de 0,029), embora seu teto na variante literal seja mais baixo (0,600). Os dois paradigmas falham, portanto, por razões diferentes — o que constitui a justificativa empírica dos braços híbridos.

\subsection{Efeito da Unidade de Recuperação}
\label{sec:granularidade}

Repetindo a medição sobre as 19 seções em vez dos 5 documentos, obtêm-se os resultados da Tabela~\ref{tab:retrieval-summary-section}.

\input{tables/retrieval-summary-section}

A comparação entre as duas granularidades revela um efeito de saturação que a avaliação por documento ocultava. Em nível de documento, R@5 vale exatamente 1,000 para \emph{todos} os braços lexicais, o que não é evidência de qualidade: com cinco documentos e profundidade de corte cinco, a métrica não tem como discriminar. Em nível de seção, os mesmos braços obtêm 0,719 e 0,812 --- valores que finalmente os separam.

O mesmo ocorre com a distância entre os braços. A diferença de P@1 entre o relacional e o BM25 é de 0,062 em nível de documento (0,750 contra 0,812) e de 0,188 em nível de seção (0,562 contra 0,750): triplica quando a unidade de recuperação corresponde à unidade de proveniência que o sistema oferece.

A Tabela~\ref{tab:retrieval-variants-section} decompõe esse resultado por variante de consulta.

\input{tables/retrieval-variants-section}

O achado mais consequente está na dependência de casamento lexical. A queda de P@1 de consultas literais para parafraseadas, no braço relacional, passa de $-0{,}429$ em nível de documento para $-0{,}857$ em nível de seção: o \emph{scorer} de produção acerta a totalidade das consultas que reutilizam termos do alvo e apenas 14\% daquelas que expressam a mesma necessidade com outras palavras. A granularidade grossa ocultava metade da dependência lexical --- exatamente a limitação que este trabalho identifica como central na abordagem relacional.

Esse resultado tem consequência direta para a leitura das seções anteriores: os números de não-inferioridade reportados em nível de documento devem ser interpretados como \emph{limite superior} do desempenho relacional, não como estimativa não enviesada. A avaliação em unidade grossa é sistematicamente mais generosa com o casamento lexical, porque um documento inteiro oferece muito mais oportunidades de correspondência de termo do que a seção que efetivamente responde à pergunta.

\subsection{Reconhecimento da Ausência de Resposta}

A Tabela~\ref{tab:retrieval-negatives} apresenta o comportamento dos braços nos dois casos negativos, em que a resposta correta é não recuperar nada.

\input{tables/retrieval-negatives}

Nenhum braço reconhece a ausência de resposta: a taxa de falso positivo é 1,000 nos seis. Nos braços lexicais, a causa é qualquer termo em comum gerar score positivo. No braço denso, a causa é estrutural: a análise de separabilidade das similaridades de cosseno mostrou que as distribuições de pares relevantes ($n{=}18$, entre 0,7852 e 0,8810) e irrelevantes ($n{=}62$, entre 0,7626 e 0,8483) se sobrepõem em quase toda a extensão, de modo que nenhum limiar preserva os relevantes e corta os irrelevantes. Esse comportamento é consistente com a anisotropia dos espaços de embeddings contextuais~\cite{ethayarajh2019contextual}: a similaridade de cosseno é informativa como ordem, não como magnitude. Por essa razão, o piso de similaridade inicialmente previsto para o braço denso foi removido do protocolo — mantê-lo penalizaria a baseline concorrente sem fundamento empírico.

\subsection{Significância Estatística}

Nenhuma das 75 comparações pareadas atingiu significância após a correção de Bonferroni (todos os $p$ corrigidos iguais a 1,000), e quase todos os tamanhos de efeito são desprezíveis pelos limiares de Romano et al.~\cite{romano2006appropriate}. A tabela completa das comparações encontra-se no Apêndice B. Como discutido na Seção de Discussão, esse resultado deve ser lido como limitação de poder amostral — com 16 consultas positivas, o teste de Wilcoxon não tem sensibilidade para detectar efeitos da magnitude observada — e não como evidência de equivalência entre os braços.

\subsection{Determinismo do Ranking}

A propriedade central reivindicada pela abordagem relacional é o determinismo da recuperação. Para medi-la, cada braço foi executado repetidamente e os rankings completos (identificadores e scores) foram comparados por hash SHA-256. Os braços A, B e D produziram rankings idênticos em 100\% das consultas ao longo de 30 execuções. A Tabela~\ref{tab:dense-determinism} apresenta o resultado para os braços densos, variando fatores que não são propriedades da consulta.

\input{tables/dense-determinism}

A varredura densa exata também é 100\% estável — o que a torna o controle do experimento: a instabilidade observada no IVF não provém do modelo de embeddings nem da aritmética de ponto flutuante, mas do próprio índice aproximado. Sob IVF-Flat, variar apenas a semente do $k$-means produziu até 4 rankings distintos para a mesma pergunta, com perda média de 48,4\% de recall em relação à varredura exata. Variar apenas a ordem de inserção dos documentos, com semente fixa, produziu o mesmo efeito — e esse é o fator de maior consequência prática, porque ordem de inserção não é um parâmetro de configuração: é histórico de uso. Dois usuários que inseriram os mesmos documentos em ordens diferentes podem receber respostas diferentes para a mesma pergunta, sem que nenhum dos dois tenha como perceber.

\subsection{Auditabilidade do Índice}
\label{sec:auditabilidade}

Auditabilidade é a propriedade central reivindicada por este trabalho, mas o termo só se torna verificável quando associado a uma medida. Definem-se aqui duas métricas, calculadas sobre o índice persistido:

\begin{itemize}
  \item \textbf{TPC (taxa de proveniência completa)}: fração de itens indexados cuja origem é integralmente resolvível --- documento, página, posição sequencial e conteúdo, todos presentes. O critério é deliberadamente rígido: saber que uma informação veio do documento 7 sem saber de qual página não permite apontar o trecho no original, e apontar o trecho é precisamente o que auditoria significa.
  \item \textbf{TCV (taxa de citação verificável)}: fração de trechos citados que existem literalmente na fonte apontada. Separa ``citou'' de ``citou corretamente'' --- um sistema pode indicar o documento certo e ainda assim exibir um trecho inexistente ali, modo de falha particularmente grave em um sistema que se apresenta como auditável.
\end{itemize}

\input{tables/provenance}

O resultado agregado (TPC global de 0,329) seria enganoso se reportado isoladamente, e por isso a Tabela~\ref{tab:provenance} é apresentada por tipo estrutural. A leitura correta é bimodal: a proveniência é \emph{completa} (1,000) em parágrafos, páginas e nós de sumário, e \emph{inexistente} (0,000) em chunks e metadados. A causa é única e foi confirmada por teste --- as 104 incompletudes são, todas, ausência de número de página; não há um único caso de identificador de documento ausente, posição inválida ou conteúdo vazio.

A origem do defeito está no próprio esquema: as entidades \texttt{TableSection}, \texttt{MetadataPair} e \texttt{DocumentChunk} não possuem coluna de página. Para o chunk, a proveniência é indireta --- resolve-se através dos parágrafos de origem, e não diretamente ---, o que a métrica contabiliza como incompleta por exigir resolução em um único passo. Trata-se de limitação de modelagem, endereçável pela adição da coluna, e não de falha do paradigma relacional: onde o esquema registra a coordenada, a recuperação a devolve sem exceção.

A TCV de 1,000 sobre 20 citações verificadas indica que a fragmentação em tabelas tipadas preserva integralmente o conteúdo textual: todo trecho recuperado existe literalmente no documento de origem. Esse resultado é o complemento necessário da TPC --- de nada adiantaria localizar a página se o texto exibido divergisse do original.

Cabe registrar o contraste com a alternativa densa, em que a unidade de proveniência é o \emph{chunk} de embedding, não a posição no documento: recuperar o vetor mais próximo não produz, por construção, a coordenada de origem. A TPC não é aplicável a esse braço --- não por deficiência de implementação, mas porque a informação necessária não existe na representação.

\subsection{Limiar de Corte e Reconhecimento de Ausência}
\label{sec:limiar}

A taxa de falso positivo de 1,000 registrada na Seção anterior é a limitação mais grave medida neste trabalho: os braços sempre devolvem algo, inclusive quando a resposta correta é o silêncio. Resta perguntar se essa limitação é intrínseca ou corrigível --- em outras palavras, se existe um score a partir do qual uma correspondência pode ser considerada real.

Para responder, coletaram-se as distribuições de score de itens relevantes e irrelevantes e buscou-se o corte que maximiza o índice de Youden ($J = \text{sensibilidade} + \text{especificidade} - 1$). Usou-se $J$, e não acurácia, porque as classes são fortemente desbalanceadas: com aproximadamente dez vezes mais itens irrelevantes que relevantes, um classificador que rejeitasse tudo teria acurácia alta e utilidade nula. A Tabela~\ref{tab:threshold} apresenta o diagnóstico.

\input{tables/threshold}

Os dois braços lexicais admitem um corte que reduz o falso positivo pela metade, mas por margens muito distintas. No BM25 a sobreposição das distribuições é de 0,980 --- quase todo item irrelevante pontua acima do pior item relevante --- e ainda assim o limiar alcança $J = 0{,}673$, porque a densidade dos scores relevantes se concentra na cauda superior. No braço relacional a sobreposição é menor (0,782), mas o poder discriminativo do corte também é ($J = 0{,}288$): o \emph{scorer} por substring produz scores inteiros de baixa granularidade, o que limita a resolução de qualquer limiar.

Duas ressalvas são obrigatórias na leitura desses números. A primeira: o corpus contém apenas \textbf{duas} consultas negativas, de modo que a redução observada de 1,000 para 0,500 corresponde a um acerto em dois casos. Trata-se de indício de viabilidade, não de demonstração de eficácia --- a estimativa da taxa com esse tamanho amostral tem intervalo de confiança que abrange praticamente todo o domínio. A segunda: o limiar foi selecionado sobre o mesmo conjunto em que é avaliado, o que produz estimativa otimista por construção; uma avaliação honesta exigiria conjuntos de ajuste e de teste separados, inviável com 18 consultas.

O que a medição sustenta, portanto, é uma afirmação modesta e verificável: \emph{a incapacidade de reconhecer ausência de resposta não é intrínseca aos braços lexicais}. Existe sinal nos scores, e ele é explorável. Transformar esse indício em mecanismo confiável requer corpus com número substancial de casos negativos e separação entre ajuste e teste --- trabalho de anotação, não de infraestrutura.

% =============================================================================
% DISCUSSAO
% =============================================================================
\section{Discussão}

Os resultados sustentam uma resposta matizada à pergunta de pesquisa formulada na Introdução, e cada achado merece leitura à luz de suas condições de validade.

\textbf{A queda literal$\rightarrow$paráfrase quantifica o que métricas ingênuas escondem.} Uma versão anterior do benchmark do projeto reportava 100\% de acerto em todos os indicadores — resultado que não evidenciava qualidade de recuperação, mas incapacidade da métrica de discriminar: o gabarito era circular e as consultas copiavam termos dos trechos-alvo. O protocolo atual, com variantes de consulta e regras anti-circularidade travadas por teste, expõe o que aqueles números ocultavam: a precisão dos braços lexicais é função direta do reúso vocabular da consulta. Esse achado reposiciona a limitação clássica do casamento léxico~\cite{manning2008ir} de argumento teórico para medida concreta sobre o sistema real.

\textbf{Relacional e denso falham por razões diferentes, e isso importa mais do que qual pontua mais alto.} O braço denso é o único cuja precisão independe do vocabulário da consulta, mas seu teto é mais baixo e sua variante aproximada degrada tanto a qualidade quanto a estabilidade. O braço relacional tem teto máximo no regime literal e a maior queda sob paráfrase. Como as forças são complementares, a fusão por RRF obtém os melhores agregados sem exigir nenhum componente novo de infraestrutura — um caminho de evolução para o ThingIndex que preserva a camada relacional como base auditável.

\textbf{A não-inferioridade observada é evidência favorável, não demonstração.} O nDCG@10 de 0,869 do braço relacional contra 0,844 do denso exato é exatamente a forma da hipótese central do trabalho — a de que uma baseline relacional pode sustentar cenários de inteligência documental usualmente atribuídos a soluções vetoriais. Contudo, nenhuma diferença entre braços é estatisticamente significativa, e a leitura honesta é de limitação de poder: com 16 consultas positivas, o experimento não distingue efeitos reais de ruído amostral. A infraestrutura de avaliação está pronta e testada; o que falta para transformar evidência em demonstração é anotação humana em escala (aproximadamente 150 consultas sobre pelo menos 30 documentos).

\textbf{Nenhum paradigma sabe dizer ``não encontrei'', mas a limitação não é intrínseca.} A taxa de falso positivo de 1,000 nos seis braços é uma limitação real do estado da prática, raramente medida. A investigação da Seção~\ref{sec:limiar} mostra que, nos braços lexicais, existe sinal explorável nos scores: um limiar de corte reduz o falso positivo pela metade, com índice de Youden de 0,673 no BM25. A ressalva é de tamanho amostral --- duas consultas negativas não demonstram eficácia, e o limiar foi selecionado no mesmo conjunto em que é avaliado. No braço denso, a sobreposição das distribuições de similaridade indica que não se trata de ajuste de parâmetro, mas de geometria do espaço de embeddings~\cite{ethayarajh2019contextual}. Para um sistema cuja tese é auditabilidade, responder com confiança quando não há resposta é o modo de falha mais grave --- e nenhuma das abordagens avaliadas, incluindo a proposta, o resolve hoje. A diferença que a medição estabelece é que, para a abordagem relacional, o caminho de correção existe e é conhecido.

\textbf{O preço do índice aproximado é o resultado que sustenta a contribuição.} A afirmação que este trabalho pode fazer após a medição é mais específica — e mais defensável — do que a formulação original da proposta: não ``a recuperação relacional é determinística e o RAG vetorial não é'', que é falsa para a varredura exata, mas: \emph{a busca densa exata é determinística, porém custa $O(N)$ por consulta; o índice aproximado escala, porém torna o resultado dependente de decisões de indexação que o usuário não controla nem observa; a recuperação relacional não paga nenhum dos dois preços.} A instabilidade por ordem de inserção é a manifestação mais séria desse custo, por ser função do histórico de uso e invisível ao usuário — a antítese da proveniência que fundamenta este trabalho~\cite{buneman2001provenance}.

\subsection{Ameaças à Validade}

\textbf{Validade de conclusão.} O corpus C1 possui 16 consultas positivas, insuficientes para que o teste de Wilcoxon detecte efeitos da magnitude observada. Por essa razão, nenhuma comparação de qualidade entre braços é apresentada como conclusiva. O script de análise devolve explicitamente $p{=}1{,}0$ quando o número de pares com diferença é inferior ao mínimo do teste, em vez de fabricar um número de aparência precisa.

\textbf{Validade interna.} O gabarito de relevância foi anotado por um único autor, sem dupla anotação nem coeficiente de concordância (Cohen's $\kappa$). As regras anti-circularidade travadas por teste mitigam o viés de construção das consultas, mas não substituem anotação independente.

\textbf{Validade externa.} Com 5 documentos, o índice IVF opera em regime degenerado (\texttt{lists}${=}2$), o que amplifica a perda de recall. Essa objeção foi medida em vez de argumentada: a Tabela~\ref{tab:ivf-escala} reporta o mesmo protocolo sobre corpora sintéticos de 768 dimensões em escalas maiores.

\begin{table}[htb]
\centering
\caption{Instabilidade do IVF-Flat em função da escala do corpus (corpora sintéticos, 768 dimensões, 8 clusters, 20 consultas, 10 sementes por consulta).}
\label{tab:ivf-escala}
\begin{tabular}{cccc}
\hline
$N$ & \texttt{lists} & Recall@10 do IVF vs.\ exato & Consultas instáveis entre sementes \\
\hline
5 & 2 & 31,2\% & 20/20 \\
50 & 7 & 69,0\% & 20/20 \\
200 & 14 & 81,3\% & 20/20 \\
1000 & 32 & 50,5\% & 20/20 \\
\hline
\end{tabular}
\end{table}

A leitura tem duas partes. A objeção procede quanto à \emph{magnitude} da perda de recall: o regime de corpus minúsculo de fato a amplifica, e o número de 48,4\% reportado na Seção de Resultados deve ser lido com essa ressalva. A objeção não procede, porém, quanto à \emph{instabilidade}, que é a afirmação que o trabalho efetivamente faz: em todas as escalas medidas, incluindo $N{=}1000$, 100\% das consultas mudam de ranking quando apenas a semente muda. A instabilidade é propriedade do algoritmo, não artefato do corpus. Ambas as conclusões estão travadas por testes automatizados.

\textbf{Validade de construto.} O braço A reproduz o scorer de produção do ThingIndex, defeitos inclusive — decisão deliberada para medir o sistema real, não uma versão idealizada. O braço C′ é uma reimplementação do IVF-Flat com os nomes e valores padrão do pgvector, não o pgvector em si, e o índice HNSW (padrão do pgvector a partir da versão 0.5.0), que possui fontes próprias de não-determinismo, não foi medido. A incerteza reportada é de amostragem sobre o conjunto de consultas, não de variabilidade de execução: o pipeline é determinístico, e repetir a mesma execução não gera variância.

\FloatBarrier


% =============================================================================
% 8. ARQUITETURA E MODELAGEM DO SISTEMA
% =============================================================================

% ALDO: ESSA SEÇÃO FOI PARA A SEÇÃO ANTERIOR

% \section{Arquitetura e Modelagem do Sistema}

% Esta seção descreve a estrutura proposta para o ThingIndex como um artefato a ser implementado. Cada decisão apresentada aqui orientará o desenvolvimento do código fonte do monorepo.

% \subsection{Visão Geral da Arquitetura}

% O sistema será organizado em quatro camadas principais: interface de usuário, API, persistência e infraestrutura auxiliar (object storage, filas), cujas tecnologias são detalhadas na Seção~3.5. A Figura~\ref{fig:arquitetura} ilustrará essa composição. Richards~\cite{richards2015patterns} classifica essa organização como uma variação do padrão \textit{Layered Architecture}, no qual cada camada possui responsabilidade bem definida e a dependência flui em uma única direção.

% A API será a peça central. Ela exporá rotas HTTP para autenticação, gerenciamento de coleções, upload de documentos, configuração de provedores, recuperação de contexto e chat. As rotas serão organizadas em \textit{handlers} finos, que apenas validarão entrada, autenticarão e serializarão respostas. A lógica de negócio residirá em \textit{use cases}, que orquestrarão chamadas a \textit{repositories} e serviços auxiliares.

% A Figura~\ref{fig:visao-geral-sistema} apresenta a visão geral da arquitetura do ThingIndex, consolidando a integração entre frontend, API, banco de dados, filas e provedores de IA.

% \begin{figure}[!ht]
%   \centering
%   \includegraphics[width=1\textwidth]{anexos/diagramas/visao-geral-sistema.png}
%   \caption{Visão geral da arquitetura do sistema.}
%   \label{fig:visao-geral-sistema}
% \end{figure}

% A Figura~\ref{fig:casos-uso} apresenta o diagrama de casos de uso do ThingIndex, com os atores e as funcionalidades disponíveis no sistema.

% \begin{figure}[!ht]
%   \centering
%   \includegraphics[width=1\textwidth]{anexos/diagramas/casos-de-uso.png}
%   \caption{Diagrama de casos de uso do ThingIndex.}
%   \label{fig:casos-uso}
% \end{figure}

% O pipeline assíncrono será implementado por meio do padrão \textit{outbox}: quando um documento for enviado, um evento será gravado na mesma transação do banco que persistir o documento. Um processo \textit{dispatcher} realizará polling na tabela de outbox e publicará os eventos no RabbitMQ. Essa estratégia reduz o risco de perda de mensagens: se a API cair antes de publicar no broker, o evento ainda estará no banco e será reenviado na próxima rodada do dispatcher. Richardson~\cite{richardson2018microservices} descreve esse padrão como a solução mais robusta para o problema de dualidade de escrita (banco + broker) em arquiteturas baseadas em microsserviços.

% O worker \textit{indexer} consumirá a fila, processará o documento e persistirá as estruturas extraídas. Se o processamento falhar, o evento será reenviado para uma fila de \textit{retry} com atraso configurável. Após esgotar as tentativas, ele será movido para uma \textit{dead-letter queue} (DLQ), onde poderá ser inspecionado manualmente. Hohpe e Woolf~\cite{hohpe2003eip} classificam esses mecanismos como \textit{Dead Letter Channel} e \textit{Resequencer}, padrões consolidados de integração empresarial que permitem entrega confiável mesmo em cenários de falha parcial.

% \begin{figure}[!ht]
%   \centering
%   \includegraphics[width=0.65\textwidth]{anexos/diagramas/arquitetura.png}
%   \caption{Arquitetura runtime do ThingIndex, com separação entre camada web, API, persistência e processamento assíncrono.}
%   \label{fig:arquitetura}
% \end{figure}

% \FloatBarrier

% \subsection{Modelagem de Dados}

% O modelo de dados será projetado para que o contexto de uma coleção seja reconstruído por consultas SQL determinísticas. O esquema completo contempla 21 tabelas organizadas em três grupos lógicos: núcleo, estruturas extraídas e infraestrutura, conforme descrito a seguir e disponível na íntegra no Apêndice A. A Figura~\ref{fig:schema-core} apresenta o diagrama entidade-relacionamento do grupo principal (núcleo), com as entidades fundamentais do sistema.

% A Figura~\ref{fig:schema-core} apresenta o diagrama entidade-relacionamento do núcleo do sistema, composto pelas entidades fundamentais: usuários, categorias, coleções e documentos. Esse núcleo estabelece as relações básicas sobre as quais todo o sistema se apoia. Um \texttt{user} pode possuir múltiplas \texttt{categories} e \texttt{collections}, e cada \texttt{collection} agrupa diversos \texttt{documents}. A \texttt{collection} pode opcionalmente pertencer a uma \texttt{category}, permitindo organização hierárquica. Cada \texttt{document} armazena metadados como caminho, tipo MIME, tamanho e status de processamento.

% \begin{figure}[!ht]
%   \centering
%   \includegraphics[width=0.85\textwidth]{anexos/database/diagrama-database-principal.png}
%   \caption{Diagrama entidade-relacionamento do núcleo: usuários, categorias, coleções e documentos.}
%   \label{fig:schema-core}
% \end{figure}

% A partir do documento, as estruturas extraídas são fragmentadas em seis tipos de tabelas, cada uma responsável por um aspecto específico do conteúdo:
% \begin{itemize}
%   \item páginas: conteúdo textual de cada página, com resumo e estimativa de tokens;
%   \item parágrafos: blocos textuais com posição sequencial e número da página;
%   \item tabelas: conteúdo tabular com cabeçalhos e linhas;
%   \item metadados: pares chave-valor associados ao documento;
%   \item unidades de recuperação: trechos com tipo, título e vínculo aos parágrafos de origem;
%   \item hierarquia de seções: estrutura de tópicos com níveis e referência ao pai.
% \end{itemize}

% Além das entidades de domínio, o esquema inclui tabelas de infraestrutura: eventos de outbox para o padrão de mensageria, tentativas de processamento para rastreabilidade de falhas, eventos de auditoria para trilha de operações, configurações de provedores com credenciais criptografadas, conversas e mensagens de chat, além de dados de precificação para rastreio de custos.

% A fragmentação intencional do conteúdo em tabelas separadas evita colunas polimórficas ou JSONB monolítico, garantindo que cada tipo de dado tenha seu próprio esquema bem definido. Índices compostos e chaves estrangeiras preservam a ordenação original e a integridade referencial entre as tabelas.

% A Figura~\ref{fig:classes} complementa essa visão com o diagrama de classes, que organiza as 14 entidades nos três grupos lógicos. O diagrama de classes permite visualizar não apenas as tabelas, mas também os métodos associados a cada entidade e os relacionamentos com suas cardinalidades, oferecendo uma perspectiva orientada a objetos do modelo de dados.

% \begin{figure}[!ht]
%   \centering
%   \includegraphics[width=1\textwidth]{anexos/diagramas/classes.png}
%   \caption{Diagrama de classes do ThingIndex, com 16 classes organizadas em núcleo (azul), estruturas (verde) e infraestrutura (roxo).}
%   \label{fig:classes}
% \end{figure}

% \FloatBarrier

% \subsection{Fluxo de Processamento}

% O fluxo de processamento de um documento no ThingIndex pode ser dividido em três etapas principais, cada uma com responsabilidades bem definidas dentro da arquitetura do sistema.

% A primeira etapa compreende o upload e a validação do documento. Quando o usuário envia um arquivo PDF, a API verifica o tipo MIME e a existência da coleção de destino. O arquivo é armazenado em object storage e registrado no banco de dados com status pendente. Nesse momento, um evento de indexação é inserido em uma tabela de outbox na mesma transação, garantindo que nenhum documento fique sem processamento mesmo em caso de falha da API.

% A segunda etapa é a indexação assíncrona, executada por um worker dedicado. O dispatcher lê os eventos da tabela de outbox e os publica no RabbitMQ. O worker indexer consome a mensagem e realiza o parsing do PDF, extraindo parágrafos, tabelas e metadados. Opcionalmente, um LLM pode enriquecer o conteúdo extraído. Todas as estruturas são persistidas no banco relacional, e a telemetria do processamento é registrada para auditoria. Em caso de falha, o evento é reenviado para uma fila de retry e, após esgotar as tentativas, movido para uma dead-letter queue para inspeção manual.

% A terceira etapa é a recuperação de contexto, detalhada na Seção~10.4. Quando uma consulta é realizada, o sistema executa joins SQL sobre as tabelas tipadas, monta um JSON hierárquico com todo o contexto da coleção e disponibiliza o resultado para o chat com citação de fontes.

% \subsubsection{Upload e Indexação}

% O usuário autenticado envia um arquivo PDF. A API valida o tipo MIME e a existência da coleção de destino. O arquivo é enviado ao object storage e registrado no PostgreSQL com status pendente. Um evento de indexação é inserido na tabela de outbox na mesma transação.

% O dispatcher publica o evento no RabbitMQ. O worker indexer consome a mensagem e executa: leitura do arquivo, parsing, extração de parágrafos e tabelas, enriquecimento opcional por LLM, persistência das estruturas e registro de telemetria. Em caso de falha, o evento segue para retry ou dead-letter queue.

% A Figura~\ref{fig:fluxo} ilustra esse fluxo completo.

% \begin{figure}[!ht]
%   \centering
%   \includegraphics[width=1\textwidth]{anexos/diagramas/fluxo-processamento.png}
%   \caption{Fluxo completo: upload, enfileiramento, indexação e recuperação de contexto.}
%   \label{fig:fluxo}
% \end{figure}

% \subsubsection{Interação entre Componentes}

% Para complementar a visão do fluxo de processamento, o diagrama de sequência apresentado a seguir mostra a ordem temporal das chamadas entre os componentes do sistema. Nele é possível visualizar como o frontend se comunica com a API, como a API interage com o banco de dados e o storage, como o evento de indexação é propagado para o RabbitMQ e consumido pelo worker, e finalmente como o contexto é recuperado e exibido ao usuário.

% A Figura~\ref{fig:sequencia} detalha essa comunicação durante os três fluxos principais, abrangendo desde o upload do documento até a resposta do chat com citação de fontes.

% \begin{figure}[!ht]
%   \centering
%   \includegraphics[width=0.85\textwidth]{anexos/diagramas/sequencia.png}
%   \caption{Diagrama de sequência do ThingIndex, ilustrando a comunicação entre frontend, API, storage, banco, fila e worker.}
%   \label{fig:sequencia}
% \end{figure}

% \FloatBarrier

% =============================================================================
% 10. DESENVOLVIMENTO DO SOFTWARE
% =============================================================================

% ALDO: No TCC I, não vejo necessidade de discutir o desenvolvimento dessa forma, pois certamente irá mudar. Isso deve estar no TCC II.

% \section{Desenvolvimento do Software}

% O desenvolvimento seguirá uma ordem que reflete dependências funcionais: sem autenticação, não há isolamento de dados; sem coleção, não há organização; sem pipeline, não há indexação.

% \subsection{Módulo de Autenticação e Usuários}

% O módulo de autenticação implementará cadastro, login, reset de senha e controle de sessão. A senha será armazenada como hash bcrypt. O login retornará um JWT assinado com expiração. O reset funcionará por token único hasheado enviado por e-mail; ao ser consumido, invalidará tokens anteriores.

% As rotas de autenticação terão rate limiting e sanitização de erros. O backend não exporá detalhes internos nas respostas HTTP.

% \subsection{Módulo de Coleções e Documentos}

% O usuário criará categorias e coleções para agrupar documentos. Uma coleção será o vínculo lógico obrigatório: todo documento deverá pertencer a uma. O upload validará o tipo MIME, persistirá o arquivo em object storage e registrará o documento com referência ao caminho remoto.

% A listagem de documentos será sempre escopada pelo usuário. A recuperação de conteúdo bruto utilizará URL temporária do provedor de storage, evitando proxy de bytes pela API.

% \subsection{Módulo de Indexação e Processamento}

% Este módulo será o núcleo operacional. O worker \textit{indexer} consumirá eventos do RabbitMQ e processará documentos em etapas: parsing, extração estrutural, enriquecimento opcional por LLM e persistência de telemetria.

% O enriquecimento por LLM será configurável por usuário. O sistema aceitará alguns dos principais provedores disponíveis no mercado. A telemetria de cada chamada, como tokens, modelo, duração e custo estimado, será registrada e exposta em um painel agregado por usuário.

% A resiliência será garantida por retry com atraso, fila de DLQ e separação entre fila principal e fila de retry. O worker não bloqueará a API; falhas no processamento não impedirão novos uploads.

% A Figura~\ref{fig:visao-indexacao} apresenta o fluxo abstrato de indexação, desde a entrada do documento até a persistência das estruturas extraídas no banco relacional.

% \begin{figure}[!ht]
%   \centering
%   \includegraphics[width=0.7\textwidth]{anexos/diagramas/visao-indexacao.png}
%   \caption{Visão geral do fluxo de indexação.}
%   \label{fig:visao-indexacao}
% \end{figure}

% \subsection{Módulo de Consulta e Recuperação de Contexto}

% A recuperação de contexto será a operação que validará a hipótese central do projeto. Dado o ID de uma coleção, o sistema reconstruirá todo o contexto em número constante de consultas:

% \begin{enumerate}
%   \item carregar coleção com autorização;
%   \item buscar todos os documentos da coleção;
%   \item buscar parágrafos, tabelas e metadados em lote;
%   \item agregar em memória.
% \end{enumerate}

% O resultado será um JSON hierárquico representando a coleção, seus documentos e todo o conteúdo extraído. O chat por coleção consumirá essa mesma estrutura: o contexto consolidado será enviado ao LLM junto com a pergunta do usuário, e a resposta será transmitida via SSE (Server-Sent Events).

% A Figura~\ref{fig:visao-recuperacao} ilustra como o contexto é recuperado: a partir da consulta, o sistema realiza joins SQL sobre as tabelas tipadas e monta a resposta com fontes citadas.

% \begin{figure}[!ht]
%   \centering
%   \includegraphics[width=0.5\textwidth]{anexos/diagramas/visao-recuperacao.png}
%   \caption{Visão geral do fluxo de recuperação de contexto.}
%   \label{fig:visao-recuperacao}
% \end{figure}

% Esse fluxo de recuperação encerra a descrição dos principais módulos de processamento do ThingIndex. A seguir, são apresentados os protótipos de tela que ilustram a interface proposta para o sistema.

% \subsection{Interface do Usuário}

% O frontend será dividido em módulos de prototipação organizados em páginas, conforme descrito a seguir.

% \subsubsection{Landing Page}

% A landing page apresenta a proposta de valor do ThingIndex. A Figura~\ref{fig:ui-home} ilustra essa tela.

% \begin{figure}[!ht]
%   \centering
%   \includegraphics[width=0.95\textwidth]{anexos/telas/pagina-inicial.png}
%   \caption{Página inicial do ThingIndex.}
%   \label{fig:ui-home}
% \end{figure}

% \subsubsection{Dashboard}

% O dashboard operacional oferece gerenciamento de categorias e coleções, listagem de documentos com preview lateral e telemetria, upload por arrastar-e-soltar, painel de monitoramento agregado e tela de chat com histórico de conversas e fontes consultadas. A Figura~\ref{fig:ui-dashboard} ilustra essa tela.

% \begin{figure}[!ht]
%   \centering
%   \includegraphics[width=0.95\textwidth]{anexos/telas/dashboard.png}
%   \caption{Dashboard operacional.}
%   \label{fig:ui-dashboard}
% \end{figure}

% \subsubsection{Documentos}

% A tela de documentos permite visualizar e gerenciar os arquivos enviados, com informações de status de processamento e acesso ao conteúdo extraído.

% \subsubsection{Configurações}

% A tela de configurações permite ao usuário gerenciar provedores de IA e storage, com criptografia de credenciais no backend. O usuário poderá definir provedores distintos para indexação e chat.

% \subsection{Observabilidade e Avaliação da Qualidade da Indexação}

% Além da extração estrutural, o ThingIndex incorporará um mecanismo de observabilidade que permitirá avaliar, por documento e de forma agregada, a qualidade e o custo do processamento. Essa camada não será meramente auxiliar: ela constituirá uma resposta ao problema de opacidade que frequentemente acompanha pipelines de indexação assistida por IA, nos quais é difícil saber se o processamento foi bem-sucedido, quanto custou e onde falhou.

% \subsubsection{Telemetria por Documento}

% Cada vez que um documento for processado, o sistema registrará uma trilha de telemetria contendo:

% \begin{itemize}
%   \item Status e etapa: fase do pipeline (pendente, processando, indexado, falha);
%   \item Duração: tempo total do processamento;
%   \item Provedor e modelo: qual LLM foi utilizado, quando aplicável;
%   \item Requests LLM: quantidade de chamadas ao provedor externo;
%   \item Tokens: número de tokens de entrada e saída, ou estimativa heurística;
%   \item Custo estimado: valor calculado a partir dos preços por token do modelo;
%   \item Score heurístico de qualidade: métrica calculada a partir da completude estrutural do documento (proporção de parágrafos e tabelas extraídos, presença de outline coerente, ausência de erros de parsing).
% \end{itemize}

% Essa telemetria será persistida no mesmo banco relacional do domínio, vinculada ao documento e ao usuário, sendo consultável, auditável e exportável.

% \subsubsection{Benchmark Público de Indexação}

% O projeto manterá um benchmark versionado e publicamente acessível. Ele será construído sobre um corpus controlado embarcado no código fonte, garantindo reprodutibilidade: qualquer instância do ThingIndex poderá executar o mesmo conjunto de documentos e comparar resultados.

% O corpus validará cenários específicos: casos de árvore documental (verificam top-1 de nó e página), casos de coleção (validam top-1 de documento), verificações estruturais (outlines e contagens de tabelas) e sinais estruturais (profundidade do outline, cobertura, taxa de títulos inesperados).

% O benchmark será intencionalmente pequeno e determinístico. Seu propósito não será afirmar acurácia em corpus arbitrários, mas estabelecer uma baseline versionada sobre a qual mudanças no pipeline poderão ser comparadas antes e depois.

% \subsubsection{Painel de Monitoramento Agregado}

% O sistema exporá um resumo agregado por usuário, consolidando: totais de documentos por status, requests e tokens, custo total, duração média, score médio e detalhamento por modelo. O frontend exibirá esses dados em um painel operacional que destacará anomalias, como filas travadas, falhas recentes ou custo acima da média.

% \subsubsection{Limitações do Score e da Estimativa}

% O score heurístico não será ground truth absoluto: ele medirá completude estrutural observada pelo sistema, não correção semântica avaliada por humanos. A estimativa de custo e de tokens também será aproximada quando o provedor não retornar dados reais de uso. Esses valores deverão ser interpretados como indicadores operacionais, não como demonstrativos contábeis.

% \subsection{Testes de Código}

% A validação do código será conduzida em quatro níveis: unitário, integração, end-to-end e smoke.

% \subsubsection{Estratégia de Testes}

% Os testes unitários verificarão funções e componentes isolados. Os testes de integração exercitarão a API com banco de dados, storage e fila reais, garantindo que consultas, outbox e processamento operem em conjunto.

% Os testes e2e do backend simularão fluxos completos autenticados: registro, upload, indexação com retry, exclusão, documento órfão, configuração de provedor e chat com stream. Os smoke tests verificarão rapidamente se as rotas protegidas respondem corretamente.

% Testes opcionais com LLM real serão disparados por variáveis de ambiente e executarão uma rodada completa de indexação com um modelo configurado.

% \subsubsection{Cenários Avaliados}

% \begin{itemize}
%   \item Autenticação e autorização: cadastro, login, JWT, reset de senha com invalidação de tokens antigos, rate limiting, isolamento por usuário.
%   \item Upload e armazenamento: envio de PDF, validação de MIME, persistência em object storage, recuperação via URL temporária.
%   \item Indexação estrutural: processamento assíncrono, extração de parágrafos e tabelas, preservação da ordem, telemetria.
%   \item Resiliência: falha simulada no worker, retry automático, fila de descarte após esgotamento.
%   \item Recuperação de contexto: reconstrução de coleção, agregação, consistência dos dados.
%   \item Chat grounded: criação de conversa, stream, rastreabilidade de fontes.
%   \item Provedores: criptografia de credenciais, descoberta de modelos, teste de conectividade.
% \end{itemize}

% \FloatBarrier

\section{Conclusão}

Este artigo apresentou o ThingIndex, uma plataforma de inteligência documental baseada em modelagem relacional, da concepção à avaliação experimental. A pergunta de pesquisa formulada na Introdução — se é possível sustentar um sistema de inteligência documental com auditabilidade utilizando exclusivamente modelagem relacional, sem bancos vetoriais ou grafos como camada primária de recuperação — recebe, com base nos resultados, uma resposta afirmativa e delimitada: no corpus avaliado, a recuperação relacional não foi inferior à densa em qualidade de ranking (nDCG@10 de 0,869 contra 0,844, sem diferença estatisticamente significativa) e é a única abordagem, entre as seis avaliadas, que combina estabilidade total do resultado com custo de consulta que não cresce linearmente com uma varredura exaustiva de vetores: a busca densa exata é determinística mas custa $O(N)$ por consulta, e o índice aproximado escala mas torna o ranking dependente de fatores que o usuário não controla nem observa, como a semente do algoritmo de particionamento e a ordem de inserção dos documentos.

As contribuições do trabalho são quatro. Primeira, um artefato completo: o sistema especificado na Proposta foi implementado como monorepo executável, com pipeline assíncrono de indexação, recuperação por consultas SQL determinísticas, trilha de proveniência por chaves estrangeiras e testes automatizados, versionado e disponível para replicação e extensão mediante solicitação. Segunda, um protocolo de avaliação reproduzível com regras anti-circularidade travadas por teste, construído após a constatação de que o benchmark original do projeto — que reportava 100\% de acerto em todos os indicadores — media casamento lexical, e não recuperação. Terceira, a quantificação do custo de determinismo dos índices vetoriais aproximados, incluindo a medição da ameaça de escala em vez de sua mera discussão: a instabilidade do ranking sob variação de semente persiste em 100\% das consultas em todas as escalas medidas, até $N{=}1000$. Quarta, a evidência, inédita para este sistema, de que nenhum dos paradigmas avaliados — relacional, lexical, denso ou híbrido — reconhece a ausência de resposta, com causas distintas e mensuradas em cada caso.

O trabalho tem limitações claras, detalhadas nas Ameaças à Validade: o poder amostral do corpus (16 consultas positivas) impede conclusões comparativas definitivas, o gabarito foi anotado por um único autor e o índice HNSW não foi medido. Delas decorrem os trabalhos futuros prioritários: expandir o corpus para aproximadamente 150 consultas sobre pelo menos 30 documentos; realizar dupla anotação em ao menos 20\% do gabarito com Cohen's $\kappa$ reportado; adicionar um braço HNSW ao experimento de determinismo; e medir o caminho de documentos digitalizados (OCR). Como evolução do sistema, os resultados apontam um caminho concreto: a fusão por RRF dos sinais relacional e BM25 obteve os melhores agregados do experimento sem exigir nenhum componente novo de infraestrutura, preservando a camada relacional — e sua proveniência — como base auditável do sistema.

% \subsection{Contribuições Esperadas}

% A principal contribuição esperada do ThingIndex é demonstrar que é possível sustentar inteligência documental com modelagem relacional rigorosa, sem depender obrigatoriamente e diretamente de vetores ou grafos. Essa postura não nega a utilidade de técnicas de IA, mas defende que uma base determinística, testada e de custo previsível deve preceder extensões experimentais.

% Academicamente, o trabalho busca demonstrar que arquiteturas convencionais aplicadas com rigor podem sustentar cenários frequentemente atribuídos a soluções neurais. Do ponto de vista prático, o projeto oferecerá uma plataforma implantável sem infraestrutura especializada ou custos variáveis obrigatórios de API.

% A limitação principal esperada é a recuperação semântica: em domínios abertos e corpus massivos, embeddings provavelmente superarão joins hierárquicos em flexibilidade de matching. No entanto, para fluxos documentais estruturados (processos jurídicos, pesquisas acadêmicas, compliance), a rastreabilidade e a previsibilidade da baseline relacional poderão ser mais valiosas que a semântica genérica.

% \subsection{Conclusão}

% Este artigo apresentou a proposta do ThingIndex, uma plataforma de inteligência documental baseada em modelagem relacional, capaz de organizar, processar, indexar e recuperar contexto documental com auditabilidade e potencial de evolução. A partir da fundamentação teórica, da definição metodológica, do levantamento de requisitos, da modelagem arquitetural e do planejamento de implementação, foi possível consolidar a arquitetura do sistema e estabelecer as bases para seu desenvolvimento futuro.

% De modo geral, os resultados esperados indicam que a plataforma deverá atender ao objetivo principal do trabalho ao estabelecer uma baseline relacional concreta para fluxos documentais. Além disso, o projeto permitirá aplicar na prática conceitos de engenharia de software, arquitetura, persistência, processamento assíncrono e validação de sistemas. Assim, espera-se que o ThingIndex represente uma contribuição técnica e acadêmica relevante, ainda que existam pontos de evolução a serem tratados em etapas futuras.

% \subsection{Trabalhos Futuros}

% Como possibilidades de continuidade do projeto, destacam-se:

% \begin{itemize}
%   \item aprimorar a interface do usuário e a experiência de navegação;
%   \item expandir a cobertura de testes automatizados e de validação com usuários;
%   \item evoluir a observabilidade e o monitoramento operacional;
%   \item comparar a baseline relacional com abordagens vetoriais e orientadas a grafo;
%   \item incorporar novos tipos de documentos, processadores ou provedores externos;
%   \item realizar avaliação de acurácia com corpus jurídico e acadêmico independente.
% \end{itemize}

\subsection*{Declaração do uso de ferramentas de IA Generativa no processo de escrita}
No desenvolvimento deste manuscrito, os autores utilizaram o serviço ChatGPT, baseado no modelo de linguagem GPT-4, especificamente para fins de revisão textual. Após aplicação desta ferramenta, os autores revisaram e refinaram meticulosamente o conteúdo conforme necessário. É imperativo destacar que os autores assumem total responsabilidade pelo conteudo final da publicação.

% =============================================================================
% 9. REFERÊNCIAS
% =============================================================================
\bibliographystyle{sbc}
\bibliography{references}

% =============================================================================
% APÊNDICES
% =============================================================================
\section*{Apêndices e Anexos}

\subsection*{Apêndice A: Esquema Completo do Banco de Dados}

O esquema relacional completo do ThingIndex, contendo todas as 21 tabelas, tipos de dados, chaves primárias, chaves estrangeiras e índices, está disponível em formato DBML no repositório do projeto, no arquivo \texttt{docs/schema/thingindex.dbml}. O arquivo pode ser importado no \texttt{dbdiagram.io} para visualização interativa.

O arquivo é gerado a partir dos mesmos modelos que a aplicação submete à migração automática, e não redigido manualmente (\texttt{api/cmd/schema-dbml}). A escolha é deliberada: um apêndice escrito à mão descreve o esquema que o autor acredita ter, enquanto um apêndice derivado descreve o esquema que o sistema efetivamente cria. Um teste de regressão falha caso o arquivo versionado divirja dos modelos, de modo que a documentação não pode silenciosamente envelhecer em relação ao código.

\subsection*{Apêndice B: Comparações Pareadas Completas}

A Tabela~\ref{tab:retrieval-significance} apresenta as 75 comparações pareadas entre os seis braços do experimento (Wilcoxon com correção de Bonferroni e $\delta$ de Cliff), resumidas na Seção de Resultados. Como o apêndice A, a tabela é gerada por script (\texttt{scripts/analyze.py}) a partir da saída bruta do experimento.

\input{tables/retrieval-significance}

% \noindent\textit{Nota: este artigo utilizou ferramentas de IA (ChatGPT) exclusivamente para correção ortográfica e gramatical, conforme orientação do professor orientador. Todo o conteúdo técnico e conceitual foi escrito pelo autor.}

\end{document}
