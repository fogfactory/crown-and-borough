import type { ChangeEvent } from 'react'

import { Button } from '@/components/ui/button'
import type { Noble, PlayerId, StateData } from '@/types'

interface OrdersPanelProps {
  state: StateData
  player: PlayerId
  chainDrafts: Record<string, string>
  winterDraft: string
  submitted: boolean
  submitting: boolean
  onChainChange: (noble: string, text: string) => void
  onWinterChange: (text: string) => void
  onSubmit: () => void
}

function ownedNobles(state: StateData, player: PlayerId): Noble[] {
  return state.nobles.filter((noble) => noble.owner === player)
}

function chainPlaceholder(): string {
  return 'XXX A YYY'
}

export function OrdersPanel({
  state,
  player,
  chainDrafts,
  winterDraft,
  submitted,
  submitting,
  onChainChange,
  onWinterChange,
  onSubmit,
}: OrdersPanelProps) {
  const handleChainChange = (noble: Noble) => (event: ChangeEvent<HTMLTextAreaElement>) => {
    onChainChange(noble.code, event.target.value)
  }

  if (state.season === 'winter') {
    return (
      <section className="space-y-3 border-t border-[#b7a786]/50 pt-5">
        <div>
          <h3 className="font-serif text-lg font-semibold">Ordres d&apos;hiver</h3>
          <p className="mt-1 text-xs leading-relaxed text-[#806f57]">
            Une ligne par investissement. Les ordres sont appliqués dans l&apos;ordre saisi. La résolution attend tous les joueurs.
          </p>
        </div>
        <textarea
          value={winterDraft}
          onChange={(event) => onWinterChange(event.target.value)}
          className="min-h-36 w-full resize-y rounded-lg border border-[#b7a786] bg-[#f8f0e2] p-3 font-mono text-xs text-[#30291f] outline-none transition focus:border-[#a84632] focus:ring-2 focus:ring-[#a84632]/20"
          placeholder={'R T ROS\nC M ROS\nE C ROS'}
          aria-label={`Ordres d'hiver de ${player}`}
        />
        {submitted && <p className="text-xs text-[#376341]">Ordres soumis. Vous pouvez encore les modifier.</p>}
        <Button type="button" className="w-full" disabled={submitting} onClick={onSubmit}>
          {submitting ? 'Envoi…' : submitted ? 'Modifier' : 'Soumettre'}
        </Button>
      </section>
    )
  }

  const nobles = ownedNobles(state, player)
  return (
    <section className="space-y-3 border-t border-[#b7a786]/50 pt-5">
      <div>
          <h3 className="font-serif text-lg font-semibold">Chaînes d&apos;ordres</h3>
          <p className="mt-1 text-xs leading-relaxed text-[#806f57]">
            Une chaîne par noble. L&apos;en-tête du noble est ajouté automatiquement avant l&apos;envoi. Une zone vide signifie que le noble n&apos;émet pas ce tour.
        </p>
      </div>
      {nobles.length === 0 ? (
        <p className="rounded-lg border border-dashed border-[#b7a786] bg-[#f8f0e2] p-3 text-sm italic text-[#806f57]">
          Aucun noble disponible pour ce joueur.
        </p>
      ) : (
        nobles.map((noble) => (
          <label key={noble.code} className="block space-y-1.5">
            <span className="flex items-center justify-between text-xs font-semibold uppercase tracking-[0.12em] text-[#806f57]">
              <span>{noble.code} · {noble.name}</span>
              <span className={noble.status === 'free' ? 'text-[#376341]' : 'text-[#a84632]'}>
                {noble.status === 'free' ? 'Libre' : noble.status}
              </span>
            </span>
            <textarea
              value={chainDrafts[noble.code] ?? ''}
              onChange={handleChainChange(noble)}
              disabled={noble.status !== 'free'}
              className="min-h-32 w-full resize-y rounded-lg border border-[#b7a786] bg-[#f8f0e2] p-3 font-mono text-xs text-[#30291f] outline-none transition focus:border-[#a84632] focus:ring-2 focus:ring-[#a84632]/20 disabled:cursor-not-allowed disabled:opacity-50"
              placeholder={chainPlaceholder()}
              aria-label={`Chaîne de ${noble.code}`}
            />
          </label>
        ))
      )}
      {submitted && <p className="text-xs text-[#376341]">Ordres soumis. Vous pouvez encore les modifier.</p>}
      <Button type="button" className="w-full" disabled={submitting} onClick={onSubmit}>
        {submitting ? 'Envoi…' : submitted ? 'Modifier' : 'Soumettre'}
      </Button>
    </section>
  )
}
