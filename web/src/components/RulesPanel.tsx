import { useEffect, useRef, useState, type ReactNode } from 'react'
import ReactMarkdown from 'react-markdown'
import rehypeRaw from 'rehype-raw'
import remarkGfm from 'remark-gfm'

import { useLanguage } from '@/i18n/LanguageContext'
import { apiTextRequest, type TokenProvider } from '@/lib/api'

export type RulesSection = 'action-orders' | 'winter-orders'

export const RULE_SECTION_IDS: Record<RulesSection, string> = {
  'action-orders': 'rules-action-orders',
  'winter-orders': 'rules-winter-orders',
}

interface RulesPanelProps {
  variant?: 'panel' | 'page'
  targetSection?: RulesSection
  navigationKey?: number
  gameId?: string
  tokenProvider?: TokenProvider
}

function markdownText(children: ReactNode): string {
  if (typeof children === 'string' || typeof children === 'number') {
    return String(children)
  }
  if (Array.isArray(children)) {
    return children.map(markdownText).join('')
  }
  return ''
}

function sectionAnchor(children: ReactNode): string | undefined {
  const heading = markdownText(children).toLowerCase()
  if (
    heading.includes('aide-mémoire des ordres') ||
    heading.includes('order cheat sheet')
  ) {
    return RULE_SECTION_IDS['action-orders']
  }
  if (
    heading.includes("ordres d'hiver") ||
    heading.includes('ordres d’hiver') ||
    heading.includes('winter orders')
  ) {
    return RULE_SECTION_IDS['winter-orders']
  }
  return undefined
}

export function RulesPanel({
  variant = 'panel',
  targetSection,
  navigationKey = 0,
  gameId,
  tokenProvider,
}: RulesPanelProps) {
  const { language, t } = useLanguage()
  const [rulesMarkdown, setRulesMarkdown] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const rulesContainerRef = useRef<HTMLDivElement>(null)
  const getIdToken = tokenProvider?.getIdToken

  useEffect(() => {
    const controller = new AbortController()

    const loadRules = async () => {
      try {
        const path = gameId
          ? `/api/games/${encodeURIComponent(gameId)}/rules?lang=${language}`
          : `/api/rules?lang=${language}`
        const content = getIdToken
          ? await apiTextRequest({ getIdToken }, path)
          : await fetch(path, { signal: controller.signal }).then(async (response) => {
              if (!response.ok) {
                throw new Error(t('rules.loadFailed', { status: response.status }))
              }
              return response.text()
            })
        if (!content.trim()) {
          throw new Error(t('rules.empty'))
        }
        if (!controller.signal.aborted) {
          setRulesMarkdown(content)
          setError(null)
        }
      } catch (loadError) {
        if (!controller.signal.aborted) {
          setError(
            loadError instanceof Error
              ? loadError.message
              : t('rules.loadFailed', { status: 500 }),
          )
        }
      }
    }

    void loadRules()
    return () => controller.abort()
  }, [gameId, getIdToken, language, t])

  useEffect(() => {
    if (!targetSection || !rulesMarkdown) return

    const frame = window.requestAnimationFrame(() => {
      const anchor = rulesContainerRef.current?.querySelector<HTMLElement>(
        `#${RULE_SECTION_IDS[targetSection]}`,
      )
      anchor?.scrollIntoView?.({ behavior: 'smooth', block: 'start' })
    })

    return () => window.cancelAnimationFrame(frame)
  }, [navigationKey, rulesMarkdown, targetSection])

  return (
    <section aria-label={t('app.rules')} className="min-w-0">
      <div className="mb-4 flex items-start justify-between gap-3">
        <p className="text-xs leading-relaxed text-ink-500">{t('rules.reference')}</p>
        <span className="shrink-0 rounded-md border border-line bg-paper-100 px-2 py-1 text-2xs font-bold uppercase tracking-[0.14em] text-ink-500">
          {language.toUpperCase()}
        </span>
      </div>

      {error ? (
        <p
          role="alert"
          className="rounded-lg border border-vermilion-700/30 bg-vermilion-100 px-3 py-3 text-sm text-vermilion-700"
        >
          {error}
        </p>
      ) : rulesMarkdown ? (
        <div
          ref={rulesContainerRef}
          className={
            variant === 'page'
              ? 'mx-auto max-w-3xl text-base leading-relaxed text-ink-700'
              : 'max-h-[70vh] overflow-y-auto pr-1 text-sm leading-relaxed text-ink-700'
          }
        >
          <ReactMarkdown
            remarkPlugins={[remarkGfm]}
            rehypePlugins={[rehypeRaw]}
            components={{
              h1: ({ children }) => (
                <h2 className="mb-4 font-display text-2xl font-semibold leading-tight text-ink-900">
                  {children}
                </h2>
              ),
              h2: ({ children }) => (
                <h3
                  id={sectionAnchor(children)}
                  className="mt-7 border-t border-line/60 pt-4 font-display text-lg font-semibold text-ink-900 first:mt-0 first:border-t-0 first:pt-0"
                >
                  {children}
                </h3>
              ),
              h3: ({ children }) => (
                <h4 className="mt-5 font-semibold text-ink-500">{children}</h4>
              ),
              p: ({ children }) => <p className="mt-2 first:mt-0">{children}</p>,
              ul: ({ children }) => (
                <ul className="mt-2 list-disc space-y-1 pl-5">{children}</ul>
              ),
              ol: ({ children }) => (
                <ol className="mt-2 list-decimal space-y-1 pl-5">{children}</ol>
              ),
              li: ({ children }) => <li className="pl-1">{children}</li>,
              blockquote: ({ children }) => (
                <blockquote className="mt-3 border-l-4 border-brass-700/50 bg-paper-100 px-3 py-2 italic text-ink-500">
                  {children}
                </blockquote>
              ),
              code: ({ children }) => (
                <code className="rounded bg-paper-100 px-1.5 py-0.5 font-mono text-[0.85em] text-vermilion-700">
                  {children}
                </code>
              ),
              pre: ({ children }) => (
                <pre className="mt-3 overflow-x-auto rounded-lg border border-line/60 bg-ink-900 p-3 font-mono text-xs leading-relaxed text-paper-50">
                  {children}
                </pre>
              ),
              table: ({ children }) => (
                <div className="mt-3 overflow-x-auto rounded-lg border border-line/60">
                  <table className="min-w-full border-collapse text-left text-xs">
                    {children}
                  </table>
                </div>
              ),
              th: ({ children }) => (
                <th className="border-b border-line/70 bg-paper-100 px-2.5 py-2 font-semibold text-ink-900">
                  {children}
                </th>
              ),
              td: ({ children }) => (
                <td className="border-b border-line/40 px-2.5 py-2 align-top last:border-b-0">
                  {children}
                </td>
              ),
              hr: () => <hr className="my-5 border-line/60" />,
              a: ({ children, href }) => (
                <a
                  href={href}
                  target="_blank"
                  rel="noreferrer"
                  className="font-medium text-vermilion-700 underline decoration-brass-700/40 underline-offset-2 hover:text-brass-700"
                >
                  {children}
                </a>
              ),
              strong: ({ children }) => (
                <strong className="font-semibold text-ink-900">{children}</strong>
              ),
            }}
          >
            {rulesMarkdown}
          </ReactMarkdown>
        </div>
      ) : (
        <p className="rounded-lg border border-dashed border-line bg-paper-100 px-3 py-4 text-sm italic text-ink-500">
          {t('rules.loading')}
        </p>
      )}
    </section>
  )
}
