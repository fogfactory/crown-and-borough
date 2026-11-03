import * as React from 'react'

import { cn } from '@/lib/utils'

const TONES = {
  error: 'border-vermilion-700/30 bg-vermilion-100 text-vermilion-700',
  success: 'border-success-700/30 bg-paper-100 text-success-700',
  info: 'border-brass-700/40 bg-brass-300/30 text-brass-700',
} as const

function Alert({
  tone = 'error',
  className,
  ...props
}: React.ComponentProps<'div'> & { tone?: keyof typeof TONES }) {
  return (
    <div
      className={cn('rounded-lg border px-3 py-2 text-sm', TONES[tone], className)}
      {...props}
    />
  )
}

export { Alert }
