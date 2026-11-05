import * as React from 'react'

import { cn } from '@/lib/utils'

/** Shared look of text fields and native selects (specs/charte-graphique.md, Composants de base). */
export const fieldClassName =
  'h-10 rounded-lg border border-line-strong bg-paper-50 px-3 text-sm outline-none transition focus:border-brass-700 focus:ring-2 focus:ring-brass-700/20'

function Input({ className, ...props }: React.ComponentProps<'input'>) {
  return <input className={cn(fieldClassName, 'w-full', className)} {...props} />
}

export { Input }
