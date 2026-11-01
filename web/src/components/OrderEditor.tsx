import {
  forwardRef,
  useEffect,
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

import { IconListSearch } from '@tabler/icons-react'

import { useLanguage } from '@/i18n/LanguageContext'
import { wordAtCursor, type CodeSuggester, type CodeSuggestion } from '@/lib/order-suggestions'
import { cn } from '@/lib/utils'

export type OrderTokenKind = 'comment' | 'trigram' | 'verb' | 'number' | 'punctuation' | 'text'

export interface OrderToken {
  kind: OrderTokenKind
  text: string
}

// One group per kind, in the order of OrderTokenKind; text matches the rest.
const TOKEN_PATTERN = /(#.*)|(\p{L}{2,3}(?!\p{L}))|(\p{L}(?!\p{L}))|(\d+)|([()-])|(\p{L}+|[^\p{L}\d#()-]+)/gu
const GROUP_KINDS: OrderTokenKind[] = ['comment', 'trigram', 'verb', 'number', 'punctuation', 'text']

/**
 * Splits one line of orders into lexical tokens: a comment, a word of two or
 * three letters (a code: a card, a territory, a noble), a one-letter verb, a number or a bracket and hyphen. It
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
// Right padding that keeps the text clear of the suggestion button.
const SUGGEST_ROOM = 'pr-12 sm:pr-10'

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
  /** Codes that complete the typed word, given the text of the zone before it. */
  suggest?: CodeSuggester
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
    // Start of the word Ctrl+Space asked suggestions for, even without letters.
    const [requested, setRequested] = useState<number | null>(null)
    // The player went through the list with the arrows.
    const [browsed, setBrowsed] = useState(false)
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

    const typed = suggest && cursor !== null ? wordAtCursor(value, cursor, 0) : null
    const word = typed && (typed.text.length >= 2 || typed.start === requested) ? typed : null
    const wordKey = word ? `${word.start}:${word.text}` : null
    const suggestions = useMemo(() => {
      if (!suggest || !word) return []
      return suggest(word.text, value.slice(0, word.start))
      // `wordKey` stands for `word`.
      // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [suggest, value, wordKey])
    const open = suggestions.length > 0 && wordKey !== dismissed
    const highlighted = Math.min(active, suggestions.length - 1)

    // Keeps the highlighted option visible in the list; the panel itself only
    // scrolls to a list the player asked for, not while they type.
    useEffect(() => {
      if (!open) return
      const list = document.getElementById(listId)
      const option = document.getElementById(`${listId}-${highlighted}`)
      if (list && option) {
        if (option.offsetTop < list.scrollTop) list.scrollTop = option.offsetTop
        const bottom = option.offsetTop + option.offsetHeight
        if (bottom > list.scrollTop + list.clientHeight) list.scrollTop = bottom - list.clientHeight
      }
      if (requested !== null) list?.scrollIntoView?.({ block: 'nearest' })
    }, [open, highlighted, listId, requested])

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
      setRequested(null)
      setBrowsed(false)
    }

    /** Opens the suggestions at `position`, even before any letter. */
    const request = (position: number) => {
      setRequested(wordAtCursor(value, position, 0)?.start ?? null)
      setDismissed(null)
      setActive(0)
      setBrowsed(false)
    }

    const handleKeyDown = (event: KeyboardEvent<HTMLTextAreaElement>) => {
      // Ctrl+Space, or Alt+Down where the system takes Ctrl+Space (macOS).
      const asks =
        (event.key === ' ' && event.ctrlKey) || (event.key === 'ArrowDown' && event.altKey)
      if (suggest && asks) {
        event.preventDefault()
        request(event.currentTarget.selectionStart)
        return
      }
      if (!open) return
      if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
        event.preventDefault()
        const step = event.key === 'ArrowDown' ? 1 : -1
        setActive((highlighted + step + suggestions.length) % suggestions.length)
        setBrowsed(true)
      } else if (event.key === 'Enter' || event.key === 'Tab') {
        // A list that opened by itself never takes the key: Enter still
        // breaks the line and Tab leaves the field.
        if (requested === null && !browsed) return
        event.preventDefault()
        accept(suggestions[highlighted])
      } else if (event.key === 'Escape') {
        event.preventDefault()
        setDismissed(wordKey)
        setRequested(null)
        setBrowsed(false)
      }
    }

    const trackCursor = (event: { currentTarget: HTMLTextAreaElement }) => {
      const { selectionStart, selectionEnd, value: text } = event.currentTarget
      setCursor(selectionStart === selectionEnd ? selectionStart : null)
      // A request holds for its word only.
      const start = wordAtCursor(text, selectionStart, 0)?.start
      setRequested((current) => (current === start ? current : null))
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
            suggest && SUGGEST_ROOM,
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
            setBrowsed(false)
          }}
          onSelect={trackCursor}
          onFocus={trackCursor}
          onBlur={() => {
            setCursor(null)
            setRequested(null)
            setBrowsed(false)
          }}
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
            suggest && SUGGEST_ROOM,
            'relative min-h-16 resize-y overflow-y-auto rounded-lg bg-transparent text-transparent outline-none transition selection:bg-brass-300/40 focus:ring-2 disabled:cursor-not-allowed',
            styles.field,
            styles.caret,
            className,
          )}
        />
        {suggest && !disabled && (
          <button
            type="button"
            aria-label={t('orders.suggestionsOpen')}
            title={t('orders.suggestionsOpen')}
            // Keep the focus, and the phone keyboard, in the textarea.
            onPointerDown={(event) => event.preventDefault()}
            onClick={() => {
              const textarea = textareaRef.current
              if (!textarea) return
              textarea.focus()
              request(textarea.selectionStart)
            }}
            className="absolute right-1 top-1 z-10 flex size-11 items-center justify-center rounded-md text-ink-500 hover:bg-paper-100 hover:text-ink-900 sm:size-8"
          >
            <IconListSearch aria-hidden="true" className="size-4" />
          </button>
        )}
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
