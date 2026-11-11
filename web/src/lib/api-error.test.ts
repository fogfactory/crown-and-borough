import { describe, expect, it } from 'vitest'

import { translate } from '@/i18n/messages'
import { ApiError } from '@/lib/api'
import { apiErrorMessage, errorText } from '@/lib/api-error'

const fr = (
  key: Parameters<typeof translate>[1],
  values?: Parameters<typeof translate>[2],
) => translate('fr', key, values)

describe('api errors', () => {
  it('translates a known API error code instead of the server message', () => {
    const error = new ApiError(409, 'game_full', 'game is full')
    expect(apiErrorMessage(error, fr)).toBe('Cette partie est complète.')
  })

  it('falls back to the HTTP status for an unknown code', () => {
    const error = new ApiError(500, 'something_new', 'boom')
    expect(apiErrorMessage(error, fr)).toBe('La requête a échoué (500)')
  })

  it('never exposes the English message of an API error', () => {
    const error = new ApiError(400, 'invalid_orders', 'line 3: unknown order')
    expect(errorText(error, fr, 'x')).not.toContain('line 3')
  })

  it('keeps plain errors and the fallback', () => {
    expect(errorText(new Error('network down'), fr, 'x')).toBe('network down')
    expect(errorText('oops', fr, 'repli')).toBe('repli')
  })
})

describe('api errors with a localized detail', () => {
  it('keeps the server message of an order rejection', () => {
    const error = new ApiError(400, 'invalid_orders', 'Ligne 2 : armée inconnue', {
      error: 'invalid_orders',
      errors: [{ line: 2, message: 'Ligne 2 : armée inconnue' }],
    })
    expect(errorText(error, fr, 'x')).toBe('Ligne 2 : armée inconnue')
  })
})
