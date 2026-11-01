import {
  forwardRef,
  useEffect,
  useId,
  useImperativeHandle,
  useMemo,
  useRef,
  useState,
  type ChangeEvent,
  type KeyboardEvent,
  type ReactNode,
  type UIEvent,
} from 'react'

import { IconListSearch } from '@tabler/icons-react'

import { useEditorMapPicker } from '@/components/OrderDialog'
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

/** What the player did with the suggestions of the word starting at `start`. */
interface ListState {
  start: number
  /** Asked with Ctrl+Space, Alt+Down or the button: open even before two letters. */
  requested: boolean
  /** Chosen option, -1 for none. */
  active: number
  /** Text of the word Escape closed the list on, until it changes. */
  dismissed?: string
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
  /** Told each three-letter word typed or chosen, to follow it on the map. */
  onCodeTyped?: (code: string) => void
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
      onCodeTyped,
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
    // What the player did with the list of one word; it lapses when they leave it.
    const [list, setList] = useState<ListState | null>(null)
    const frameRef = useRef<HTMLDivElement>(null)
    // Where a territory clicked on the map goes: the last selection, kept
    // through the blur that the click causes.
    const selection = useRef<{ start: number; end: number } | null>(null)
    // The last code told to `onCodeTyped`, so a code is followed once.
    const lastCode = useRef<string | null>(null)
    const setEditorPicker = useEditorMapPicker()
    const [mapTarget, setMapTarget] = useState(false)
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

    const typed = suggest && cursor !== null ? wordAtCursor(value, cursor) : null
    const current = typed && list?.start === typed.start ? list : null
    // A single letter is a verb: the list waits for two, unless asked.
    const word = typed && (typed.text.length >= 2 || current?.requested) ? typed : null
    const wordText = word?.text ?? null
    const textBefore = word ? value.slice(0, word.start) : ''
    const suggestions = useMemo(
      () => (suggest && wordText !== null ? suggest(wordText, textBefore) : []),
      [suggest, wordText, textBefore],
    )
    const open = suggestions.length > 0 && current?.dismissed !== wordText
    // No option is chosen until the player asks for the list or uses the arrows.
    const highlighted = current ? Math.min(current.active, suggestions.length - 1) : -1
    const requested = Boolean(current?.requested)

    // Keeps the highlighted option visible in the list; the panel itself only
    // scrolls to a list the player asked for, not while they type.
    useEffect(() => {
      if (!open) return
      const listBox = document.getElementById(listId)
      const option = document.getElementById(`${listId}-${highlighted}`)
      if (listBox && option) {
        if (option.offsetTop < listBox.scrollTop) listBox.scrollTop = option.offsetTop
        const bottom = option.offsetTop + option.offsetHeight
        if (bottom > listBox.scrollTop + listBox.clientHeight) {
          listBox.scrollTop = bottom - listBox.clientHeight
        }
      }
      if (requested) listBox?.scrollIntoView?.({ block: 'nearest' })
    }, [open, highlighted, listId, requested])

    const accept = (suggestion: CodeSuggestion) => {
      const textarea = textareaRef.current
      if (!word || !textarea) return
      // The textarea puts the cursor after the code itself.
      textarea.setRangeText(suggestion.code, word.start, word.end, 'end')
      onChange(textarea.value)
      setCursor(textarea.selectionStart)
      setList(null)
      lastCode.current = suggestion.code
      onCodeTyped?.(suggestion.code)
    }

    /**
     * Inserts the territory clicked on the map at the kept selection: over
     * the code being typed, spaced from its neighbours.
     */
    const insertFromMap = (code: string) => {
      const textarea = textareaRef.current
      if (!textarea) return
      const text = textarea.value
      const { start, end } = selection.current ?? { start: text.length, end: text.length }
      const typedWord = start === end ? wordAtCursor(text, start) : null
      // A code being typed gives way; a one-letter verb stays.
      const from = typedWord && typedWord.text.length >= 2 ? typedWord.start : start
      const before = text.slice(0, from)
      const after = text.slice(end)
      const inserted =
        (before && !/[\s(]$/.test(before) ? ' ' : '') +
        code +
        (after && !/^[\s)]/.test(after) ? ' ' : '')
      textarea.setRangeText(inserted, from, end, 'end')
      onChange(textarea.value)
      selection.current = { start: textarea.selectionStart, end: textarea.selectionEnd }
      lastCode.current = code
      textarea.focus({ preventScroll: true })
    }
    const insertRef = useRef(insertFromMap)
    insertRef.current = insertFromMap

    /** Hands the map clicks to this editor, or takes them back. */
    const targetMap = (on: boolean) => {
      if (!setEditorPicker || disabled) return
      setEditorPicker(on ? (code) => insertRef.current(code) : null)
      setMapTarget(on)
    }
    const release = useRef<(() => void) | null>(null)
    release.current = mapTarget ? () => setEditorPicker?.(null) : null
    // An editor that goes away takes its map clicks with it.
    useEffect(() => () => release.current?.(), [])

    /** Opens the suggestions at `position`, even before any letter. */
    const request = (position: number) => {
      const at = wordAtCursor(value, position)
      setList(at ? { start: at.start, requested: true, active: 0 } : null)
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
      if (!open || !word) return
      if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
        event.preventDefault()
        const step = event.key === 'ArrowDown' ? 1 : -1
        const next =
          highlighted < 0
            ? step > 0
              ? 0
              : suggestions.length - 1
            : (highlighted + step + suggestions.length) % suggestions.length
        setList({ start: word.start, requested, active: next })
      } else if (event.key === 'Enter' || event.key === 'Tab') {
        // Without a chosen option, Enter breaks the line and Tab leaves the field.
        if (highlighted < 0) return
        event.preventDefault()
        accept(suggestions[highlighted])
      } else if (event.key === 'Escape') {
        event.preventDefault()
        setList({ start: word.start, requested: false, active: -1, dismissed: word.text })
      }
    }

    const handleEscape = (event: KeyboardEvent<HTMLTextAreaElement>) => {
      // Escape with no list open gives the map its clicks back.
      if (event.key === 'Escape' && !open && mapTarget) targetMap(false)
    }

    const trackCursor = (event: { currentTarget: HTMLTextAreaElement }) => {
      const { selectionStart, selectionEnd, value: text } = event.currentTarget
      setCursor(selectionStart === selectionEnd ? selectionStart : null)
      selection.current = { start: selectionStart, end: selectionEnd }
      // Leaving the word forgets what was done with its list.
      const start = wordAtCursor(text, selectionStart)?.start
      setList((state) => (state?.start === start ? state : null))
    }

    // Shared by the textarea and its mirror, like TEXT_LAYOUT.
    const layout = cn(TEXT_LAYOUT, suggest && SUGGEST_ROOM)

    const syncScroll = (event: UIEvent<HTMLTextAreaElement>) => {
      const mirror = mirrorRef.current
      if (!mirror) return
      mirror.scrollTop = event.currentTarget.scrollTop
      mirror.scrollLeft = event.currentTarget.scrollLeft
    }

    return (
      <>
        <div
          ref={frameRef}
          data-map-target={mapTarget || undefined}
          className={cn(
            'relative rounded-lg',
            styles.frame,
            disabled && 'cursor-not-allowed opacity-50',
            // The field the map clicks fill (charte § Poste de commandement).
            mapTarget && 'ring-2 ring-brass-700',
          )}
        >
          <pre
            ref={mirrorRef}
            aria-hidden="true"
            data-order-mirror
            className={cn(
              layout,
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
              // A whole three-letter word just typed: the map follows it once.
              const typedNow = wordAtCursor(event.target.value, event.target.selectionStart)
              const code = typedNow?.text.length === 3 ? typedNow.text.toUpperCase() : null
              if (code && code !== lastCode.current) onCodeTyped?.(code)
              lastCode.current = code
              // A new letter: the choice made among the old suggestions lapses.
              setList((state) => state && { ...state, active: state.requested ? 0 : -1 })
            }}
            onSelect={trackCursor}
            onFocus={(event) => {
              trackCursor(event)
              targetMap(true)
            }}
            onBlur={(event) => {
              setCursor(null)
              setList(null)
              // A click on the map keeps the clicks here; any other focus takes them back.
              const next = event.relatedTarget
              if (next && !next.closest('[data-map-size]') && !frameRef.current?.contains(next)) {
                targetMap(false)
              }
            }}
            onKeyDown={(event) => {
              handleKeyDown(event)
              handleEscape(event)
            }}
            onScroll={syncScroll}
            role={suggest ? 'combobox' : undefined}
            aria-autocomplete={suggest ? 'list' : undefined}
            aria-expanded={suggest ? open : undefined}
            aria-controls={open ? listId : undefined}
            aria-activedescendant={open && highlighted >= 0 ? `${listId}-${highlighted}` : undefined}
            disabled={disabled}
            placeholder={placeholder}
            aria-label={ariaLabel}
            aria-invalid={invalid || undefined}
            aria-describedby={invalid ? errorsId : undefined}
            spellCheck={false}
            className={cn(
              layout,
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
        {mapTarget && <p className="text-2xs text-ink-500">{t('orders.mapPickHint')}</p>}
      </>
    )
  },
)
