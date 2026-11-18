import { useState } from 'react'

import { HouseNameField } from '@/components/HouseNameField'
import { Alert } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { useLanguage } from '@/i18n/LanguageContext'
import { errorText } from '@/lib/api-error'

/**
 * Lets the player of an incognito game name their house, until their first
 * order submission locks the name.
 */
export function HouseNameForm({
  currentName,
  onRename,
}: {
  currentName: string
  onRename: (name: string) => Promise<void>
}) {
  const { t } = useLanguage()
  const [name, setName] = useState(currentName)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const trimmed = name.trim()

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setSaving(true)
    setError(null)
    try {
      await onRename(trimmed)
    } catch (failure) {
      setError(errorText(failure, t, t('error.serverUnavailable')))
    } finally {
      setSaving(false)
    }
  }

  return (
    <form
      className="space-y-2 rounded-lg border border-line/70 bg-paper-100 px-3 py-3"
      onSubmit={(event) => void handleSubmit(event)}
    >
      <HouseNameField
        value={name}
        onChange={setName}
        action={
          <Button
            type="submit"
            variant="outline"
            disabled={saving || trimmed === '' || trimmed === currentName}
          >
            {saving ? t('online.houseNameSaving') : t('online.houseNameSave')}
          </Button>
        }
      />
      {error && <Alert role="alert">{error}</Alert>}
    </form>
  )
}
