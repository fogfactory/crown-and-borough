import { IconBrandGithub } from '@tabler/icons-react'

import { useLanguage } from '@/i18n/LanguageContext'
import { REPOSITORY_URL } from '@/lib/repository'
import { cn } from '@/lib/utils'

/** Icon link to the GitHub repository, opened in a new tab. */
export function GithubLink({ className }: { className?: string }) {
  const { t } = useLanguage()

  return (
    <a
      href={REPOSITORY_URL}
      target="_blank"
      rel="noopener noreferrer"
      aria-label={t('app.github')}
      title={t('app.github')}
      className={cn(
        'inline-flex size-11 shrink-0 items-center justify-center rounded-md text-ink-500 transition hover:bg-paper-100 hover:text-ink-900 lg:size-8',
        className,
      )}
    >
      <IconBrandGithub aria-hidden="true" className="size-5" />
    </a>
  )
}
