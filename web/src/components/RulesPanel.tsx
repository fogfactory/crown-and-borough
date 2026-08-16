import { useEffect, useRef, useState, type ReactNode } from 'react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'

import { useLanguage } from '@/i18n/LanguageContext'
import { apiTextRequest, type TokenProvider } from '@/lib/api'

export type RulesSection = 'action-orders' | 'winter-orders'

export const RULE_SECTION_IDS: Record<RulesSection, string> = {
  'action-orders': 'rules-action-orders',
  'winter-orders': 'rules-winter-orders',
}

interface RulesPanelProps {
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
        <p className="text-xs leading-relaxed text-[#806f57]">{t('rules.reference')}</p>
        <span className="shrink-0 rounded-md border border-[#b7a786] bg-[#f3ead9] px-2 py-1 text-[10px] font-bold uppercase tracking-[0.14em] text-[#806f57]">
          {language.toUpperCase()}
        </span>
      </div>

      {error ? (
        <p
          role="alert"
          className="rounded-lg border border-[#a84632]/30 bg-[#f8e5dd] px-3 py-3 text-sm text-[#8d321e]"
        >
          {error}
        </p>
      ) : rulesMarkdown ? (
        <div
          ref={rulesContainerRef}
          className="max-h-[70vh] overflow-y-auto pr-1 text-sm leading-relaxed text-[#594b3c]"
        >
          <ReactMarkdown
            remarkPlugins={[remarkGfm]}
            components={{
              h1: ({ children }) => (
                <h2 className="mb-4 font-serif text-2xl font-semibold leading-tight text-[#30291f]">
                  {children}
                </h2>
              ),
              h2: ({ children }) => (
                <h3
                  id={sectionAnchor(children)}
                  className="mt-7 border-t border-[#b7a786]/60 pt-4 font-serif text-lg font-semibold text-[#30291f] first:mt-0 first:border-t-0 first:pt-0"
                >
                  {children}
                </h3>
              ),
              h3: ({ children }) => (
                <h4 className="mt-5 font-semibold text-[#806f57]">{children}</h4>
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
                <blockquote className="mt-3 border-l-4 border-[#a84632]/50 bg-[#f8f0e2] px-3 py-2 italic text-[#806f57]">
                  {children}
                </blockquote>
              ),
              code: ({ children }) => (
                <code className="rounded bg-[#f3ead9] px-1.5 py-0.5 font-mono text-[0.85em] text-[#8d321e]">
                  {children}
                </code>
              ),
              pre: ({ children }) => (
                <pre className="mt-3 overflow-x-auto rounded-lg border border-[#b7a786]/60 bg-[#30291f] p-3 font-mono text-xs leading-relaxed text-[#fffaf0]">
                  {children}
                </pre>
              ),
              table: ({ children }) => (
                <div className="mt-3 overflow-x-auto rounded-lg border border-[#b7a786]/60">
                  <table className="min-w-full border-collapse text-left text-xs">
                    {children}
                  </table>
                </div>
              ),
              th: ({ children }) => (
                <th className="border-b border-[#b7a786]/70 bg-[#f3ead9] px-2.5 py-2 font-semibold text-[#30291f]">
                  {children}
                </th>
              ),
              td: ({ children }) => (
                <td className="border-b border-[#b7a786]/40 px-2.5 py-2 align-top last:border-b-0">
                  {children}
                </td>
              ),
              hr: () => <hr className="my-5 border-[#b7a786]/60" />,
              a: ({ children, href }) => (
                <a
                  href={href}
                  target="_blank"
                  rel="noreferrer"
                  className="font-medium text-[#8d321e] underline decoration-[#a84632]/40 underline-offset-2 hover:text-[#a84632]"
                >
                  {children}
                </a>
              ),
              strong: ({ children }) => (
                <strong className="font-semibold text-[#30291f]">{children}</strong>
              ),
            }}
          >
            {rulesMarkdown}
          </ReactMarkdown>
        </div>
      ) : (
        <p className="rounded-lg border border-dashed border-[#b7a786] bg-[#f8f0e2] px-3 py-4 text-sm italic text-[#806f57]">
          {t('rules.loading')}
        </p>
      )}
    </section>
  )
}
