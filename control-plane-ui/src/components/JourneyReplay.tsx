import { useEffect, useMemo, useState } from 'react';
import { parseEvents, groupByTrace, attr } from '../utils/journeyParser';
import type { NormalizedEvent } from '../utils/journeyParser';

function formatTime(ms: number): string {
  if (!ms) return '—';
  const d = new Date(ms);
  return d.toISOString().slice(11, 23);
}

function ScreenshotRow({
  event,
  onOpen,
}: {
  event: NormalizedEvent;
  onOpen: (dataUrl: string, label: string) => void;
}) {
  const dataUrl = attr<string>(event, 'mobile.screenshot.data_url');
  const trigger = attr<string>(event, 'mobile.screenshot.trigger') ?? 'manual';
  const screen = attr<string>(event, 'screen.name') ?? '';
  const sizeBytes = attr<number>(event, 'mobile.screenshot.size_bytes');
  const width = attr<number>(event, 'mobile.screenshot.width');
  const height = attr<number>(event, 'mobile.screenshot.height');
  const label = `${formatTime(event.timestampMs)} · ${trigger}${screen ? ` · ${screen}` : ''}`;
  return (
    <div style={{ border: '1px solid #ccc', borderRadius: 4, padding: 8, marginBottom: 8 }}>
      <div style={{ fontSize: 12, color: '#666' }}>
        <strong>{formatTime(event.timestampMs)}</strong> · ui.screenshot · trigger=<code>{trigger}</code>
        {screen && <> · screen=<code>{screen}</code></>}
        {typeof sizeBytes === 'number' && <> · {Math.round(sizeBytes / 1024)} KB</>}
        {typeof width === 'number' && typeof height === 'number' && <> · {width}×{height}</>}
      </div>
      {dataUrl && (
        <img
          src={dataUrl}
          alt={`screenshot ${trigger}`}
          onClick={() => onOpen(dataUrl, label)}
          title="Click to view full-size"
          style={{
            maxWidth: 320,
            maxHeight: 480,
            marginTop: 4,
            border: '1px solid #eee',
            cursor: 'zoom-in',
          }}
        />
      )}
    </div>
  );
}

interface WireframeNode {
  type?: string;
  bounds?: { left: number; top: number; right: number; bottom: number };
  text?: string;
  id?: string;
  clickable?: boolean;
  children?: WireframeNode[];
}

function WireframeTree({ node, depth = 0 }: { node: WireframeNode; depth?: number }) {
  const [open, setOpen] = useState(depth < 2);
  const hasChildren = (node.children?.length ?? 0) > 0;
  const indent = depth * 12;
  return (
    <div style={{ marginLeft: indent, fontSize: 11, fontFamily: 'monospace' }}>
      <div onClick={() => hasChildren && setOpen(!open)} style={{ cursor: hasChildren ? 'pointer' : 'default' }}>
        {hasChildren ? (open ? '▾ ' : '▸ ') : '· '}
        <strong>{node.type ?? '(unknown)'}</strong>
        {node.id && <span style={{ color: '#888' }}> #{node.id}</span>}
        {node.text && <span style={{ color: '#0a6' }}> "{node.text}"</span>}
        {node.clickable && <span style={{ color: '#a60' }}> [clickable]</span>}
      </div>
      {open && hasChildren && node.children!.map((c, i) => (
        <WireframeTree key={i} node={c} depth={depth + 1} />
      ))}
    </div>
  );
}

function WireframeRow({ event }: { event: NormalizedEvent }) {
  const json = attr<string>(event, 'mobile.wireframe.data');
  const trigger = attr<string>(event, 'mobile.wireframe.trigger') ?? 'manual';
  const screen = attr<string>(event, 'screen.name') ?? '';
  let tree: WireframeNode | null = null;
  if (json) {
    try {
      tree = JSON.parse(json) as WireframeNode;
    } catch {
      // malformed wireframe payload — render the raw string instead
    }
  }
  return (
    <div style={{ border: '1px solid #ccc', borderRadius: 4, padding: 8, marginBottom: 8 }}>
      <div style={{ fontSize: 12, color: '#666' }}>
        <strong>{formatTime(event.timestampMs)}</strong> · ui.wireframe · trigger=<code>{trigger}</code>
        {screen && <> · screen=<code>{screen}</code></>}
      </div>
      {tree ? <WireframeTree node={tree} /> : (
        <div style={{ color: '#a00', fontSize: 11 }}>
          (could not parse mobile.wireframe.data as JSON)
        </div>
      )}
    </div>
  );
}

function GenericRow({ event }: { event: NormalizedEvent }) {
  return (
    <div style={{ borderLeft: '3px solid #ddd', padding: '4px 8px', marginBottom: 4, fontSize: 12 }}>
      <span style={{ color: '#666' }}>{formatTime(event.timestampMs)}</span>
      {' '}
      <strong>{event.body || '(no body)'}</strong>
      {event.spanId && <span style={{ color: '#888' }}> · span <code>{event.spanId.slice(0, 8)}…</code></span>}
    </div>
  );
}

function ScreenshotStrip({
  events,
  onOpen,
}: {
  events: NormalizedEvent[];
  onOpen: (dataUrl: string, label: string) => void;
}) {
  const shots = events.filter((e) => e.body === 'ui.screenshot' && attr<string>(e, 'mobile.screenshot.data_url'));
  if (shots.length === 0) return null;
  return (
    <div style={{
      display: 'flex',
      gap: 8,
      overflowX: 'auto',
      padding: 8,
      background: '#f4f4f4',
      borderRadius: 4,
      marginBottom: 12,
    }}>
      {shots.map((e, i) => {
        const dataUrl = attr<string>(e, 'mobile.screenshot.data_url')!;
        const trigger = attr<string>(e, 'mobile.screenshot.trigger') ?? 'manual';
        const label = `${formatTime(e.timestampMs)} · ${trigger}`;
        return (
          <div key={i} style={{ flex: '0 0 auto', textAlign: 'center' }}>
            <img
              src={dataUrl}
              alt={label}
              onClick={() => onOpen(dataUrl, label)}
              title={label}
              style={{
                width: 96,
                height: 192,
                objectFit: 'contain',
                background: '#fff',
                border: '1px solid #ddd',
                cursor: 'zoom-in',
              }}
            />
            <div style={{ fontSize: 10, color: '#666', marginTop: 2 }}>{trigger}</div>
          </div>
        );
      })}
    </div>
  );
}

function JourneyTimeline({
  events,
  onOpenScreenshot,
}: {
  events: NormalizedEvent[];
  onOpenScreenshot: (dataUrl: string, label: string) => void;
}) {
  return (
    <div>
      <ScreenshotStrip events={events} onOpen={onOpenScreenshot} />
      {events.map((e, i) => {
        if (e.body === 'ui.screenshot') return <ScreenshotRow key={i} event={e} onOpen={onOpenScreenshot} />;
        if (e.body === 'ui.wireframe') return <WireframeRow key={i} event={e} />;
        return <GenericRow key={i} event={e} />;
      })}
    </div>
  );
}

function ScreenshotLightbox({
  src,
  label,
  onClose,
}: {
  src: string;
  label: string;
  onClose: () => void;
}) {
  return (
    <div
      onClick={onClose}
      role="dialog"
      aria-label="Full-size screenshot"
      style={{
        position: 'fixed',
        inset: 0,
        background: 'rgba(0,0,0,0.85)',
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        zIndex: 9999,
        cursor: 'zoom-out',
        padding: 24,
      }}
    >
      <img
        src={src}
        alt={label}
        onClick={(e) => e.stopPropagation()}
        style={{
          maxWidth: '90vw',
          maxHeight: '85vh',
          background: '#fff',
          border: '2px solid #fff',
          cursor: 'default',
          boxShadow: '0 8px 32px rgba(0,0,0,0.5)',
        }}
      />
      <div style={{
        color: '#fff',
        fontFamily: 'monospace',
        fontSize: 12,
        marginTop: 12,
      }}>
        {label}
        {' '}<span style={{ color: '#aaa' }}>(click outside or press Esc to close)</span>
      </div>
    </div>
  );
}

const SAMPLE_HINT = `Paste OTLP/JSON or NDJSON log records below.

Example shapes accepted:
  - Standard OTLP/JSON envelope: { "resourceLogs": [{ "scopeLogs": [...] }] }
  - Flat array: [ { "traceId": "...", "body": { "stringValue": "ui.tap" }, ... } ]
  - NDJSON (one record per line)

You can pipe \`dash0 logs export --filter "trace_id=...XXX..." --format json\`
into a file and paste it here. Captures (mobile.screenshot.*, mobile.wireframe.*)
render inline; everything else shows as a timeline entry.`;

async function fetchByTraceId(traceId: string, fromWindow: string): Promise<string> {
  const params = new URLSearchParams({ trace_id: traceId, from: fromWindow, limit: '100' });
  const resp = await fetch(`/api/v1/replay/by-trace?${params.toString()}`);
  const text = await resp.text();
  if (!resp.ok) {
    throw new Error(`Gateway returned ${resp.status}: ${text.slice(0, 200)}`);
  }
  return text;
}

export function JourneyReplay() {
  const [raw, setRaw] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [traceIdInput, setTraceIdInput] = useState('');
  const [fromWindow, setFromWindow] = useState('now-1h');
  const [fetching, setFetching] = useState(false);
  const [lightbox, setLightbox] = useState<{ src: string; label: string } | null>(null);

  // Esc closes the lightbox.
  useEffect(() => {
    if (!lightbox) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setLightbox(null);
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [lightbox]);

  const openScreenshot = (src: string, label: string) => setLightbox({ src, label });

  const handleFetch = async () => {
    const tid = traceIdInput.trim();
    if (!tid) {
      setError('enter a trace_id (32-char hex)');
      return;
    }
    setError(null);
    setFetching(true);
    try {
      const text = await fetchByTraceId(tid, fromWindow);
      setRaw(text);
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setFetching(false);
    }
  };

  const events = useMemo(() => {
    setError(null);
    if (!raw.trim()) return [];
    try {
      const parsed = parseEvents(raw);
      return parsed.sort((a, b) => a.timestampMs - b.timestampMs);
    } catch (e) {
      setError((e as Error).message);
      return [];
    }
  }, [raw]);

  const byTrace = useMemo(() => groupByTrace(events), [events]);

  return (
    <div style={{ padding: 16 }}>
      <h2 style={{ marginBottom: 8 }}>Journey Replay</h2>
      <p style={{ fontSize: 13, color: '#666', marginTop: 0 }}>
        Viewer for the User Journey + Screen/Wireframe Captures epic. Either
        fetch a journey live from Dash0 by <code>trace_id</code>, or paste a
        JSON export below for offline review.
      </p>

      <div style={{
        border: '1px solid #ddd', borderRadius: 4, padding: 12, marginBottom: 12,
        background: '#fafafa',
      }}>
        <div style={{ fontSize: 12, fontWeight: 600, marginBottom: 6 }}>Fetch from Dash0</div>
        <div style={{ display: 'flex', gap: 8, alignItems: 'center', flexWrap: 'wrap' }}>
          <input
            type="text"
            value={traceIdInput}
            onChange={(e) => setTraceIdInput(e.target.value)}
            placeholder="trace_id (32-char hex, e.g. a4f225e54cf8a4f6fdf84be3d9dfa1fb)"
            style={{ flex: '1 1 480px', padding: 6, fontFamily: 'monospace', fontSize: 12 }}
          />
          <select
            value={fromWindow}
            onChange={(e) => setFromWindow(e.target.value)}
            style={{ padding: 6, fontSize: 12 }}
          >
            <option value="now-15m">Last 15m</option>
            <option value="now-1h">Last 1h</option>
            <option value="now-6h">Last 6h</option>
            <option value="now-24h">Last 24h</option>
            <option value="now-7d">Last 7d</option>
          </select>
          <button
            onClick={handleFetch}
            disabled={fetching || !traceIdInput.trim()}
            style={{ padding: '6px 14px', fontSize: 12 }}
          >
            {fetching ? 'Fetching…' : 'Fetch'}
          </button>
        </div>
        <div style={{ fontSize: 11, color: '#888', marginTop: 6 }}>
          Gateway must be running with <code>DASH0_API_URL</code>,
          {' '}<code>DASH0_AUTH_TOKEN</code>, <code>DASH0_DATASET</code> env set.
          Token never reaches the browser.
        </div>
      </div>

      <textarea
        value={raw}
        onChange={(e) => setRaw(e.target.value)}
        placeholder={SAMPLE_HINT}
        style={{
          width: '100%',
          height: 160,
          fontFamily: 'monospace',
          fontSize: 11,
          padding: 8,
          marginBottom: 8,
        }}
      />

      {error && (
        <div style={{ color: '#a00', marginBottom: 8, fontSize: 13 }}>
          Parse error: {error}
        </div>
      )}

      {events.length > 0 && (
        <div style={{ fontSize: 13, color: '#666', marginBottom: 12 }}>
          {events.length} event{events.length === 1 ? '' : 's'} ·
          {' '}{byTrace.length} journey {byTrace.length === 1 ? 'trace' : 'traces'}
        </div>
      )}

      {byTrace.map(([traceId, traceEvents]) => (
        <details key={traceId} open style={{ marginBottom: 16 }}>
          <summary style={{ cursor: 'pointer', fontWeight: 600, marginBottom: 8 }}>
            Trace <code>{traceId}</code> · {traceEvents.length} events
          </summary>
          <JourneyTimeline events={traceEvents} onOpenScreenshot={openScreenshot} />
        </details>
      ))}

      {lightbox && (
        <ScreenshotLightbox
          src={lightbox.src}
          label={lightbox.label}
          onClose={() => setLightbox(null)}
        />
      )}
    </div>
  );
}
