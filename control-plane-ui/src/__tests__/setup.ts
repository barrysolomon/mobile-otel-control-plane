import '@testing-library/jest-dom/vitest';

// React Flow (used by WorkflowBuilder) calls ResizeObserver during effects.
// jsdom doesn't ship one — stub a minimal no-op so the canvas mount doesn't
// throw and the rest of the App can render in tests.
class ResizeObserverPolyfill {
  observe() {}
  unobserve() {}
  disconnect() {}
}
if (typeof globalThis.ResizeObserver === 'undefined') {
  // @ts-expect-error — adding stub to global
  globalThis.ResizeObserver = ResizeObserverPolyfill
}

// React Flow reads getBoundingClientRect with non-zero dimensions to
// initialise its viewport. jsdom returns all-zero, producing warnings.
// Stub something sensible.
if (typeof window !== 'undefined') {
  const proto = Element.prototype as Element & { getBoundingClientRect: () => DOMRect }
  const original = proto.getBoundingClientRect
  proto.getBoundingClientRect = function () {
    const r = original.call(this) as DOMRect
    if (r.width === 0 && r.height === 0) {
      return { x: 0, y: 0, top: 0, left: 0, right: 800, bottom: 600, width: 800, height: 600, toJSON: () => ({}) } as DOMRect
    }
    return r
  }

  if (!window.matchMedia) {
    window.matchMedia = (() => ({
      matches: false,
      media: '',
      onchange: null,
      addListener: () => {},
      removeListener: () => {},
      addEventListener: () => {},
      removeEventListener: () => {},
      dispatchEvent: () => false,
    })) as unknown as typeof window.matchMedia
  }
}
