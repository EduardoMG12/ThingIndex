/**
 * Static data definitions for the system architecture diagram.
 * Node/edge positions and stage-ID mappings are kept here so
 * the rendering component stays lean and testable.
 */

export type ArchNode = {
  id: string;
  label: string;
  x: number;
  y: number;
  w: number;
  h: number;
  color: string;
  /** Pipeline stage IDs that make this node "active" */
  stages?: string[];
  children?: { label: string; stage?: string }[];
};

export type ArchEdge = {
  from: string;
  to: string;
  label?: string;
  color: string;
  offsetX?: number;
  offsetY?: number;
  textOffsetX?: number;
  textOffsetY?: number;
};

/* ── Nodes ──
 *
 * Layout follows the data flow top-to-bottom:
 *
 *   Row 0:  Frontend ──► API ──► Fila Crawler
 *   Row 1:  MinIO (left)         Worker Crawler (right)
 *   Row 2:              Fila Parser
 *   Row 3:              Worker Parser
 *   Row 4:              Fila Neo4j
 *   Row 5:              Worker Neo4j
 *   Row 6:              Neo4j DB
 */
export const ARCH_NODES: ArchNode[] = [
  /* ── Row 0 — entry ── */
  {
    id: "frontend",
    label: "Frontend (Next.js)",
    x: 40, y: 20, w: 155, h: 44,
    color: "hsl(40,90%,55%)",
    stages: ["api_call"],
  },
  {
    id: "api",
    label: "API Gateway",
    x: 250, y: 20, w: 155, h: 44,
    color: "hsl(40,90%,55%)",
    stages: ["api_call"],
  },
  {
    id: "fila-crawler",
    label: "Fila Crawler",
    x: 500, y: 20, w: 140, h: 44,
    color: "hsl(40,90%,55%)",
    stages: ["fila_crawler"],
  },

  /* ── Row 1 — crawler ── */
  {
    id: "worker-crawler",
    label: "Worker Crawler",
    x: 400, y: 95, w: 240, h: 170,
    color: "hsl(200,70%,55%)",
    stages: ["download_arquivo", "upload_minio_raw", "extracao_links", "fila_parser"],
    children: [
      { label: "Download Arquivo", stage: "download_arquivo" },
      { label: "Upload MinIO (Raw)", stage: "upload_minio_raw" },
      { label: "Extração de Links", stage: "extracao_links" },
      { label: "Envia → Fila Parser", stage: "fila_parser" },
    ],
  },
  {
    id: "minio",
    label: "MinIO (Object Storage)",
    x: 40, y: 115, w: 195, h: 100,
    color: "hsl(30,80%,55%)",
    stages: ["upload_minio_raw", "upload_json_parsed", "leitura_minio"],
    children: [
      { label: "Bucket Raw", stage: "upload_minio_raw" },
      { label: "Bucket Parsed", stage: "upload_json_parsed" },
    ],
  },

  /* ── Row 2 — parser queue ── */
  {
    id: "fila-parser",
    label: "Fila Parser",
    x: 320, y: 295, w: 130, h: 44,
    color: "hsl(200,70%,55%)",
    stages: ["fila_parser"],
  },

  /* ── Row 3 — parser ── */
  {
    id: "worker-parser",
    label: "Worker Parser",
    x: 270, y: 370, w: 240, h: 170,
    color: "hsl(150,60%,50%)",
    stages: ["leitura_minio", "processamento_regex", "upload_json_parsed", "fila_populate_neo4j"],
    children: [
      { label: "Leitura MinIO", stage: "leitura_minio" },
      { label: "Processamento Regex", stage: "processamento_regex" },
      { label: "Upload JSON (Parsed)", stage: "upload_json_parsed" },
      { label: "Envia → Fila Neo4j", stage: "fila_populate_neo4j" },
    ],
  },

  /* ── Row 4 — neo4j queue ── */
  {
    id: "fila-neo4j",
    label: "Fila Neo4j",
    x: 340, y: 570, w: 140, h: 44,
    color: "hsl(150,60%,50%)",
    stages: ["fila_populate_neo4j"],
  },

  /* ── Row 5 — neo4j populator ── */
  {
    id: "worker-neo4j",
    label: "Worker Neo4j",
    x: 310, y: 645, w: 230, h: 80,
    color: "hsl(320,60%,55%)",
    stages: ["criar_nos_relacoes"],
    children: [
      { label: "Criar Nós / Relações", stage: "criar_nos_relacoes" },
    ],
  },

  /* ── Row 6 — database ── */
  {
    id: "neo4j-db",
    label: "Neo4j (Graph DB)",
    x: 340, y: 755, w: 170, h: 44,
    color: "hsl(320,60%,55%)",
    stages: ["criar_nos_relacoes"],
  },
];

/* ── Edges ── */
export const ARCH_EDGES: ArchEdge[] = [
  { from: "frontend", to: "api", label: "HTTP", color: "hsl(40,90%,55%)" },
  { from: "api", to: "fila-crawler", label: "publish", color: "hsl(40,90%,55%)" },
  { from: "fila-crawler", to: "worker-crawler", label: "consume", color: "hsl(200,70%,55%)", offsetX: -20, textOffsetX: -15, textOffsetY: -5 },
  { from: "worker-crawler", to: "minio", label: "upload raw", color: "hsl(200,70%,55%)" },
  { from: "worker-crawler", to: "fila-parser", label: "publish", color: "hsl(200,70%,55%)" },
  { from: "worker-crawler", to: "fila-crawler", label: "novos links", color: "hsl(200,70%,55%)", offsetX: 20, textOffsetX: 25, textOffsetY: 5 },
  { from: "fila-parser", to: "worker-parser", label: "consume", color: "hsl(150,60%,50%)" },
  { from: "worker-parser", to: "minio", label: "read / write", color: "hsl(150,60%,50%)" },
  { from: "worker-parser", to: "fila-neo4j", label: "publish", color: "hsl(150,60%,50%)" },
  { from: "fila-neo4j", to: "worker-neo4j", label: "consume", color: "hsl(320,60%,55%)" },
  { from: "worker-neo4j", to: "neo4j-db", label: "write", color: "hsl(320,60%,55%)" },
];

/* ── SVG dimensions ── */
export const SVG_W = 700;
export const SVG_H = 820;
export const CHILD_BOX_H = 32;

/* ── Geometry helpers ── */
export function nodeCenter(n: ArchNode) {
  return { x: n.x + n.w / 2, y: n.y + n.h / 2 };
}

export function edgeEndpoints(from: ArchNode, to: ArchNode) {
  const fc = nodeCenter(from);
  const tc = nodeCenter(to);
  const dx = tc.x - fc.x;
  const dy = tc.y - fc.y;

  let x1: number, y1: number, x2: number, y2: number;

  if (Math.abs(dx) > Math.abs(dy)) {
    x1 = dx > 0 ? from.x + from.w : from.x;
    y1 = fc.y;
    x2 = dx > 0 ? to.x : to.x + to.w;
    y2 = tc.y;
  } else {
    x1 = fc.x;
    y1 = dy > 0 ? from.y + from.h : from.y;
    x2 = tc.x;
    y2 = dy > 0 ? to.y : to.y + to.h;
  }

  return { x1, y1, x2, y2 };
}
