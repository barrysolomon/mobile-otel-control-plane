import type { LogRecord, LogAttribute } from '../../utils/journeyParser';

export const SAMPLE_TRACE_ID = 'a4f225e54cf8a4f6fdf84be3d9dfa1fb';

export const SAMPLE_WIREFRAME_JSON = JSON.stringify({
  type: 'UIWindow',
  bounds: { left: 0, top: 0, right: 390, bottom: 844 },
  children: [
    {
      type: 'UINavigationController',
      children: [
        { type: 'UILabel', text: 'Book a Room', bounds: { left: 16, top: 60, right: 374, bottom: 90 } },
        { type: 'UIButton', text: 'Confirm', id: 'confirm_btn', clickable: true, bounds: { left: 100, top: 700, right: 290, bottom: 744 } },
      ],
    },
  ],
});

export function mkLogRecord(
  body: string,
  traceId: string,
  timeUnixNano: string,
  attributes: LogAttribute[] = [],
): LogRecord {
  return {
    timeUnixNano,
    body: { stringValue: body },
    traceId,
    spanId: timeUnixNano.slice(0, 16),
    attributes,
  };
}

export function mkFlatRecord(
  body: string,
  traceId: string,
  timestamp: string,
  attributes: Record<string, unknown> = {},
) {
  return {
    body,
    trace_id: traceId,
    span_id: traceId.slice(0, 16),
    timestamp,
    attributes,
  };
}

export function mkScreenshotRecord(traceId: string, timeUnixNano: string): LogRecord {
  return mkLogRecord('ui.screenshot', traceId, timeUnixNano, [
    { key: 'mobile.screenshot.data_url', value: { stringValue: 'data:image/png;base64,iVBORw0KGgoAAAANSUhEUg==' } },
    { key: 'mobile.screenshot.trigger', value: { stringValue: 'journey_start' } },
    { key: 'mobile.screenshot.width', value: { intValue: 390 } },
    { key: 'mobile.screenshot.height', value: { intValue: 844 } },
    { key: 'mobile.screenshot.size_bytes', value: { intValue: 2048 } },
  ]);
}

export function mkWireframeRecord(traceId: string, timeUnixNano: string): LogRecord {
  return mkLogRecord('ui.wireframe', traceId, timeUnixNano, [
    { key: 'mobile.wireframe.data', value: { stringValue: SAMPLE_WIREFRAME_JSON } },
    { key: 'mobile.wireframe.trigger', value: { stringValue: 'screen_view' } },
  ]);
}

export function mkOtlpEnvelope(logRecords: LogRecord[]) {
  return {
    resourceLogs: [
      {
        scopeLogs: [
          { logRecords },
        ],
      },
    ],
  };
}
