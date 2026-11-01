import {
  forwardRef,
  useId,
  useImperativeHandle,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
  type ChangeEvent,
  type KeyboardEvent,
  type ReactNode,
  type UIEvent,
} from 'react'

import { useLanguage } from '@/i18n/LanguageContext'
import { wordAtCursor, type CodeSuggestion } from '@/lib/order-suggestions'
import { cn } from '@/lib/utils'

export type OrderTokenKind = 'comment' | 'trigram' | 'verb' | 'number' | 'punctuation' | 'text'

export interface OrderToken {
  kind: OrderTokenKind
  text: string
}

// One group per kind, in the order of OrderTokenKind; text matches the rest.
const TOKEN_PATTERN = /(#.*)|(\p{L}{3}(?!\p{L}))|(\p{L}(?!\p{L}))|(\d+)|([()-])|(\p{L}+|[^\p{L}\d#()-]+)/gu
const GROUP_KINDS: OrderTokenKind[] = ['comment', 'trigram', 'verb', 'number', 'punctuation', 'text']

/**
 * Splits one line of orders into lexical tokens: a comment, a word of three
 * letters (a code), a one-letter verb, a number or a bracket and hyphen. It
 * only looks at the shape of the words, never at what an order means, and the
 * tokens joined give the line back.
 */
export function tokenizeOrderLine(line: string): OrderToken[] {
  return [...line.matchAll(TOKEN_PATTERN)].map((match) => ({
    kind: GROUP_KINDS[match.slice(1).findIndex((group) => group !== undefined)],
    text: match[0],
  }))
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
  /** Marks the zone invalid, even when no line can be underlined. */
  invalid?: boolean
  tone?: OrderEditorTone
  disabled?: boolean
  placeholder?: string
  'aria-label': string
  /** Classes of the textarea sizing (height), for instance `min-h-24`. */
  className?: string
  /** Codes that complete the typed word, given the line before it. */
  suggest?: (word: string, lineBefore: string) => CodeSuggestion[]
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
      suggest,
    },
    forwardedRef,
  ) {
    const { t } = useLanguage()
    const mirrorRef = useRef<HTMLPreElement>(null)
    const textareaRef = useRef<HTMLTextAreaElement>(null)
    useImperativeHandle(forwardedRef, () => textareaRef.current as HTMLTextAreaElement)
    const listId = useId()
    // Cursor of a focused textarea; null while the focus is elsewhere.
    const [cursor, setCursor] = useState<number | null>(null)
    const [active, setActive] = useState(0)
    // Word whose suggestions Escape closed, until the player types on.
    const [dismissed, setDismissed] = useState<string | null>(null)
    const pendingCursor = useRef<number | null>(null)
    const styles = TONES[tone]
    const faultyKey = errorLines.join(',')
    // Re-tokenized only when the text, its faulty lines or the tone change.
    const mirrorLines = useMemo(() => {
      const faulty = new Set(errorLines)
      return value.split('\n').map((line, index, lines) => (
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
      ))
      // `faultyKey` stands for the content of `errorLines`.
      // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [value, faultyKey, styles.trigram])

    const word = suggest && cursor !== null ? wordAtCursor(value, cursor) : null
    const wordKey = word ? `${word.start}:${word.text}` : null
    const suggestions = useMemo(() => {
      if (!suggest || !word) return []
      const lineStart = value.lastIndexOf('\n', word.start - 1) + 1
      return suggest(word.text, value.slice(lineStart, word.start))
      // `wordKey` stands for `word`.
      // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [suggest, value, wordKey])
    const open = suggestions.length > 0 && wordKey !== dismissed
    const highlighted = Math.min(active, suggestions.length - 1)

    useLayoutEffect(() => {
      const textarea = textareaRef.current
      if (pendingCursor.current === null || !textarea) return
      textarea.setSelectionRange(pendingCursor.current, pendingCursor.current)
      setCursor(pendingCursor.current)
      pendingCursor.current = null
    }, [value])

    const accept = (suggestion: CodeSuggestion) => {
      if (!word) return
      pendingCursor.current = word.start + suggestion.code.length
      onChange(value.slice(0, word.start) + suggestion.code + value.slice(word.end))
      setActive(0)
    }

    const handleKeyDown = (event: KeyboardEvent<HTMLTextAreaElement>) => {
      if (!open) return
      if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
        event.preventDefault()
        const step = event.key === 'ArrowDown' ? 1 : -1
        setActive((highlighted + step + suggestions.length) % suggestions.length)
      } else if (event.key === 'Enter' || event.key === 'Tab') {
        event.preventDefault()
        accept(suggestions[highlighted])
      } else if (event.key === 'Escape') {
        event.preventDefault()
        setDismissed(wordKey)
      }
    }

    const trackCursor = (event: { currentTarget: HTMLTextAreaElement }) => {
      const { selectionStart, selectionEnd } = event.currentTarget
      setCursor(selectionStart === selectionEnd ? selectionStart : null)
    }

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
          {mirrorLines}
        </pre>
        <textarea
          ref={textareaRef}
          value={value}
          onChange={(event: ChangeEvent<HTMLTextAreaElement>) => {
            onChange(event.target.value)
            // Text inserted without keys (paste, phone keyboards) fires no select.
            trackCursor(event)
            setActive(0)
          }}
          onSelect={trackCursor}
          onFocus={trackCursor}
          onBlur={() => setCursor(null)}
          onKeyDown={handleKeyDown}
          onScroll={syncScroll}
          role={suggest ? 'combobox' : undefined}
          aria-autocomplete={suggest ? 'list' : undefined}
          aria-expanded={suggest ? open : undefined}
          aria-controls={open ? listId : undefined}
          aria-activedescendant={open ? `${listId}-${highlighted}` : undefined}
          disabled={disabled}
          placeholder={placeholder}
          aria-label={ariaLabel}
          aria-invalid={invalid || undefined}
          aria-describedby={invalid ? errorsId : undefined}
          spellCheck={false}
          className={cn(
            TEXT_LAYOUT,
            'relative min-h-16 resize-y overflow-y-auto rounded-lg bg-transparent text-transparent outline-none transition selection:bg-brass-300/40 focus:ring-2 disabled:cursor-not-allowed',
            styles.field,
            styles.caret,
            className,
          )}
        />
        {open && (
          <ul
            id={listId}
            role="listbox"
            aria-label={t('orders.suggestionsAria')}
            className="absolute inset-x-0 top-full z-30 mt-1 max-h-48 overflow-y-auto rounded-md border border-line bg-paper-50 py-1 shadow-panel"
          >
            {suggestions.map((suggestion, index) => (
              <li
                key={suggestion.code}
                id={`${listId}-${index}`}
                role="option"
                aria-label={`${suggestion.name} (${suggestion.code})`}
                aria-selected={index === highlighted}
                // Keep the focus in the textarea: the click only picks.
                onMouseDown={(event) => {
                  event.preventDefault()
                  accept(suggestion)
                }}
                className={cn(
                  'flex cursor-pointer items-baseline gap-2 px-3 py-1.5 text-sm text-ink-900 sm:py-1',
                  index === highlighted && 'bg-paper-100',
                )}
              >
                <span className="font-mono text-xs font-semibold">{suggestion.code}</span>
                <span className="truncate">{suggestion.name}</span>
                {suggestion.kind === 'noble' && (
                  <span className="ml-auto text-xs text-ink-500">{t('orders.suggestionNoble')}</span>
                )}
              </li>
            ))}
          </ul>
        )}
      </div>
    )
  },
)
