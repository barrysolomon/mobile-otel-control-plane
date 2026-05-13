import { describe, it, expect } from 'vitest';
import {
  parseEvents,
  normalizeLogRecord,
  normalizeFlat,
  groupByTrace,
  attr,
} from '../utils/journeyParser';
import type { NormalizedEvent } from '../utils/journeyParser';
import {
  mkOtlpEnvelope,
  mkLogRecord,
  mkFlatRecord,
  mkScreenshotRecord,
  mkWireframeRecord,
  SAMPLE_TRACE_ID,
} from './fixtures/journeyFixtures';

// ---------------------------------------------------------------------------
// parseEvents — OTLP/JSON envelope
// ---------------------------------------------------------------------------
describe('parseEvents: OTLP/JSON envelope', () => {
  it('parses a standard resourceLogs envelope', () => {
    const envelope = mkOtlpEnvelope([
      mkLogRecord('ui.screen_view', SAMPLE_TRACE_ID, '1000000000000000'),
      mkLogRecord('ui.tap', SAMPLE_TRACE_ID, '2000000000000000'),
    ]);
    const events = parseEvents(JSON.stringify(envelope));
    expect(events).toHaveLength(2);
    expect(events[0].body).toBe('ui.screen_view');
    expect(events[1].body).toBe('ui.tap');
    expect(events[0].traceId).toBe(SAMPLE_TRACE_ID);
  });

  it('extracts attributes from OTLP key-value pairs', () => {
    const record = mkLogRecord('ui.tap', SAMPLE_TRACE_ID, '1000000000000000', [
      { key: 'screen.name', value: { stringValue: 'BookingScreen' } },
      { key: 'ui.tap.x', value: { intValue: 195 } },
      { key: 'ui.tap.enabled', value: { boolValue: true } },
      { key: 'ui.tap.pressure', value: { doubleValue: 0.75 } },
    ]);
    const envelope = mkOtlpEnvelope([record]);
    const events = parseEvents(JSON.stringify(envelope));
    expect(events[0].attributes['screen.name']).toBe('BookingScreen');
    expect(events[0].attributes['ui.tap.x']).toBe(195);
    expect(events[0].attributes['ui.tap.enabled']).toBe(true);
    expect(events[0].attributes['ui.tap.pressure']).toBe(0.75);
  });

  it('converts timeUnixNano to milliseconds', () => {
    const envelope = mkOtlpEnvelope([
      mkLogRecord('test', 'abc', '1715000000123456789'),
    ]);
    const events = parseEvents(JSON.stringify(envelope));
    expect(events[0].timestampMs).toBe(1715000000123);
  });

  it('falls back to observedTimeUnixNano when timeUnixNano is missing', () => {
    const record = {
      observedTimeUnixNano: '1715000000999000000',
      body: { stringValue: 'test' },
      traceId: 'abc',
      spanId: 'def',
      attributes: [],
    };
    const event = normalizeLogRecord(record);
    expect(event.timestampMs).toBe(1715000000999);
  });

  it('handles multiple resourceLogs and scopeLogs', () => {
    const envelope = {
      resourceLogs: [
        { scopeLogs: [{ logRecords: [mkLogRecord('a', 't1', '1000000000000000')] }] },
        { scopeLogs: [
          { logRecords: [mkLogRecord('b', 't1', '2000000000000000')] },
          { logRecords: [mkLogRecord('c', 't1', '3000000000000000')] },
        ] },
      ],
    };
    const events = parseEvents(JSON.stringify(envelope));
    expect(events).toHaveLength(3);
    expect(events.map(e => e.body)).toEqual(['a', 'b', 'c']);
  });

  it('handles empty envelope gracefully', () => {
    expect(parseEvents(JSON.stringify({ resourceLogs: [] }))).toEqual([]);
    expect(parseEvents(JSON.stringify({ resourceLogs: [{ scopeLogs: [] }] }))).toEqual([]);
  });
});

// ---------------------------------------------------------------------------
// parseEvents — flat array
// ---------------------------------------------------------------------------
describe('parseEvents: flat array', () => {
  it('parses a flat array of OTLP log records', () => {
    const records = [
      mkLogRecord('ui.screen_view', SAMPLE_TRACE_ID, '1000000000000000'),
      mkLogRecord('ui.tap', SAMPLE_TRACE_ID, '2000000000000000'),
    ];
    const events = parseEvents(JSON.stringify(records));
    expect(events).toHaveLength(2);
    expect(events[0].body).toBe('ui.screen_view');
  });

  it('parses a flat array of Dash0 CLI-style records', () => {
    const records = [
      mkFlatRecord('ui.tap', SAMPLE_TRACE_ID, '2026-05-07T10:00:00Z'),
      mkFlatRecord('app.foreground', SAMPLE_TRACE_ID, '2026-05-07T10:00:01Z'),
    ];
    const events = parseEvents(JSON.stringify(records));
    expect(events).toHaveLength(2);
    expect(events[0].body).toBe('ui.tap');
    expect(events[1].body).toBe('app.foreground');
  });

  it('handles mixed OTLP and flat records in one array', () => {
    const records = [
      mkLogRecord('ui.tap', 'trace-a', '1000000000000000'),
      mkFlatRecord('app.foreground', 'trace-a', '2026-05-07T10:00:00Z'),
    ];
    const events = parseEvents(JSON.stringify(records));
    expect(events).toHaveLength(2);
    expect(events[0].body).toBe('ui.tap');
    expect(events[1].body).toBe('app.foreground');
  });
});

// ---------------------------------------------------------------------------
// parseEvents — NDJSON
// ---------------------------------------------------------------------------
describe('parseEvents: NDJSON', () => {
  it('parses newline-delimited JSON records', () => {
    const ndjson = [
      JSON.stringify(mkFlatRecord('ui.tap', 'trace-a', '2026-05-07T10:00:00Z')),
      JSON.stringify(mkFlatRecord('ui.screen_view', 'trace-a', '2026-05-07T10:00:01Z')),
    ].join('\n');
    const events = parseEvents(ndjson);
    expect(events).toHaveLength(2);
  });

  it('skips malformed lines without crashing', () => {
    const ndjson = [
      JSON.stringify(mkFlatRecord('ui.tap', 'trace-a', '2026-05-07T10:00:00Z')),
      'this is not json {{{',
      '',
      JSON.stringify(mkFlatRecord('ui.screen_view', 'trace-a', '2026-05-07T10:00:01Z')),
    ].join('\n');
    const events = parseEvents(ndjson);
    expect(events).toHaveLength(2);
  });

  it('handles NDJSON with trailing newlines', () => {
    const ndjson = JSON.stringify(mkFlatRecord('ui.tap', 'trace-a', '2026-05-07T10:00:00Z')) + '\n\n';
    const events = parseEvents(ndjson);
    expect(events).toHaveLength(1);
  });
});

// ---------------------------------------------------------------------------
// parseEvents — edge cases
// ---------------------------------------------------------------------------
describe('parseEvents: edge cases', () => {
  it('returns empty array for empty string', () => {
    expect(parseEvents('')).toEqual([]);
    expect(parseEvents('   ')).toEqual([]);
  });

  it('returns empty array for whitespace-only input', () => {
    expect(parseEvents('\n\n\n')).toEqual([]);
  });

  it('handles records with missing fields', () => {
    const record = { body: { stringValue: 'test' } };
    const envelope = mkOtlpEnvelope([record as never]);
    const events = parseEvents(JSON.stringify(envelope));
    expect(events[0].traceId).toBe('');
    expect(events[0].spanId).toBe('');
    expect(events[0].timestampMs).toBe(0);
  });
});

// ---------------------------------------------------------------------------
// normalizeFlat — Dash0 CLI output shapes
// ---------------------------------------------------------------------------
describe('normalizeFlat: Dash0 CLI output', () => {
  it('handles trace_id (snake_case) from dash0 CLI', () => {
    const event = normalizeFlat({
      body: 'ui.tap',
      trace_id: 'abc123',
      span_id: 'def456',
      timestamp: '2026-05-07T10:00:00Z',
    });
    expect(event.traceId).toBe('abc123');
    expect(event.spanId).toBe('def456');
  });

  it('handles traceId (camelCase) from OTLP JSON', () => {
    const event = normalizeFlat({
      body: 'ui.tap',
      traceId: 'abc123',
      spanId: 'def456',
      timestamp: '2026-05-07T10:00:00Z',
    });
    expect(event.traceId).toBe('abc123');
    expect(event.spanId).toBe('def456');
  });

  it('parses ISO timestamp string', () => {
    const event = normalizeFlat({
      body: 'test',
      timestamp: '2026-05-07T12:30:00Z',
    });
    expect(event.timestampMs).toBe(Date.parse('2026-05-07T12:30:00Z'));
  });

  it('handles numeric timestamp (epoch ms)', () => {
    const event = normalizeFlat({
      body: 'test',
      timestamp: 1715000000000,
    });
    expect(event.timestampMs).toBe(1715000000000);
  });

  it('uses `time` field as fallback', () => {
    const event = normalizeFlat({
      body: 'test',
      time: '2026-05-07T12:30:00Z',
    });
    expect(event.timestampMs).toBe(Date.parse('2026-05-07T12:30:00Z'));
  });
});

// ---------------------------------------------------------------------------
// Screenshot and wireframe attribute extraction
// ---------------------------------------------------------------------------
describe('capture event attributes', () => {
  it('screenshot record carries data_url, trigger, dimensions', () => {
    const envelope = mkOtlpEnvelope([mkScreenshotRecord(SAMPLE_TRACE_ID, '1000000000000000')]);
    const events = parseEvents(JSON.stringify(envelope));
    const e = events[0];
    expect(e.body).toBe('ui.screenshot');
    expect(attr<string>(e, 'mobile.screenshot.data_url')).toMatch(/^data:image\/png;base64,/);
    expect(attr<string>(e, 'mobile.screenshot.trigger')).toBe('journey_start');
    expect(attr<number>(e, 'mobile.screenshot.width')).toBe(390);
    expect(attr<number>(e, 'mobile.screenshot.height')).toBe(844);
    expect(attr<number>(e, 'mobile.screenshot.size_bytes')).toBe(2048);
  });

  it('wireframe record carries JSON data and trigger', () => {
    const envelope = mkOtlpEnvelope([mkWireframeRecord(SAMPLE_TRACE_ID, '1000000000000000')]);
    const events = parseEvents(JSON.stringify(envelope));
    const e = events[0];
    expect(e.body).toBe('ui.wireframe');
    expect(attr<string>(e, 'mobile.wireframe.trigger')).toBe('screen_view');
    const json = attr<string>(e, 'mobile.wireframe.data');
    expect(json).toBeTruthy();
    const tree = JSON.parse(json!);
    expect(tree.type).toBe('UIWindow');
    expect(tree.children).toBeDefined();
  });
});

// ---------------------------------------------------------------------------
// groupByTrace
// ---------------------------------------------------------------------------
describe('groupByTrace', () => {
  it('groups events by traceId', () => {
    const events: NormalizedEvent[] = [
      { timestampMs: 1, body: 'a', traceId: 'trace-1', spanId: 's1', attributes: {} },
      { timestampMs: 2, body: 'b', traceId: 'trace-2', spanId: 's2', attributes: {} },
      { timestampMs: 3, body: 'c', traceId: 'trace-1', spanId: 's3', attributes: {} },
    ];
    const groups = groupByTrace(events);
    expect(groups).toHaveLength(2);
    expect(groups[0][0]).toBe('trace-1');
    expect(groups[0][1]).toHaveLength(2);
    expect(groups[1][0]).toBe('trace-2');
    expect(groups[1][1]).toHaveLength(1);
  });

  it('sorts trace groups by earliest timestamp', () => {
    const events: NormalizedEvent[] = [
      { timestampMs: 5000, body: 'late', traceId: 'trace-late', spanId: 's1', attributes: {} },
      { timestampMs: 1000, body: 'early', traceId: 'trace-early', spanId: 's2', attributes: {} },
    ];
    const groups = groupByTrace(events);
    expect(groups[0][0]).toBe('trace-early');
    expect(groups[1][0]).toBe('trace-late');
  });

  it('assigns "(no trace_id)" for events without traceId', () => {
    const events: NormalizedEvent[] = [
      { timestampMs: 1, body: 'orphan', traceId: '', spanId: 's1', attributes: {} },
    ];
    const groups = groupByTrace(events);
    expect(groups[0][0]).toBe('(no trace_id)');
  });

  it('returns empty array for no events', () => {
    expect(groupByTrace([])).toEqual([]);
  });
});

// ---------------------------------------------------------------------------
// attr helper
// ---------------------------------------------------------------------------
describe('attr helper', () => {
  const event: NormalizedEvent = {
    timestampMs: 0,
    body: 'test',
    traceId: '',
    spanId: '',
    attributes: { 'screen.name': 'Home', count: 42, enabled: true },
  };

  it('returns string attribute', () => {
    expect(attr<string>(event, 'screen.name')).toBe('Home');
  });

  it('returns number attribute', () => {
    expect(attr<number>(event, 'count')).toBe(42);
  });

  it('returns undefined for missing key', () => {
    expect(attr(event, 'nonexistent')).toBeUndefined();
  });
});

// ---------------------------------------------------------------------------
// Realistic Dash0 journey — full booking flow
// ---------------------------------------------------------------------------
describe('realistic Dash0 booking journey', () => {
  it('parses a complete journey with screen views, taps, captures, and lifecycle events', () => {
    const baseNano = 1715000000000000000n;
    const records = [
      mkLogRecord('ui.screen_view', SAMPLE_TRACE_ID, String(baseNano), [
        { key: 'screen.name', value: { stringValue: 'CalendarScreen' } },
      ]),
      mkScreenshotRecord(SAMPLE_TRACE_ID, String(baseNano + 100000000n)),
      mkWireframeRecord(SAMPLE_TRACE_ID, String(baseNano + 150000000n)),
      mkLogRecord('ui.tap', SAMPLE_TRACE_ID, String(baseNano + 1200000000n), [
        { key: 'screen.name', value: { stringValue: 'CalendarScreen' } },
        { key: 'ui.tap.target', value: { stringValue: 'date_cell' } },
      ]),
      mkLogRecord('ui.screen_view', SAMPLE_TRACE_ID, String(baseNano + 2500000000n), [
        { key: 'screen.name', value: { stringValue: 'BookingScreen' } },
      ]),
      mkLogRecord('ui.tap', SAMPLE_TRACE_ID, String(baseNano + 7000000000n), [
        { key: 'ui.tap.target', value: { stringValue: 'confirm_btn' } },
      ]),
      mkLogRecord('app.foreground', SAMPLE_TRACE_ID, String(baseNano + 7500000000n)),
      mkScreenshotRecord(SAMPLE_TRACE_ID, String(baseNano + 8000000000n)),
    ];

    const envelope = mkOtlpEnvelope(records);
    const events = parseEvents(JSON.stringify(envelope));

    expect(events).toHaveLength(8);

    const screenViews = events.filter(e => e.body === 'ui.screen_view');
    expect(screenViews).toHaveLength(2);
    expect(attr(screenViews[0], 'screen.name')).toBe('CalendarScreen');
    expect(attr(screenViews[1], 'screen.name')).toBe('BookingScreen');

    const screenshots = events.filter(e => e.body === 'ui.screenshot');
    expect(screenshots).toHaveLength(2);

    const wireframes = events.filter(e => e.body === 'ui.wireframe');
    expect(wireframes).toHaveLength(1);

    const taps = events.filter(e => e.body === 'ui.tap');
    expect(taps).toHaveLength(2);
    expect(attr(taps[1], 'ui.tap.target')).toBe('confirm_btn');

    const groups = groupByTrace(events);
    expect(groups).toHaveLength(1);
    expect(groups[0][0]).toBe(SAMPLE_TRACE_ID);
    expect(groups[0][1]).toHaveLength(8);
  });
});
