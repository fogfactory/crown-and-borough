import ReactMarkdown from 'react-markdown'

import { useLanguage } from '@/i18n/LanguageContext'

const FAQ_ENTRIES = [
  { question: 'faq.q1', answer: 'faq.a1' },
  { question: 'faq.q2', answer: 'faq.a2' },
  { question: 'faq.q3', answer: 'faq.a3' },
  { question: 'faq.q4', answer: 'faq.a4' },
  { question: 'faq.q5', answer: 'faq.a5' },
  { question: 'faq.q6', answer: 'faq.a6' },
  { question: 'faq.q7', answer: 'faq.a7' },
  { question: 'faq.q8', answer: 'faq.a8' },
  { question: 'faq.q9', answer: 'faq.a9' },
] as const

export function FaqPanel({ showHeading = true }: { showHeading?: boolean }) {
  const { t } = useLanguage()

  return (
    <section
      aria-labelledby={showHeading ? 'faq-title' : undefined}
      aria-label={showHeading ? undefined : t('faq.title')}
      className="min-w-0"
    >
      {showHeading && (
        <div className="mb-6 max-w-3xl">
          <h2 id="faq-title" className="font-serif text-2xl font-semibold text-[#30291f]">
            {t('faq.title')}
          </h2>
          <p className="mt-2 text-sm leading-relaxed text-[#806f57]">{t('faq.intro')}</p>
        </div>
      )}

      <div className="max-w-3xl space-y-3">
        {FAQ_ENTRIES.map((entry) => (
          <details
            key={entry.question}
            open
            className="group rounded-xl border border-[#b7a786]/70 bg-[#f8f0e2] px-4 py-3 shadow-sm sm:px-5"
          >
            <summary className="cursor-pointer list-none pr-7 font-serif text-base font-semibold text-[#30291f] marker:hidden group-open:text-[#a84632] [&::-webkit-details-marker]:hidden">
              <span className="relative block after:absolute after:right-0 after:top-1/2 after:size-2 after:-translate-y-1/2 after:rotate-45 after:border-r-2 after:border-t-2 after:border-[#a84632] after:transition-transform group-open:after:rotate-[135deg]">
                {t(entry.question)}
              </span>
            </summary>
            <div className="mt-3 border-t border-[#b7a786]/50 pt-3 text-sm leading-relaxed text-[#594b3c]">
              <ReactMarkdown
                components={{
                  p: ({ children }) => <p className="mt-2 first:mt-0">{children}</p>,
                  ul: ({ children }) => (
                    <ul className="mt-2 list-disc space-y-1 pl-5">{children}</ul>
                  ),
                  ol: ({ children }) => (
                    <ol className="mt-2 list-decimal space-y-1 pl-5">{children}</ol>
                  ),
                  li: ({ children }) => <li className="pl-1">{children}</li>,
                  code: ({ children }) => (
                    <code className="rounded bg-[#f3ead9] px-1.5 py-0.5 font-mono text-[0.85em] text-[#8d321e]">
                      {children}
                    </code>
                  ),
                  strong: ({ children }) => (
                    <strong className="font-semibold text-[#30291f]">{children}</strong>
                  ),
                }}
              >
                {t(entry.answer)}
              </ReactMarkdown>
            </div>
          </details>
        ))}
      </div>
    </section>
  )
}
