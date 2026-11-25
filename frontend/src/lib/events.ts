export interface ReleaseProgressEvent {
  job_id: string;
  project_id: string;
  stage: string;
  bytes_done: number;
  bytes_total: number;
  percent: number;
}

export interface BuildLogEvent {
  job_id: string;
  project_id: string;
  stream: "stdout" | "stderr";
  line: string;
  ts: number;
}

export interface BuildStatusEvent {
  job_id: string;
  project_id: string;
  status: "running" | "success" | "failed" | "cancelled";
  error?: string;
}

export interface ProcessChangedEvent {
  project_id: string;
  running: boolean;
  pid: number;
  ports: number[];
}

declare global {
  interface Window {
    runtime?: {
      EventsOnMultiple?: (
        eventName: string,
        callback: (...data: unknown[]) => void,
        maxCallbacks: number
      ) => () => void;
    };
  }
}

export function subscribeEvent<T>(
  eventName: string,
  handler: (payload: T) => void
): () => void {
  const on = window.runtime?.EventsOnMultiple;
  if (!on) {
    return () => {};
  }
  return on(
    eventName,
    (payload: unknown) => {
      handler(payload as T);
    },
    -1
  );
}
