import { render } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { translate } from '@/i18n/messages'
import { joinNodes, richTranslate } from '@/lib/rich-text'

const t = (
  key: Parameters<typeof translate>[1],
  values?: Record<string, string | number>,
) => translate('fr', key, values)

describe('richTranslate', () => {
  it('returns plain text when every value is text', () => {
    expect(richTranslate(t, 'reports.noble', { noble: 'JEA' })).toBe('Noble JEA')
  })

  it('splices nodes at their placeholders', () => {
    const { container } = render(
      <p>
        {richTranslate(t, 'reports.productionSent', {
          count: 3,
          territory: <strong>Bruyères (BRU)</strong>,
        })}
      </p>,
    )

    expect(container).toHaveTextContent('envoyé 3 vers Bruyères (BRU)')
    expect(container.querySelector('strong')).toHaveTextContent('Bruyères (BRU)')
  })
})

describe('joinNodes', () => {
  it('joins nodes with a separator', () => {
    const { container } = render(<p>{joinNodes([<b key="a">A</b>, 'B', 'C'], ' · ')}</p>)

    expect(container).toHaveTextContent('A · B · C')
  })
})
