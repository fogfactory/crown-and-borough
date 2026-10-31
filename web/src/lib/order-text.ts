function escapeRegExp(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

/**
 * Index (0-based) of the line of `text` that is the typed header of the noble:
 * its first content line when it only holds the noble code, else -1.
 */
export function typedHeaderIndex(nobleCode: string, text: string): number {
  const lines = text.replace(/\r\n/g, '\n').split('\n')
  const firstContentIndex = lines.findIndex((line) => line.split('#', 1)[0].trim() !== '')
  if (firstContentIndex < 0) return -1
  const headerPattern = new RegExp(`^${escapeRegExp(nobleCode)}(?:\\s+#.*)?$`, 'i')
  return headerPattern.test(lines[firstContentIndex].trim()) ? firstContentIndex : -1
}

export function addNobleHeader(nobleCode: string, text: string): string {
  const lines = text.replace(/\r\n/g, '\n').split('\n')
  const headerIndex = typedHeaderIndex(nobleCode, text)
  if (headerIndex >= 0) lines.splice(headerIndex, 1)
  return `${nobleCode}\n${lines.join('\n')}`.trimEnd()
}

export function hasChainContent(nobleCode: string, text: string): boolean {
  return text.replace(new RegExp(`^${escapeRegExp(nobleCode)}\\s*`), '').trim() !== ''
}

export function stripNobleHeader(nobleCode: string, text: string): string {
  const lines = text.replace(/\r\n/g, '\n').split('\n')
  const firstContentIndex = lines.findIndex((line) => line.split('#', 1)[0].trim() !== '')
  if (firstContentIndex >= 0) {
    const headerPattern = new RegExp(`^${escapeRegExp(nobleCode)}(?:\\s+#.*)?$`, 'i')
    if (headerPattern.test(lines[firstContentIndex].trim())) {
      lines.splice(firstContentIndex, 1)
    }
  }
  return lines.join('\n').trim()
}
