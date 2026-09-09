# Corpus de demonstração: uma faculdade

Documentos públicos do Instituto Federal do Paraná (IFPR), capturados dos sites
oficiais da instituição em 2026-09-08. São atos normativos, editais e calendários
publicados abertamente pela administração pública federal, sem restrição de
acesso. Estão aqui só para demonstração e teste do ThingIndex; a fonte de cada um
é o site do IFPR.

Semeie numa stack de pé com `./scripts/seed-demo-faculdade.py`. O script cria o
usuário `demo-faculdade@thingindex.local`, envia os dez arquivos com categoria e
collection, espera a indexação e imprime, por documento, páginas, parágrafos e
parágrafos com bounding box. Falha se algum documento terminar `FAILED`, ficar sem
box, ou se alguma collection não tiver conexão entre documentos diferentes.

| Arquivo | Categoria | Collection | Páginas | Origem |
|---|---|---|---|---|
| `edital-56-2026-ps-2027-licenciatura.pdf` | Editais | Processo Seletivo 2027 | 24 | https://ifpr.edu.br/wp-content/uploads/2026/06/SEI_IFPR-4225605-Edital-Licenciatura.pdf |
| `edital-57-2026-ps-2027-tecnicos-subsequentes.pdf` | Editais | Processo Seletivo 2027 | 23 | https://ifpr.edu.br/wp-content/uploads/2026/06/SEI_IFPR-4225620-Edital-57.pdf |
| `regimento-geral-ifpr.pdf` | Normas | Regimento e resoluções | 27 | https://ifpr.edu.br/wp-content/uploads/2012/10/regimento-geral.pdf |
| `regimento-interno-comum-campi-res-08-2014.pdf` | Normas | Regimento e resoluções | 20 | https://ifpr.edu.br/wp-content/uploads/2014/10/Anexo-Res.-08.141.pdf |
| `ppc-sistemas-de-informacao-palmas-2019.pdf` | Cursos | Sistemas de Informação - Palmas | 213 | https://ifpr.edu.br/palmas/wp-content/uploads/sites/31/2023/06/2019-PPC-Sistemas-da-Informacao.pdf |
| `calendario-academico-palmas-2026.pdf` | Cursos | Sistemas de Informação - Palmas | 5 | https://ifpr.edu.br/palmas/wp-content/uploads/sites/31/2026/01/CALENDARIO-ACADEMICO-Palmas-2026-ENSINO-SUPERIOR-2026-1.pdf |
| `matriz-curricular-si-palmas-2023.pdf` | Cursos | Sistemas de Informação - Palmas | 3 | https://ifpr.edu.br/palmas/wp-content/uploads/sites/31/2023/06/Matriz_SistemasdeInformacao_2023-1.pdf |
| `ementas-si-palmas-ppc-2023.pdf` | Cursos | Sistemas de Informação - Palmas | 66 | https://ifpr.edu.br/palmas/wp-content/uploads/sites/31/2026/06/Ementas_Sistemas_de_Informacao_PPC_2023.pdf |
| `resolucao-consup-82-2022-politica-de-estagios.pdf` | Normas | Regimento e resoluções | 15 | https://ifpr.edu.br/paranagua/wp-content/uploads/sites/20/2022/06/Resolucao-CONSUP-IFPR-82-02062022.pdf |
| `ppc-tecnico-informatica-palmas.pdf` | Cursos | Sistemas de Informação - Palmas | 160 | https://ifpr.edu.br/palmas/wp-content/uploads/sites/31/2024/01/Projeto_Pedagogico_de_Cursos_Tecnicos_do_IFPR__Informatica.pdf |

Todos têm camada de texto (extraídos com `pdftotext -bbox-layout`, sem OCR).
A matriz e as ementas conversam com o PPC: mesmas disciplinas, mesma carga
horária, ementa por ementa.

Descartados em 2026-09-08 por serem digitalizados sem camada de texto (o OCR por
tesseract não produz bounding box, então não há grifo): Resolução Consup
55/2011 (22 MB, 42 páginas de imagem), Portaria 195/2017 de Palmas e o
Regulamento do Corpo Discente de Palmas. O Regulamento de TCC de Palmas e a
Resolução 36/2019 (estágios) estão fora do ar nos sites do IFPR (404 em todas as
variantes de URL testadas). O
PPC, com 213 páginas, é o caso de estresse do orçamento de renderização do
workspace. Os editais 56 e 57 são do Processo Seletivo 2027, publicados no
Boletim de Serviço de 2026-06-08.

Páginas de índice consultadas: https://ifpr.edu.br/processo-seletivo/processo-seletivo-2026/graduacao/ ,
https://ifpr.edu.br/institucional/o-instituto/legislacao/regimento-geral/ ,
https://reitoria.ifpr.edu.br/resolucao-082014/ ,
https://palmas.ifpr.edu.br/about/nossos-cursos/graduacao/sistemas-de-informacao/ppc-sistemas-de-informacao/ ,
https://ifpr.edu.br/palmas/about/menu-academico/calendario-academico/ .

## Adicionados em 2026-09-09

- **Resolução CONSUP/IFPR nº 82/2022 (política de estágios):** conversa com o Apêndice A do PPC de SI (regulamento de estágio) e com o Regimento Geral. Entrou na collection de normas.
- **PPC do Curso Técnico em Informática (Palmas):** cita o curso de Sistemas de Informação (co-orientação por estudantes de SI) e compartilha laboratórios, calendário e regulamentos com o PPC de SI. Entrou na collection de SI para as conexões entre cursos do mesmo campus.
- O stub `resolucao-consup-36-2019-estagios.pdf` (273 bytes, resposta 404 salva por engano) foi removido.
