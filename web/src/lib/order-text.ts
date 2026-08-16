function escapeRegExp(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

export function addNobleHeader(nobleCode: string, text: string): string {
  const lines = text.replace(/\r\n/g, '\n').split('\n')
  const firstContentIndex = lines.findIndex((line) => line.split('#', 1)[0].trim() !== '')
  if (firstContentIndex >= 0) {
    const headerPattern = new RegExp(`^${escapeRegExp(nobleCode)}(?:\\s+#.*)?$`, 'i')
    if (headerPattern.test(lines[firstContentIndex].trim())) {
      lines.splice(firstContentIndex, 1)
    }
  }
  return `${nobleCode}\n${lines.join('\n')}`.trimEnd()
}

export function hasChainContent(nobleCode: string, text: string): boolean {
  return text.replace(new RegExp(`^${escapeRegExp(nobleCode)}\\s*`), '').trim() !== ''
}
