"use client";

import { useState, useEffect, useRef, useCallback } from "react";
import type { LogEntry, Metrics, ActiveStages } from "./use-pipeline-simulation";

/**
 * Maps worker names from the backend telemetry events to the
 * frontend stage IDs used by the pipeline visualization.
 */
const STAGE_MAP: Record<string, string> = {
  // Crawler stages
  "download_arquivo": "download_arquivo",
  "upload_minio_raw": "upload_minio_raw",
  "extracao_links": "extracao_links",
  "fila_parser": "fila_parser",
  "fila_crawler": "fila_crawler",
  // Parser stages
  "leitura_minio": "leitura_minio",
  "processamento_regex": "processamento_regex",
  "upload_json_parsed": "upload_json_parsed",
  "fila_populate_neo4j": "fila_populate_neo4j",
  // Populator stages
  "criar_nos_relacoes": "criar_nos_relacoes",
};

type TelemetryEvent = {
  worker: string;
  stage: string;
  event_type: "info" | "warning" | "error" | "success";
  message: string;
  timestamp: string;
  metadata?: Record<string, number>;
};

type ConnectionStatus = "disconnected" | "connecting" | "connected";

/**
 * Hook that connects to the FastAPI WebSocket telemetry stream
 * and transforms incoming events into the same data shape used
 * by usePipelineSimulation — making it a drop-in replacement.
 */
export function useTelemetryStream(enabled: boolean) {
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [metrics, setMetrics] = useState<Metrics>({
    queueMessages: 0,
    minioFiles: 0,
    neo4jNodes: 0,
    currentDepth: 0,
  });
  const [activeStages, setActiveStages] = useState<ActiveStages>(new Map());
  const [status, setStatus] = useState<ConnectionStatus>("disconnected");

  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeout = useRef<NodeJS.Timeout | null>(null);
  const reconnectAttempt = useRef(0);
  const stageTimers = useRef<Map<string, NodeJS.Timeout>>(new Map());
  
  // Real-time fetching of queue sizes
  useEffect(() => {
     if (!enabled) return;
     const interval = setInterval(async () => {
         try {
             // Let's rely on telemetry metadata for now, 
             // but handle the counter logic safely to prevent stuck UI mapping.
         } catch(e) {}
     }, 2000);
     return () => clearInterval(interval);
  }, [enabled]);

  // Build the WebSocket URL from the public API URL env var
  const getWsUrl = useCallback(() => {
    const apiUrl = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8000/api/v1";
    // Strip /api/v1 suffix and convert http(s) to ws(s)
    const base = apiUrl.replace(/\/api\/v1\/?$/, "");
    const wsBase = base.replace(/^http/, "ws");
    return `${wsBase}/api/v1/monitoring/stream`;
  }, []);

  const connect = useCallback(() => {
    if (wsRef.current?.readyState === WebSocket.OPEN) return;

    const url = getWsUrl();
    setStatus("connecting");

    try {
      const ws = new WebSocket(url);
      wsRef.current = ws;

      ws.onopen = () => {
        setStatus("connected");
        reconnectAttempt.current = 0;
      };

      ws.onmessage = (event) => {
        try {
          const data: TelemetryEvent = JSON.parse(event.data);
          const stageId = STAGE_MAP[data.stage] ?? data.stage;

          // Add log entry
          const logEntry: LogEntry = {
            id: `${Date.now()}-${Math.random().toString(36).substring(7)}`,
            ts: new Date(data.timestamp),
            stage: stageId,
            message: data.message,
            type: data.event_type,
          };

          setLogs((prev) => {
            const next = [logEntry, ...prev];
            return next.slice(0, 200); // Keep last 200 entries
          });

          // Increment the stage counter
          setActiveStages((prev) => {
            const next = new Map(prev);
            next.set(stageId, (next.get(stageId) ?? 0) + 1);
            return next;
          });

          // Only set a timer if one doesn't already exist for this stage.
          // This allows the "blink" to actually happen by completing the timeout cycle
          // rather than constantly deferring the off-state.
          if (!stageTimers.current.has(stageId)) {
            const timer = setTimeout(() => {
              setActiveStages((prev) => {
                const next = new Map(prev);
                next.delete(stageId); // fully turn off the light
                return next;
              });
              stageTimers.current.delete(stageId);
            }, 600); // Shorter blink time (600ms) for better visual feedback
            stageTimers.current.set(stageId, timer);
          }

          // Update metrics from event metadata
          if (data.metadata) {
            setMetrics((prev) => {
               const newQueue = data.metadata!.queue_messages;
               // Ensure we update minio and neo4j only if they increase
               const newMinio = data.metadata!.minio_files;
               const newNeo4j = data.metadata!.neo4j_nodes;
               
               return {
                 queueMessages: newQueue !== undefined ? newQueue : prev.queueMessages,
                 minioFiles: newMinio !== undefined && newMinio > prev.minioFiles ? newMinio : prev.minioFiles,
                 neo4jNodes: newNeo4j !== undefined && newNeo4j > prev.neo4jNodes ? newNeo4j : prev.neo4jNodes,
                 currentDepth: data.metadata!.current_depth ?? prev.currentDepth,
               };
            });
          }
        } catch {
          // Ignore malformed messages
        }
      };

      ws.onclose = () => {
        setStatus("disconnected");
        wsRef.current = null;

        // Exponential backoff reconnect (up to 30s)
        if (enabled) {
          const delay = Math.min(1000 * 2 ** reconnectAttempt.current, 30000);
          reconnectAttempt.current++;
          reconnectTimeout.current = setTimeout(connect, delay);
        }
      };

      ws.onerror = () => {
        ws.close();
      };
    } catch {
      setStatus("disconnected");
    }
  }, [enabled, getWsUrl]);

  const disconnect = useCallback(() => {
    if (reconnectTimeout.current) {
      clearTimeout(reconnectTimeout.current);
      reconnectTimeout.current = null;
    }
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }
    // Clear all stage timers
    stageTimers.current.forEach((timer) => clearTimeout(timer));
    stageTimers.current.clear();

    setStatus("disconnected");
    setActiveStages(new Map());
  }, []);

  // Connect/disconnect based on the enabled flag
  useEffect(() => {
    if (enabled) {
      connect();
    } else {
      disconnect();
    }
    return disconnect;
  }, [enabled, connect, disconnect]);

  return {
    logs,
    metrics,
    activeStages,
    status,
  };
}
