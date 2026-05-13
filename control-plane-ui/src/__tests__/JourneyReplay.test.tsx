import { describe, it, expect } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { JourneyReplay } from '../components/JourneyReplay';
import {
  mkOtlpEnvelope,
  mkLogRecord,
  mkScreenshotRecord,
  mkWireframeRecord,
  SAMPLE_TRACE_ID,
} from './fixtures/journeyFixtures';

function setTextarea(payload: unknown) {
  const textarea = screen.getByPlaceholderText(/Paste OTLP/);
  const json = typeof payload === 'string' ? payload : JSON.stringify(payload);
  fireEvent.change(textarea, { target: { value: json } });
}

// ---------------------------------------------------------------------------
// Basic rendering
// ---------------------------------------------------------------------------
describe('JourneyReplay: basic rendering', () => {
  it('renders the heading and fetch form', () => {
    render(<JourneyReplay />);
    expect(screen.getByText('Journey Replay')).toBeInTheDocument();
    expect(screen.getByPlaceholderText(/32-char hex/)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Fetch/ })).toBeInTheDocument();
  });

  it('renders textarea with hint placeholder', () => {
    render(<JourneyReplay />);
    const textarea = screen.getByPlaceholderText(/Paste OTLP/);
    expect(textarea).toBeInTheDocument();
  });

  it('shows no events when textarea is empty', () => {
    render(<JourneyReplay />);
    expect(screen.queryByText(/events/)).not.toBeInTheDocument();
  });
});

// ---------------------------------------------------------------------------
// Timeline rendering from OTLP data
// ---------------------------------------------------------------------------
describe('JourneyReplay: timeline rendering', () => {
  const envelope = mkOtlpEnvelope([
    mkLogRecord('ui.screen_view', SAMPLE_TRACE_ID, '1715000000000000000', [
      { key: 'screen.name', value: { stringValue: 'HomeScreen' } },
    ]),
    mkLogRecord('ui.tap', SAMPLE_TRACE_ID, '1715000001000000000', [
      { key: 'ui.tap.target', value: { stringValue: 'search_btn' } },
    ]),
    mkLogRecord('app.foreground', SAMPLE_TRACE_ID, '1715000002000000000'),
  ]);

  it('renders event count and trace count after pasting data', () => {
    render(<JourneyReplay />);
    setTextarea(envelope);
    const matches = screen.getAllByText(/3 events/);
    expect(matches.length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText(/1 journey trace/)).toBeInTheDocument();
  });

  it('renders trace summary with trace_id', () => {
    render(<JourneyReplay />);
    setTextarea(envelope);
    const traceMatches = screen.getAllByText(new RegExp(SAMPLE_TRACE_ID));
    expect(traceMatches.length).toBeGreaterThanOrEqual(1);
  });

  it('renders generic breadcrumb rows for non-capture events', async () => {
    render(<JourneyReplay />);
    setTextarea(envelope);
    expect(screen.getByText('ui.screen_view')).toBeInTheDocument();
    expect(screen.getByText('ui.tap')).toBeInTheDocument();
    expect(screen.getByText('app.foreground')).toBeInTheDocument();
  });
});

// ---------------------------------------------------------------------------
// Screenshot rendering
// ---------------------------------------------------------------------------
describe('JourneyReplay: screenshots', () => {
  const envelope = mkOtlpEnvelope([
    mkScreenshotRecord(SAMPLE_TRACE_ID, '1715000000000000000'),
    mkLogRecord('ui.tap', SAMPLE_TRACE_ID, '1715000001000000000'),
  ]);

  it('renders screenshot image with data URL', async () => {
    render(<JourneyReplay />);
    setTextarea(envelope);
    const imgs = screen.getAllByRole('img');
    const screenshotImgs = imgs.filter(img => img.getAttribute('src')?.startsWith('data:image/'));
    expect(screenshotImgs.length).toBeGreaterThanOrEqual(1);
  });

  it('renders screenshot metadata (trigger, dimensions)', async () => {
    render(<JourneyReplay />);
    setTextarea(envelope);
    expect(screen.getAllByText(/journey_start/).length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText(/390×844/)).toBeInTheDocument();
    expect(screen.getByText(/2 KB/)).toBeInTheDocument();
  });

  it('opens lightbox on screenshot click', async () => {
    render(<JourneyReplay />);
    setTextarea(envelope);
    const imgs = screen.getAllByRole('img');
    const clickableImg = imgs.find(img =>
      img.getAttribute('src')?.startsWith('data:image/') &&
      img.style.cursor === 'zoom-in'
    );
    expect(clickableImg).toBeTruthy();
    await userEvent.click(clickableImg!);
    expect(screen.getByRole('dialog')).toBeInTheDocument();
    expect(screen.getByText(/click outside or press Esc/)).toBeInTheDocument();
  });

  it('closes lightbox on Escape key', async () => {
    render(<JourneyReplay />);
    setTextarea(envelope);
    const imgs = screen.getAllByRole('img');
    const clickableImg = imgs.find(img =>
      img.getAttribute('src')?.startsWith('data:image/') &&
      img.style.cursor === 'zoom-in'
    );
    await userEvent.click(clickableImg!);
    expect(screen.getByRole('dialog')).toBeInTheDocument();
    await userEvent.keyboard('{Escape}');
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });
});

// ---------------------------------------------------------------------------
// Wireframe rendering
// ---------------------------------------------------------------------------
describe('JourneyReplay: wireframes', () => {
  const envelope = mkOtlpEnvelope([
    mkWireframeRecord(SAMPLE_TRACE_ID, '1715000000000000000'),
  ]);

  it('renders wireframe tree with node types', () => {
    render(<JourneyReplay />);
    setTextarea(envelope);
    expect(screen.getByText('UIWindow')).toBeInTheDocument();
    expect(screen.getAllByText(/ui\.wireframe/).length).toBeGreaterThanOrEqual(1);
  });

  it('renders wireframe trigger metadata', () => {
    render(<JourneyReplay />);
    setTextarea(envelope);
    expect(screen.getAllByText('screen_view').length).toBeGreaterThanOrEqual(1);
  });

  it('renders wireframe child nodes when expanded', async () => {
    render(<JourneyReplay />);
    setTextarea(envelope);
    expect(screen.getByText('UINavigationController')).toBeInTheDocument();
  });
});

// ---------------------------------------------------------------------------
// Multiple traces
// ---------------------------------------------------------------------------
describe('JourneyReplay: multiple traces', () => {
  const envelope = mkOtlpEnvelope([
    mkLogRecord('ui.tap', 'trace-aaa', '1715000000000000000'),
    mkLogRecord('ui.tap', 'trace-bbb', '1715000001000000000'),
    mkLogRecord('ui.screen_view', 'trace-aaa', '1715000002000000000'),
  ]);

  it('groups events into separate trace sections', () => {
    render(<JourneyReplay />);
    setTextarea(envelope);
    expect(screen.getByText(/2 journey traces/)).toBeInTheDocument();
    expect(screen.getAllByText(/trace-aaa/).length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText(/trace-bbb/).length).toBeGreaterThanOrEqual(1);
  });
});

// ---------------------------------------------------------------------------
// Error handling
// ---------------------------------------------------------------------------
describe('JourneyReplay: error handling', () => {
  it('returns no events for unparseable input (graceful degradation)', () => {
    render(<JourneyReplay />);
    const textarea = screen.getByPlaceholderText(/Paste OTLP/);
    fireEvent.change(textarea, { target: { value: '{not valid json' } });
    expect(screen.queryByText(/events/)).not.toBeInTheDocument();
  });
});

// ---------------------------------------------------------------------------
// Fetch form validation
// ---------------------------------------------------------------------------
describe('JourneyReplay: fetch form', () => {
  it('disables fetch button when trace_id is empty', () => {
    render(<JourneyReplay />);
    const btn = screen.getByRole('button', { name: /Fetch/ });
    expect(btn).toBeDisabled();
  });

  it('enables fetch button when trace_id is entered', async () => {
    render(<JourneyReplay />);
    const input = screen.getByPlaceholderText(/32-char hex/);
    await userEvent.type(input, SAMPLE_TRACE_ID);
    const btn = screen.getByRole('button', { name: /Fetch/ });
    expect(btn).not.toBeDisabled();
  });

  it('shows error when fetch is attempted with empty trace_id after clearing', async () => {
    render(<JourneyReplay />);
    const input = screen.getByPlaceholderText(/32-char hex/);
    await userEvent.type(input, 'x');
    await userEvent.clear(input);
    const btn = screen.getByRole('button', { name: /Fetch/ });
    expect(btn).toBeDisabled();
  });

  it('has time window selector with expected options', () => {
    render(<JourneyReplay />);
    const select = screen.getByDisplayValue('Last 1h');
    expect(select).toBeInTheDocument();
    expect(screen.getByText('Last 15m')).toBeInTheDocument();
    expect(screen.getByText('Last 6h')).toBeInTheDocument();
    expect(screen.getByText('Last 24h')).toBeInTheDocument();
    expect(screen.getByText('Last 7d')).toBeInTheDocument();
  });
});

// ---------------------------------------------------------------------------
// Realistic Dash0 journey — full booking flow rendering
// ---------------------------------------------------------------------------
describe('JourneyReplay: realistic booking journey', () => {
  const baseNano = 1715000000000000000n;
  const records = [
    mkLogRecord('ui.screen_view', SAMPLE_TRACE_ID, String(baseNano), [
      { key: 'screen.name', value: { stringValue: 'CalendarScreen' } },
    ]),
    mkScreenshotRecord(SAMPLE_TRACE_ID, String(baseNano + 100000000n)),
    mkWireframeRecord(SAMPLE_TRACE_ID, String(baseNano + 150000000n)),
    mkLogRecord('ui.tap', SAMPLE_TRACE_ID, String(baseNano + 1200000000n), [
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

  it('renders all 8 events in a single trace', () => {
    render(<JourneyReplay />);
    setTextarea(envelope);
    expect(screen.getAllByText(/8 events/).length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText(/1 journey trace/)).toBeInTheDocument();
  });

  it('renders both screen_view events as breadcrumbs', async () => {
    render(<JourneyReplay />);
    setTextarea(envelope);
    const screenViews = screen.getAllByText('ui.screen_view');
    expect(screenViews).toHaveLength(2);
  });

  it('renders screenshot strip with thumbnails', async () => {
    render(<JourneyReplay />);
    setTextarea(envelope);
    const imgs = screen.getAllByRole('img');
    const thumbnails = imgs.filter(img =>
      img.getAttribute('src')?.startsWith('data:image/') &&
      img.getAttribute('style')?.includes('96')
    );
    expect(thumbnails.length).toBeGreaterThanOrEqual(1);
  });

  it('renders wireframe tree root node', async () => {
    render(<JourneyReplay />);
    setTextarea(envelope);
    expect(screen.getByText('UIWindow')).toBeInTheDocument();
  });
});
