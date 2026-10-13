import { useLayoutEffect, useState, type RefObject } from 'react'

/** Rendered size of an element in CSS pixels, kept current on resize. */
export function useElementSize(ref: RefObject<Element | null>): {
  width: number
  height: number
} {
  const [size, setSize] = useState({ width: 0, height: 0 })
  useLayoutEffect(() => {
    const element = ref.current
    if (!element) return
    const measure = () => {
      const { width, height } = element.getBoundingClientRect()
      setSize((current) =>
        current.width === width && current.height === height
          ? current
          : { width, height },
      )
    }
    measure()
    const observer = new ResizeObserver(measure)
    observer.observe(element)
    return () => observer.disconnect()
  }, [ref])
  return size
}
