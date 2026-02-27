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

  // Use refs to hold mutable timeout references
  const timeoutIds = useRef<NodeJS.Timeout[]>([]);
  // Store a monotonic clock reference to align scheduled events
  const startTime = useRef<number>(0);

  const addLog = useCallback((stage: string, message: string, type: LogEntry["type"] = "info") => {
    setLogs((prev) => {
      const newLog = { id: Math.random().toString(36).substring(7), ts: new Date(), stage, message, type };
      // Keep only last 100 logs to prevent memory leak and lag during mass concurrency
      const nextLogs = [newLog, ...prev];
      return nextLogs.slice(0, 100); 
    });
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

    startTime.current = Date.now();
    
    // Helper to schedule a function at an *absolute* offset from when simulation started
    const scheduleAbsolute = (offsetMs: number, fn: () => void) => {
      const msSinceStart = Date.now() - startTime.current;
      const delay = Math.max(0, offsetMs - msSinceStart);
      timeoutIds.current.push(setTimeout(fn, delay));
    };

    const activate = (stage: string) => setActiveStages((p) => new Set(p).add(stage));
    const deactivate = (stage: string) =>
      setActiveStages((p) => {
        const next = new Set(p);
        next.delete(stage);
        return next;
      });

    // Helper to update metrics functionally
    const updateMetrics = (updater: (prev: Metrics) => Partial<Metrics>) => {
      setMetrics((prev) => ({ ...prev, ...updater(prev) }));
    };

    // --- Mocks: Simulating processing of one unit of work through the pipeline ---
    
    // Simulate Crawler Worker processing a given URL ID at a specific start time
    const simulateCrawlerTask = (taskId: string, absoluteStartMs: number, processingSpeed: number) => {
      const t1 = absoluteStartMs;
      const t2 = t1 + 300 * processingSpeed;
      const t3 = t2 + 200 * processingSpeed;
      const t4 = t3 + 300 * processingSpeed;

      scheduleAbsolute(t1, () => {
        activate("download_arquivo");
        updateMetrics((m) => ({ queueMessages: Math.max(0, m.queueMessages - 1) }));
        addLog("download_arquivo", `[${taskId}] Baixando página HTML alvo`);
      });

      scheduleAbsolute(t2, () => {
        deactivate("download_arquivo");
        activate("upload_minio_raw");
        addLog("upload_minio_raw", `[${taskId}] Salvando arquivo original no MinIO`);
        updateMetrics((m) => ({ minioFiles: m.minioFiles + 1 }));
      });

      scheduleAbsolute(t3, () => {
        deactivate("upload_minio_raw");
        activate("extracao_links");
        addLog("extracao_links", `[${taskId}] Extraindo links internos (depth+1)`);
      });

      scheduleAbsolute(t4, () => {
        deactivate("extracao_links");
        activate("fila_parser");
        addLog("fila_parser", `[${taskId}] Enfileirando HTML para parsing`);
        updateMetrics((m) => ({ queueMessages: m.queueMessages + 1 }));
        
        // As soon as crawler finishes, parser picks it up almost immediately
        setTimeout(() => deactivate("fila_parser"), 400); 
        simulateParserTask(taskId, t4 + 200, processingSpeed);
      });
    };

    // Simulate Parser Worker
    const simulateParserTask = (taskId: string, absoluteStartMs: number, processingSpeed: number) => {
      const t1 = absoluteStartMs;
      const t2 = t1 + 250 * processingSpeed;
      const t3 = t2 + 400 * processingSpeed;
      const t4 = t3 + 200 * processingSpeed;

      scheduleAbsolute(t1, () => {
        activate("leitura_minio");
        updateMetrics((m) => ({ queueMessages: Math.max(0, m.queueMessages - 1) }));
        addLog("leitura_minio", `[${taskId}] Lendo arquivo Raw do MinIO`);
      });

      scheduleAbsolute(t2, () => {
        deactivate("leitura_minio");
        activate("processamento_regex");
        addLog("processamento_regex", `[${taskId}] Aplicando regex para estruturação`);
      });

      scheduleAbsolute(t3, () => {
        deactivate("processamento_regex");
        activate("upload_json_parsed");
        addLog("upload_json_parsed", `[${taskId}] Salvando payload processado (JSON)`);
        updateMetrics((m) => ({ minioFiles: m.minioFiles + 1 }));
      });

      scheduleAbsolute(t4, () => {
        deactivate("upload_json_parsed");
        activate("fila_populate_neo4j");
        addLog("fila_populate_neo4j", `[${taskId}] Enfileirando JSON para Neo4j`);
        updateMetrics((m) => ({ queueMessages: m.queueMessages + 1 }));

        setTimeout(() => deactivate("fila_populate_neo4j"), 400);
        simulateNeo4jTask(taskId, t4 + 200, processingSpeed);
      });
    };

    // Simulate Neo4j Worker
    const simulateNeo4jTask = (taskId: string, absoluteStartMs: number, processingSpeed: number) => {
      const t1 = absoluteStartMs;
      const duration = 500 * processingSpeed;

      scheduleAbsolute(t1, () => {
        activate("criar_nos_relacoes");
        updateMetrics((m) => ({ queueMessages: Math.max(0, m.queueMessages - 1) }));
        addLog("criar_nos_relacoes", `[${taskId}] Criando Nós e Relações no Grafo`);
      });

      // Neo4j Node creation is "bulk" fast
      for (let i = 1; i <= 5; i++) {
        scheduleAbsolute(t1 + (duration / 5) * i, () => {
           updateMetrics((m) => ({ neo4jNodes: m.neo4jNodes + Math.floor(Math.random() * 15) + 5 }));
        });
      }

      scheduleAbsolute(t1 + duration, () => {
        deactivate("criar_nos_relacoes");
      });
    };


    // ----------------------------------------------------
    // PHASE 1: Warmup (Slow, didactic initial request)
    // ----------------------------------------------------
    let currentMs = 500;
    
    scheduleAbsolute(currentMs, () => {
      activate("api_call");
      addLog("api_call", "Recebendo requisição da API (Seed URL)");
      updateMetrics(() => ({ currentDepth: 1 }));
    });
    
    currentMs += 800;
    scheduleAbsolute(currentMs, () => {
      deactivate("api_call");
      activate("fila_crawler");
      addLog("fila_crawler", "Enfileirando Seed Job para o Crawler");
      updateMetrics((m) => ({ queueMessages: m.queueMessages + 1 }));
    });

    currentMs += 600;
    scheduleAbsolute(currentMs, () => {
       deactivate("fila_crawler");
    });
    
    // Run the first task very slowly (speed multiplier 2.5)
    simulateCrawlerTask("Seed", currentMs, 2.5);

    // ----------------------------------------------------
    // PHASE 2: Explosion of Concurrency (Cascade Effect)
    // ----------------------------------------------------
    // The seed task will finish its crawler phase around currentMs + (1000 * 2.5) = currentMs + 2500.
    // Let's start the cascade right around when it extracts links.
    
    const cascadeStartMs = currentMs + 2200; 

    scheduleAbsolute(cascadeStartMs, () => {
      updateMetrics(() => ({ currentDepth: 2 }));
      addLog("api_call", "Profundidade 2 alcançada: Descobertos múltiplos links! Iniciando processamento em lote.", "warning");
    });

    // We'll spawn 15 concurrent tasks rapidly
    // As Depth increases, processing speed factor gets smaller (faster execution)
    for (let i = 0; i < 15; i++) {
        const spawnDelay = i * 200 + (Math.random() * 150); // Spread them out slightly
        const taskSpeed = 0.6 + (Math.random() * 0.4); // Faster than seed: 0.6 to 1.0 multiplier

        scheduleAbsolute(cascadeStartMs + spawnDelay, () => {
             // Fake the queue jumping up quickly for each discovered link
             activate("fila_crawler");
             updateMetrics((m) => ({ queueMessages: m.queueMessages + 1 }));
             
             // Deactivate shortly after just to flash the UI indicator
             setTimeout(() => deactivate("fila_crawler"), 300);
             
             simulateCrawlerTask(`Link-${i+1}`, Date.now() - startTime.current + 300, taskSpeed);
        });
    }

    // Phase 3: Even more explosion
    const cascadeStartMs2 = cascadeStartMs + 3000;
    scheduleAbsolute(cascadeStartMs2, () => {
      updateMetrics(() => ({ currentDepth: 3 }));
      addLog("api_call", "Profundidade 3! Carga exponencial acionada...", "warning");
    });

    for (let i = 0; i < 20; i++) {
        const spawnDelay = i * 100 + (Math.random() * 100); 
        const taskSpeed = 0.3 + (Math.random() * 0.3); // Super fast

        scheduleAbsolute(cascadeStartMs2 + spawnDelay, () => {
             activate("fila_crawler");
             updateMetrics((m) => ({ queueMessages: m.queueMessages + 1 }));
             setTimeout(() => deactivate("fila_crawler"), 200);
             simulateCrawlerTask(`SubL-${i+1}`, Date.now() - startTime.current + 200, taskSpeed);
        });
    }

    // ----------------------------------------------------
    // Finish Simulation
    // ----------------------------------------------------
    // 20 tasks * 100ms = 2000ms spread. The last task starts around cascadeStartMs2 + 2000.
    // It takes some time to flow through (around 1500ms max with 0.3 speed).
    // Let's cap the simulation at 12 seconds total.
    
    const finishTime = cascadeStartMs2 + 6000; 

    scheduleAbsolute(finishTime, () => {
      setSimRunning(false);
      setSimDone(true);
      setActiveStages(new Set()); // Clear all glowing states
      addLog("api_call", "Simulação de carga concluída. Sistema estabilizado.", "success");
      // Force queue to 0
      updateMetrics(() => ({ queueMessages: 0 })); 
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
