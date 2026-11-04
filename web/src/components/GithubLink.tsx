import { IconBrandGithub } from '@tabler/icons-react'

import { Button } from '@/components/ui/button'
import { useLanguage } from '@/i18n/LanguageContext'
import { REPOSITORY_URL } from '@/lib/repository'
import { cn } from '@/lib/utils'

/** Icon link to the GitHub repository, opened in a new tab. */
export function GithubLink({
  variant = 'ghost',
  className,
}: {
  variant?: 'ghost' | 'outline'
  className?: string
}) {
  const { t } = useLanguage()

  return (
    <Button
      asChild
      variant={variant}
      size="icon"
      className={cn(
        'size-11 lg:size-8',
        variant === 'ghost' && 'text-ink-500',
        className,
      )}
    >
      <a
        href={REPOSITORY_URL}
        target="_blank"
        rel="noopener noreferrer"
        aria-label={t('app.github')}
        title={t('app.github')}
      >
        <IconBrandGithub aria-hidden="true" className="size-5" />
      </a>
    </Button>
  )
}
