/**
 * Text matcher for `getByText` over text split across elements (a sentence
 * whose names are map links): matches the innermost element whose whole text
 * content equals the string or matches the pattern.
 */
export function withText(expected: string | RegExp) {
  const matches = (text: string) =>
    typeof expected === 'string' ? text === expected : expected.test(text)
  return (_content: string, element: Element | null): boolean => {
    if (!element || !matches(element.textContent ?? '')) return false
    return Array.from(element.children).every(
      (child) => !matches(child.textContent ?? ''),
    )
  }
}
