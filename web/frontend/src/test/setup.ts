import '@testing-library/jest-dom';

// Mock ResizeObserver for tests
global.ResizeObserver = class ResizeObserver {
  observe() {}
  unobserve() {}
  disconnect() {}
};

// Mock IntersectionObserver for tests
global.IntersectionObserver = class IntersectionObserver {
  constructor(callback: IntersectionObserverCallback, options?: IntersectionObserverInit) {
    // Immediately trigger the callback with a mock entry
    setTimeout(() => {
      callback([{
        isIntersecting: true,
        target: document.createElement('div'),
        intersectionRatio: 1,
        boundingClientRect: {} as DOMRectReadOnly,
        intersectionRect: {} as DOMRectReadOnly,
        rootBounds: {} as DOMRectReadOnly,
        time: Date.now(),
      }], this);
    }, 0);
  }
  
  observe() {}
  unobserve() {}
  disconnect() {}
};