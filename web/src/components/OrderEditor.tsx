import {
  forwardRef,
  useCallback,
  useRef,
  type ChangeEvent,
  type ReactNode,
  type UIEvent,
} from 'react'

import { cn } from '@/lib/utils'

export type OrderTokenKind = 'comment' | 'trigram' | 'verb' | 'number' | 'punctuation' | 'text'

export interface OrderToken {
  kind: OrderTokenKind
  text: string
}

const TOKEN_PATTERN = /#.*|\p{L}+|\d+|[()-]|[^\p{L}\d#()-]+/gu

function wordKind(word: string): OrderTokenKind {
  if (word.length === 1) return 'verb'
  if (word.length === 3) return 'trigram'
  return 'text'
}

/**
 * Splits one line of orders into lexical tokens: a comment, a word of three
 * letters (a code), a one-letter verb, a number or a bracket and hyphen. It
 * only looks at the shape of the words, never at what an order means, and the
 * tokens joined give the line back.
 */
export function tokenizeOrderLine(line: string): OrderToken[] {
  const tokens: OrderToken[] = []
  for (const [text] of line.matchAll(TOKEN_PATTERN)) {
    let kind: OrderTokenKind = 'text'
    if (text.startsWith('#')) kind = 'comment'
    else if (/^\p{L}/u.test(text)) kind = wordKind(text)
    else if (/^\d/.test(text)) kind = 'number'
    else if (/^[()-]$/.test(text)) kind = 'punctuation'
    tokens.push({ kind, text })
  }
  return tokens
}

export type OrderEditorTone = 'default' | 'winter'

// Shared by the textarea and its mirror: any difference shifts the colors.
const TEXT_LAYOUT = 'block w-full border p-3 font-mono text-xs whitespace-pre-wrap break-words'

const TONES: Record<
  OrderEditorTone,
  { frame: string; field: string; base: string; caret: string; trigram: string }
> = {
  default: {
    frame: 'bg-paper-50',
    field:
      'border-line-strong focus:border-brass-700 focus:ring-brass-700/20 placeholder:text-ink-500',
    base: 'text-ink-900',
    caret: 'caret-ink-900',
    trigram: 'bg-paper-100',
  },
  winter: {
    frame: 'bg-winter-100',
    field:
      'border-winter-700/30 focus:border-winter-700 focus:ring-winter-700/20 placeholder:text-winter-700/60',
    base: 'text-winter-700',
    caret: 'caret-winter-700',
    trigram: 'bg-winter-700/10',
  },
}

const TOKEN_CLASSES: Record<OrderTokenKind, string> = {
  comment: 'text-ink-500',
  trigram: '', // tone dependent
  verb: 'font-semibold text-brass-700',
  number: 'text-success-700',
  punctuation: 'text-ink-500',
  text: '',
}

function renderLine(line: string, trigramClass: string): ReactNode {
  return tokenizeOrderLine(line).map((token, index) => (
    <span
      key={index}
      className={token.kind === 'trigram' ? cn('rounded-sm', trigramClass) : TOKEN_CLASSES[token.kind]}
    >
      {token.text}
    </span>
  ))
}

interface OrderEditorProps {
  value: string
  onChange: (value: string) => void
  /** 1-based lines to underline as faulty. */
  errorLines?: readonly number[]
  /** Id of the list that explains the errors of the zone. */
  errorsId?: string
  /** Marks the zone invalid even when no line can be underlined. */
  invalid?: boolean
  tone?: OrderEditorTone
  disabled?: boolean
  placeholder?: string
  'aria-label': string
  /** Classes of the textarea sizing (height), for instance `min-h-24`. */
  className?: string
}

/**
 * Textarea of orders with lexical coloring: the real textarea, whose text is
 * transparent, lies over an aria-hidden mirror that draws the same text in
 * colors. Both share their font, padding and wrapping so that the characters
 * line up, and the mirror follows the textarea scroll.
 */
export const OrderEditor = forwardRef<HTMLTextAreaElement, OrderEditorProps>(
  function OrderEditor(
    {
      value,
      onChange,
      errorLines = [],
      errorsId,
      invalid,
      tone = 'default',
      disabled,
      placeholder,
      'aria-label': ariaLabel,
      className,
    },
    forwardedRef,
  ) {
    const mirrorRef = useRef<HTMLPreElement>(null)
    const styles = TONES[tone]
    const faulty = new Set(errorLines)
    const isInvalid = invalid ?? faulty.size > 0

    const setRefs = useCallback(
      (node: HTMLTextAreaElement | null) => {
        if (typeof forwardedRef === 'function') forwardedRef(node)
        else if (forwardedRef) forwardedRef.current = node
      },
      [forwardedRef],
    )

    const syncScroll = (event: UIEvent<HTMLTextAreaElement>) => {
      const mirror = mirrorRef.current
      if (!mirror) return
      mirror.scrollTop = event.currentTarget.scrollTop
      mirror.scrollLeft = event.currentTarget.scrollLeft
    }

    return (
      <div
        className={cn(
          'relative rounded-lg',
          styles.frame,
          disabled && 'cursor-not-allowed opacity-50',
        )}
      >
        <pre
          ref={mirrorRef}
          aria-hidden="true"
          data-order-mirror
          className={cn(
            TEXT_LAYOUT,
            'pointer-events-none absolute inset-0 m-0 overflow-y-auto border-transparent [scrollbar-color:transparent_transparent]',
            styles.base,
          )}
        >
          {value.split('\n').map((line, index, lines) => (
            <span key={index}>
              {faulty.has(index + 1) ? (
                <span
                  data-order-error-line
                  className="underline decoration-vermilion-700 decoration-wavy"
                >
                  {renderLine(line, styles.trigram)}
                </span>
              ) : (
                renderLine(line, styles.trigram)
              )}
              {/* The space gives the last empty line its height. */}
              {index < lines.length - 1 ? '\n' : ' '}
            </span>
          ))}
        </pre>
        <textarea
          ref={setRefs}
          value={value}
          onChange={(event: ChangeEvent<HTMLTextAreaElement>) => onChange(event.target.value)}
          onScroll={syncScroll}
          disabled={disabled}
          placeholder={placeholder}
          aria-label={ariaLabel}
          aria-invalid={isInvalid || undefined}
          aria-describedby={isInvalid ? errorsId : undefined}
          spellCheck={false}
          className={cn(
            TEXT_LAYOUT,
            'relative min-h-16 resize-y overflow-y-auto rounded-lg bg-transparent text-transparent outline-none transition selection:bg-brass-300/40 focus:ring-2 disabled:cursor-not-allowed',
            styles.field,
            styles.caret,
            className,
          )}
        />
      </div>
    )
  },
)
