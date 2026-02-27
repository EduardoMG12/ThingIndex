import { useState, useCallback, useRef, useEffect } from "react";

export type PipelineStage = {
  id: string;
  label: string;
  group: string;
};

export type LogEntry = {
  id: string;
  ts: Date;
  stage: string;
  message: string;
  type: "info" | "warning" | "error" | "success";
};

export type Metrics = {
  queueMessages: number;
  minioFiles: number;
  neo4jNodes: number;
  currentDepth: number;
};

export const GROUPS = ["Entrada", "Worker Crawler", "Worker Parser", "Worker Neo4j"];

export const STAGES: PipelineStage[] = [
  // Entrada
  { id: "api_call", label: "API Call (URL + Depth)", group: "Entrada" },
  { id: "fila_crawler", label: "Fila Crawler", group: "Entrada" },
  // Worker Crawler
  { id: "download_arquivo", label: "Download Arquivo", group: "Worker Crawler" },
  { id: "upload_minio_raw", label: "Upload MinIO (Raw)", group: "Worker Crawler" },
  { id: "extracao_links", label: "Extração de Links", group: "Worker Crawler" },
  { id: "fila_parser", label: "Fila Parser", group: "Worker Crawler" },
  // Worker Parser
  { id: "leitura_minio", label: "Leitura MinIO", group: "Worker Parser" },
  { id: "processamento_regex", label: "Processamento Regex", group: "Worker Parser" },
  { id: "upload_json_parsed", label: "Upload JSON (Parsed)", group: "Worker Parser" },
  { id: "fila_populate_neo4j", label: "Fila Populate Neo4j", group: "Worker Parser" },
  // Worker Neo4j
  { id: "criar_nos_relacoes", label: "Criar Nós / Relações", group: "Worker Neo4j" },
];

export const GROUP_COLORS: Record<string, string> = {
  Entrada: "hsl(40, 90%, 55%)",
  "Worker Crawler": "hsl(200, 70%, 55%)",
  "Worker Parser": "hsl(150, 60%, 50%)",
  "Worker Neo4j": "hsl(320, 60%, 55%)",
};

export type ActiveStages = Set<string>;

export function usePipelineSimulation() {
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [metrics, setMetrics] = useState<Metrics>({
    queueMessages: 0,
    minioFiles: 0,
    neo4jNodes: 0,
    currentDepth: 0,
  });
  const [activeStages, setActiveStages] = useState<ActiveStages>(new Set());
  const [simRunning, setSimRunning] = useState(false);
  const [simDone, setSimDone] = useState(false);

  const timeoutIds = useRef<NodeJS.Timeout[]>([]);

  const addLog = useCallback((stage: string, message: string, type: LogEntry["type"] = "info") => {
    setLogs((prev) => [
      ...prev,
      { id: Math.random().toString(36).substring(7), ts: new Date(), stage, message, type },
    ]);
  }, []);

  const clearTimeouts = useCallback(() => {
    timeoutIds.current.forEach(clearTimeout);
    timeoutIds.current = [];
  }, []);

  useEffect(() => {
    return clearTimeouts;
  }, [clearTimeouts]);

  const startSimulation = useCallback(() => {
    if (simRunning) return;
    setSimRunning(true);
    setSimDone(false);
    setLogs([]);
    setMetrics({ queueMessages: 0, minioFiles: 0, neo4jNodes: 0, currentDepth: 0 });
    setActiveStages(new Set());
    clearTimeouts();

    let time = 0;
    const schedule = (delay: number, fn: () => void) => {
      time += delay;
      timeoutIds.current.push(setTimeout(fn, time));
    };

    const activate = (stage: string) => setActiveStages((p) => new Set(p).add(stage));
    const deactivate = (stage: string) =>
      setActiveStages((p) => {
        const next = new Set(p);
        next.delete(stage);
        return next;
      });

    // Entrada Phase
    schedule(500, () => {
      activate("api_call");
      addLog("api_call", "Recebendo requisição da API para iniciar pipeline");
      setMetrics((m) => ({ ...m, currentDepth: 1 }));
    });
    schedule(800, () => {
      deactivate("api_call");
      activate("fila_crawler");
      addLog("fila_crawler", "Enfileirando job para o Crawler");
      setMetrics((m) => ({ ...m, queueMessages: 1 }));
    });

    // Worker Crawler Phase
    schedule(1000, () => {
      deactivate("fila_crawler");
      activate("download_arquivo");
      setMetrics((m) => ({ ...m, queueMessages: 0 }));
      addLog("download_arquivo", "Baixando página HTML alvo");
    });
    schedule(1200, () => {
      deactivate("download_arquivo");
      activate("upload_minio_raw");
      addLog("upload_minio_raw", "Salvando arquivo original no MinIO (Raw)");
      setMetrics((m) => ({ ...m, minioFiles: 1 }));
    });
    schedule(800, () => {
      deactivate("upload_minio_raw");
      activate("extracao_links");
      addLog("extracao_links", "Extraindo links internos para proxima depth");
    });
    schedule(900, () => {
      deactivate("extracao_links");
      activate("fila_parser");
      addLog("fila_parser", "Enfileirando documento para parsing subsequente");
      setMetrics((m) => ({ ...m, queueMessages: 1 }));
    });

    // Worker Parser Phase
    schedule(1000, () => {
      deactivate("fila_parser");
      activate("leitura_minio");
      setMetrics((m) => ({ ...m, queueMessages: 0 }));
      addLog("leitura_minio", "Lendo arquivo Raw do MinIO");
    });
    schedule(1100, () => {
      deactivate("leitura_minio");
      activate("processamento_regex");
      addLog("processamento_regex", "Aplicando regex para limpeza e estruturação do texto");
    });
    schedule(1200, () => {
      deactivate("processamento_regex");
      activate("upload_json_parsed");
      addLog("upload_json_parsed", "Salvando payload processado (JSON) no MinIO");
      setMetrics((m) => ({ ...m, minioFiles: 2 }));
    });
    schedule(800, () => {
      deactivate("upload_json_parsed");
      activate("fila_populate_neo4j");
      addLog("fila_populate_neo4j", "Enfileirando JSON para população no Grafo");
      setMetrics((m) => ({ ...m, queueMessages: 1 }));
    });

    // Worker Neo4j Phase
    schedule(1000, () => {
      deactivate("fila_populate_neo4j");
      activate("criar_nos_relacoes");
      setMetrics((m) => ({ ...m, queueMessages: 0 }));
      addLog("criar_nos_relacoes", "Criando Nós e Relações baseadas no documento Parsed");
      setMetrics((m) => ({ ...m, neo4jNodes: 243 }));
    });

    // Finish
    schedule(1500, () => {
      deactivate("criar_nos_relacoes");
      addLog("criar_nos_relacoes", "Pipeline concluído com sucesso", "success");
      setSimRunning(false);
      setSimDone(true);
    });
  }, [addLog, clearTimeouts, simRunning]);

  return {
    logs,
    metrics,
    activeStages,
    simRunning,
    simDone,
    startSimulation,
  };
}
