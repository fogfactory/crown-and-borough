import '@testing-library/jest-dom/vitest'

import { cleanup } from '@testing-library/react'
import { afterEach } from 'vitest'

afterEach(cleanup)

class ResizeObserverStub {
  disconnect() {}

  observe() {}

  unobserve() {}
}

globalThis.ResizeObserver = ResizeObserverStub

Object.defineProperty(SVGElement.prototype, 'getBoundingClientRect', {
  configurable: true,
  value: () => ({
    bottom: 700,
    height: 700,
    left: 0,
    right: 1000,
    top: 0,
    width: 1000,
    x: 0,
    y: 0,
    toJSON: () => ({}),
  }),
})

SVGElement.prototype.setPointerCapture = () => undefined
SVGElement.prototype.releasePointerCapture = () => undefined
SVGElement.prototype.hasPointerCapture = () => true
