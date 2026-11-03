import * as React from 'react'

import { cn } from '@/lib/utils'

/** Tone classes shared by boxes and pills (specs/charte-graphique.md, Composants de base). */
export const alertToneClassName = {
  error: 'border-vermilion-700/30 bg-vermilion-100 text-vermilion-700',
  success: 'border-success-700/30 bg-paper-100 text-success-700',
  info: 'border-brass-700/40 bg-brass-300/30 text-brass-700',
} as const

export type AlertTone = keyof typeof alertToneClassName

const SIZES = {
  default: 'px-3 py-2',
  banner: 'px-4 py-3',
} as const

/**
 * A toned box. It carries no ARIA role of its own: add `role="alert"` or
 * `role="status"` only for a live message; a static banner (victory,
 * observer, invitation code) stays a plain box.
 */
function Alert({
  tone = 'error',
  size = 'default',
  className,
  ...props
}: React.ComponentProps<'div'> & { tone?: AlertTone; size?: keyof typeof SIZES }) {
  return (
    <div
      className={cn(
        'rounded-lg border text-sm',
        SIZES[size],
        alertToneClassName[tone],
        className,
      )}
      {...props}
    />
  )
}

export { Alert }
