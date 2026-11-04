import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { GithubLink } from '@/components/GithubLink'
import { LanguageProvider } from '@/i18n/LanguageContext'
import { REPOSITORY_URL } from '@/lib/repository'

describe('GithubLink', () => {
  it('links to the repository in a new tab without leaking the opener', () => {
    render(
      <LanguageProvider>
        <GithubLink />
      </LanguageProvider>,
    )

    const link = screen.getByRole('link', { name: 'Source code on GitHub' })
    expect(link).toHaveAttribute('href', REPOSITORY_URL)
    expect(link).toHaveAttribute('target', '_blank')
    expect(link).toHaveAttribute('rel', 'noopener noreferrer')
    expect(link).toHaveAttribute('title', 'Source code on GitHub')
  })

  it('is labelled in French when the language is French', () => {
    render(
      <LanguageProvider initialLanguage="fr">
        <GithubLink />
      </LanguageProvider>,
    )

    expect(
      screen.getByRole('link', { name: 'Code source sur GitHub' }),
    ).toBeInTheDocument()
  })
})
