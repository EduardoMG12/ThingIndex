# LegisGraph Populator

O `graph-populator` é o componente responsável por ingerir os dados processados (arquivos JSON) no banco de dados orientado a grafos Neo4j.

## 📋 Pré-requisitos

- **Go** 1.22+
- **Neo4j** (pode ser via Docker, verifique o `docker-compose.yml` na raiz do projeto)
- Dados processados pelo `legal-parser` (normalmente localizados em `../legal-parser/parsed_data`)

## ⚙️ Configuração

O projeto utiliza variáveis de ambiente para configuração. Você pode definir essas variáveis no seu terminal ou criar um arquivo `.env` (embora o script atual leia do ambiente).

| Variável | Descrição | Valor Padrão (Exemplo) |
|----------|-----------|------------------------|
| `NEO4J_URI` | Endereço do banco Neo4j | `bolt://localhost:7687` |
| `NEO4J_USERNAME` | Usuário do Neo4j | `neo4j` |
| `NEO4J_PASSWORD` | Senha do Neo4j | `minhasenha123` |
| `DATA_DIR` | Caminho para a pasta de arquivos JSON | `../legal-parser/parsed_data` |

## 🚀 Como Rodar

### 1. Inicie o Neo4j
Certifique-se de que o Neo4j está rodando. Se estiver usando o Docker Compose na raiz do projeto:

```bash
# Na raiz do projeto (LegisGraph/)
docker-compose up -d neo4j
```

### 2. Execute o Populator

Navegue até a pasta do `graph-populator` e execute o comando `go run`:

```bash
cd graph-populator

# Exemplo definindo as variáveis em linha (Linux/Mac)
export NEO4J_URI=bolt://localhost:7687
export NEO4J_USERNAME=neo4j
export NEO4J_PASSWORD=minhasenha123
export DATA_DIR=../legal-parser/parsed_data

go run cmd/populator/main.go
```

Ou, se você já tiver as variáveis exportadas:

```bash
go run cmd/populator/main.go
```

## 📂 Estrutura de Dados
O script espera encontrar arquivos `.json` no diretório especificado. Cada arquivo deve representar uma lei e seguir o schema de saída do `legal-parser`.

O script irá criar:
- Nós de **Lei**
- Nós de **Artigo**
- Nós de **Paragrafo**
- Nós de **Inciso**
- Nós de **Alinea**
- Relacionamentos de hierarquia (`:CONTEM`) entre eles.
