# Modelo de Texto — 4.4 Modelagem Relacional Aplicada a Sistemas Documentais

## Texto Modelo

### 4.4 Modelagem Relacional Aplicada a Sistemas Documentais

O **modelo relacional** foi proposto por Edgar F. Codd em 1970, no artigo "A Relational Model of Data for Large Shared Data Banks" (Codd, 1970). Antes de Codd, sistemas de banco de dados eram baseados em modelos hierárquicos ou de rede, nos quais a navegação entre os dados era controlada por ponteiros e caminhos pré-definidos. O modelo relacional introduziu uma abordagem radicalmente diferente: os dados são organizados em **relações** (tabelas), cada linha é uma **tupla** e cada coluna é um **atributo**.

A consistência dos dados é garantida por três mecanismos principais:

**Chaves e integridade referencial.** Cada tupla em uma relação é identificada de forma única por uma **chave primária** (primary key). Quando uma tabela faz referência a outra, utiliza-se uma **chave estrangeira** (foreign key) que aponta para a chave primária da tabela referenciada. Isso garante que não existam referências órfãs — uma propriedade chamada integridade referencial.

**Normalização.** Para evitar redundância e anomalias de atualização, o modelo relacional preconiza a **normalização** dos dados. As três formas normais principais (Elmasri e Navathe, 2016) são:
- 1FN: cada coluna deve conter valores atômicos (não listas ou conjuntos);
- 2FN: cada coluna não-chave deve depender da totalidade da chave primária;
- 3FN: colunas não-chave não devem depender de outras colunas não-chave.

**Propriedades ACID.** Bancos relacionais garantem quatro propriedades fundamentais em suas transações (Date, 2004):
- **Atomicidade:** uma transação é executada por completo ou não é executada;
- **Consistência:** uma transação leva o banco de um estado válido a outro estado válido;
- **Isolamento:** transações simultâneas não interferem entre si;
- **Durabilidade:** uma vez confirmada, a transação persiste mesmo em caso de falha do sistema.

**Proveniência de dados.** Buneman et al. (2001) introduziram o conceito de **proveniência** como a capacidade de rastrear a origem e as transformações de um dado dentro de um sistema. Em bancos relacionais, a proveniência pode ser implementada por meio de chaves estrangeiras, registros de auditoria e consultas determinísticas — cada dado pode ser rastreado até sua origem.

**Aplicação ao ThingIndex.** O ThingIndex utiliza o PostgreSQL como banco relacional principal, adotando a modelagem relacional como estratégia central de organização documental. Os documentos são fragmentados em tabelas tipadas (parágrafos, tabelas, metadados), cada uma com chaves estrangeiras que apontam para o documento de origem e índices compostos por `(document_id, sequence_order)` para preservar a ordenação original.

Essa abordagem materializa o conceito de proveniência descrito por Buneman: cada parágrafo recuperado pode ser rastreado até o documento, página e posição de origem por meio de uma sequência de consultas SQL determinísticas. A escolha pelo modelo relacional — com sua rigidez de schema e custo de normalização — é deliberada: prioriza-se a auditabilidade e a previsibilidade operacional em detrimento da flexibilidade de esquemas NoSQL ou da opacidade de bancos vetoriais.

---

## Referências para buscar no Google Acadêmico

- Codd, E. F. (1970). "A Relational Model of Data for Large Shared Data Banks". *Communications of the ACM*.
- Date, C. J. (2004). *An Introduction to Database Systems*. 8th ed.
- Elmasri, R.; Navathe, S. B. (2016). *Fundamentals of Database Systems*. 7th ed.
- Buneman, P.; Khanna, S.; Tan, W. C. (2001). "Why and Where: A Characterization of Data Provenance". *ICDT*.

---

## Checklist

- [ ] Entendo a diferença entre chave primária e estrangeira
- [ ] Sei explicar as 3 formas normais sem consultar
- [ ] Entendo ACID e como se aplica ao ThingIndex
- [ ] Sei o que é proveniência de dados e por que é relevante
- [ ] Conectei ao ThingIndex (20 tabelas, chaves estrangeiras, consultas SQL)
