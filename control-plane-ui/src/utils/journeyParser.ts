/**
 * Journey telemetry parser — extracts and normalizes log records from
 * OTLP/JSON envelopes, flat arrays, and NDJSON into a unified shape
 * that the JourneyReplay component renders.
 *
 * Extracted from JourneyReplay.tsx for testability.
 */

export type LogAttributeValue = {
  stringValue?: string;
  intValue?: string | number;
  boolValue?: boolean;
  doubleValue?: number;
};

export type LogAttribute = { key: string; value: LogAttributeValue };

export type LogRecord = {
  timeUnixNano?: string | number;
  observedTimeUnixNano?: string | number;
  severityText?: string;
  severityNumber?: number;
  body?: { stringValue?: string };
  attributes?: LogAttribute[];
  traceId?: string;
  spanId?: string;
};

export type FlatRecord = {
  body?: string;
  trace_id?: string;
  traceId?: string;
  span_id?: string;
  spanId?: string;
  timestamp?: string | number;
  time?: string | number;
  attributes?: Record<string, unknown>;
};

export interface NormalizedEvent {
  timestampMs: number;
  body: string;
  traceId: string;
  spanId: string;
  attributes: Record<string, unknown>;
}

export function parseEvents(raw: string): NormalizedEvent[] {
  const trimmed = raw.trim();
  if (!trimmed) return [];

  let parsed: unknown;
  try {
    parsed = JSON.parse(trimmed);
  } catch {
    // Fall through to NDJSON parsing.
  }

  if (parsed && typeof parsed === 'object' && 'resourceLogs' in parsed) {
    const events: NormalizedEvent[] = [];
    const env = parsed as { resourceLogs: { scopeLogs: { logRecords: LogRecord[] }[] }[] };
    for (const rl of env.resourceLogs ?? []) {
      for (const sl of rl.scopeLogs ?? []) {
        for (const r of sl.logRecords ?? []) {
          events.push(normalizeLogRecord(r));
        }
      }
    }
    return events;
  }

  if (Array.isArray(parsed)) {
    return (parsed as Array<LogRecord | FlatRecord>).map(normalizeAny);
  }

  const events: NormalizedEvent[] = [];
  for (const line of trimmed.split('\n')) {
    const t = line.trim();
    if (!t) continue;
    try {
      events.push(normalizeAny(JSON.parse(t)));
    } catch {
      // skip malformed lines
    }
  }
  return events;
}

export function normalizeAny(record: LogRecord | FlatRecord | Record<string, unknown>): NormalizedEvent {
  const r = record as LogRecord & FlatRecord;
  if (r.body && typeof r.body === 'object' && 'stringValue' in r.body) {
    return normalizeLogRecord(r as LogRecord);
  }
  return normalizeFlat(r as FlatRecord);
}

export function normalizeLogRecord(r: LogRecord): NormalizedEvent {
  const attrs: Record<string, unknown> = {};
  for (const a of r.attributes ?? []) {
    attrs[a.key] = (
      a.value.stringValue
      ?? a.value.intValue
      ?? a.value.boolValue
      ?? a.value.doubleValue
    );
  }
  const t = Number(r.timeUnixNano ?? r.observedTimeUnixNano ?? 0);
  return {
    timestampMs: Math.floor(t / 1_000_000),
    body: r.body?.stringValue ?? '',
    traceId: r.traceId ?? '',
    spanId: r.spanId ?? '',
    attributes: attrs,
  };
}

export function normalizeFlat(r: FlatRecord): NormalizedEvent {
  const ts = r.timestamp ?? r.time;
  const tsMs = typeof ts === 'string' ? Date.parse(ts) : Number(ts) || 0;
  return {
    timestampMs: tsMs,
    body: typeof r.body === 'string' ? r.body : '',
    traceId: r.trace_id ?? r.traceId ?? '',
    spanId: r.span_id ?? r.spanId ?? '',
    attributes: r.attributes ?? {},
  };
}

export function attr<T = string>(e: NormalizedEvent, key: string): T | undefined {
  const v = e.attributes[key];
  return v as T | undefined;
}

export function groupByTrace(events: NormalizedEvent[]): Array<[string, NormalizedEvent[]]> {
  const m = new Map<string, NormalizedEvent[]>();
  for (const e of events) {
    const key = e.traceId || '(no trace_id)';
    if (!m.has(key)) m.set(key, []);
    m.get(key)!.push(e);
  }
  return Array.from(m.entries()).sort((a, b) =>
    (a[1][0]?.timestampMs ?? 0) - (b[1][0]?.timestampMs ?? 0)
  );
}
