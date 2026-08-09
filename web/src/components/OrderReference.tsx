import { BookOpen } from 'lucide-react'

import type { Season } from '@/types'
import {
  ACTION_ORDER_REFERENCES,
  isActionSeason,
  LIAISON_MODE_REFERENCES,
  SPECIAL_ORDER_NOTES,
  type OrderReferenceEntry,
  type OrderReferenceNote,
  WINTER_INVESTMENT_REFERENCES,
  WINTER_REFERENCE_NOTES,
} from '@/lib/order-reference'

interface OrderReferenceProps {
  season: Season
}

function formatCost(cost: number | null): string {
  return cost === null ? 'Aucun' : `${cost} R`
}

function Examples({
  examples,
}: {
  examples: readonly { syntax: string; description: string }[]
}) {
  if (examples.length === 0) return null

  return (
    <div className="mt-3 border-t border-[#b7a786]/40 pt-3">
      <p className="text-[10px] font-bold uppercase tracking-[0.16em] text-[#806f57]">
        Exemples
      </p>
      <ul className="mt-1.5 space-y-1.5">
        {examples.map((example) => (
          <li
            key={`${example.syntax}-${example.description}`}
            className="text-xs text-[#594b3c]"
          >
            <code className="break-all rounded bg-[#f3ead9] px-1.5 py-0.5 font-mono text-[11px] text-[#30291f]">
              {example.syntax}
            </code>{' '}
            <span>{example.description}</span>
          </li>
        ))}
      </ul>
    </div>
  )
}

function ReferenceEntry({ entry }: { entry: OrderReferenceEntry }) {
  return (
    <article className="rounded-lg border border-[#b7a786]/60 bg-[#fffaf0] p-3">
      <div className="flex items-start gap-2.5">
        <span className="flex size-7 shrink-0 items-center justify-center rounded-md bg-[#f8e5dd] font-mono text-sm font-bold text-[#a84632]">
          {entry.symbol}
        </span>
        <div className="min-w-0">
          <h5 className="text-sm font-semibold text-[#30291f]">{entry.name}</h5>
          <code className="mt-1 block break-all font-mono text-xs text-[#8d321e]">
            {entry.syntax}
          </code>
        </div>
      </div>
      <dl className="mt-3 grid gap-2 border-t border-[#b7a786]/40 pt-3 text-xs text-[#594b3c]">
        <div>
          <dt className="font-semibold text-[#806f57]">Condition</dt>
          <dd className="mt-0.5">{entry.condition}</dd>
        </div>
        <div>
          <dt className="font-semibold text-[#806f57]">Portée</dt>
          <dd className="mt-0.5">{entry.scope}</dd>
        </div>
        <div>
          <dt className="font-semibold text-[#806f57]">Coût</dt>
          <dd className="mt-0.5">{formatCost(entry.cost)}</dd>
        </div>
        <div>
          <dt className="font-semibold text-[#806f57]">Résultat</dt>
          <dd className="mt-0.5">{entry.result}</dd>
        </div>
      </dl>
      <Examples examples={entry.examples} />
    </article>
  )
}

function Notes({ notes }: { notes: readonly OrderReferenceNote[] }) {
  return (
    <div className="space-y-2">
      {notes.map((note) => (
        <article
          key={note.title}
          className="rounded-lg border border-[#b7a786]/50 bg-[#f8f0e2] p-3"
        >
          <h5 className="text-xs font-bold uppercase tracking-[0.12em] text-[#806f57]">
            {note.title}
          </h5>
          <p className="mt-1 text-xs leading-relaxed text-[#594b3c]">
            {note.description}
          </p>
          {note.examples && <Examples examples={note.examples} />}
        </article>
      ))}
    </div>
  )
}

function ReferenceSection({
  title,
  entries,
}: {
  title: string
  entries: readonly OrderReferenceEntry[]
}) {
  return (
    <section className="space-y-2">
      <h4 className="text-xs font-bold uppercase tracking-[0.16em] text-[#806f57]">
        {title}
      </h4>
      <div className="space-y-2">
        {entries.map((entry) => (
          <ReferenceEntry key={entry.symbol} entry={entry} />
        ))}
      </div>
    </section>
  )
}

export function OrderReference({ season }: OrderReferenceProps) {
  const actionSeason = isActionSeason(season)
  const entries = actionSeason ? ACTION_ORDER_REFERENCES : WINTER_INVESTMENT_REFERENCES

  return (
    <details className="overflow-hidden rounded-xl border border-[#b7a786] bg-[#f3ead9]/70">
      <summary className="cursor-pointer px-3 py-3 text-sm font-semibold text-[#30291f] transition hover:bg-[#f3ead9]">
        <span className="inline-flex items-center gap-2">
          <BookOpen aria-hidden="true" className="size-4 text-[#a84632]" />
          Aide-mémoire des ordres
        </span>
      </summary>
      <div className="max-h-[70vh] space-y-4 overflow-y-auto border-t border-[#b7a786]/60 p-3">
        <p className="text-xs leading-relaxed text-[#806f57]">
          {actionSeason
            ? 'Les ordres ci-dessous sont disponibles au printemps, en été et en automne.'
            : "L'hiver est une phase d'investissements directs : aucune chaîne, aucun mouvement ni ravitaillement."}
        </p>
        <ReferenceSection
          title={actionSeason ? "Ordres des saisons d'action" : 'Investissements d’hiver'}
          entries={entries}
        />
        {actionSeason ? (
          <>
            <section className="space-y-2 border-t border-[#b7a786]/50 pt-4">
              <h4 className="text-xs font-bold uppercase tracking-[0.16em] text-[#806f57]">
                Liaison des ordres
              </h4>
              <Notes notes={LIAISON_MODE_REFERENCES} />
            </section>
            <section className="space-y-2 border-t border-[#b7a786]/50 pt-4">
              <h4 className="text-xs font-bold uppercase tracking-[0.16em] text-[#806f57]">
                Particularités
              </h4>
              <Notes notes={SPECIAL_ORDER_NOTES} />
            </section>
          </>
        ) : (
          <section className="space-y-2 border-t border-[#b7a786]/50 pt-4">
            <h4 className="text-xs font-bold uppercase tracking-[0.16em] text-[#806f57]">
              Règles de la phase d’hiver
            </h4>
            <Notes notes={WINTER_REFERENCE_NOTES} />
          </section>
        )}
      </div>
    </details>
  )
}
