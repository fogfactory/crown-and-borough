import { Fragment, isValidElement, type ReactNode } from 'react'

import type { MessageKey, Translate } from '@/i18n/messages'

/** Values of a rich message: text, numbers, or nodes such as map links. */
export type RichValues = Record<string, ReactNode>

// Private-use marker framing a placeholder index; never typed by a player.
const MARK = ''

/**
 * Translates `key` like `t`, but lets placeholder values be React nodes (a
 * territory link inside a report sentence). Text and numbers are interpolated
 * as usual; each node is spliced in at its placeholder.
 */
export function richTranslate(
  t: Translate,
  key: MessageKey,
  values: RichValues = {},
): ReactNode {
  const nodes: ReactNode[] = []
  const plain: Record<string, string | number> = {}
  for (const [name, value] of Object.entries(values)) {
    if (typeof value === 'string' || typeof value === 'number') {
      plain[name] = value
    } else if (isValidElement(value) || Array.isArray(value)) {
      plain[name] = `${MARK}${nodes.length}${MARK}`
      nodes.push(value)
    } else {
      plain[name] = value == null || typeof value === 'boolean' ? '' : String(value)
    }
  }
  const text = t(key, plain)
  if (nodes.length === 0) return text
  return text
    .split(MARK)
    .map((part, index) =>
      index % 2 === 1 ? <Fragment key={index}>{nodes[Number(part)]}</Fragment> : part,
    )
}

/** Joins nodes with a separator, like `Array.join` for text. */
export function joinNodes(items: ReactNode[], separator: ReactNode): ReactNode[] {
  return items.flatMap((item, index) =>
    index === 0
      ? [<Fragment key={index}>{item}</Fragment>]
      : [
          <Fragment key={`separator-${index}`}>{separator}</Fragment>,
          <Fragment key={index}>{item}</Fragment>,
        ],
  )
}
