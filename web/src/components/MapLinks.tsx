import { createContext, useContext, useEffect, useRef, type ReactNode } from 'react'

import { useLanguage } from '@/i18n/LanguageContext'
import type { EventMarks } from '@/lib/map-frame'
import { cn } from '@/lib/utils'

/**
 * What a page offers to the names it writes outside the order sheets
 * (charte-graphique.md § Identifiants et liens vers la carte): framing and
 * selecting a territory on the map, and lighting it while a link is hovered.
 */
export interface MapLinks {
  /** Frames and selects the territory; `noble` also gets the pawn halo. */
  focus: (territory: string, noble?: string) => void
  /**
   * Frames the map on a report event and draws it there until the next
   * selection (charte-graphique.md § Rapport de tour).
   */
  showEvent: (marks: EventMarks) => void
  /** Lights the territory on the map, or nothing with null. */
  highlight: (territory: string | null) => void
  /** Whether the territory exists on the map. */
  hasTerritory: (territory: string) => boolean
  /** Commune name of a territory. */
  territoryName: (territory: string) => string | undefined
  /** Full name of a noble the viewer knows, living or dead. */
  nobleName: (code: string) => string | undefined
  /** Territory where the viewer knows the noble stands, if any. */
  nobleTerritory: (code: string) => string | undefined
  /** Whether the noble is dead. */
  isDeceased: (code: string) => boolean
}

const MapLinksContext = createContext<MapLinks | null>(null)

export function MapLinksProvider({
  value,
  children,
}: {
  value: MapLinks
  children: ReactNode
}) {
  return <MapLinksContext.Provider value={value}>{children}</MapLinksContext.Provider>
}

export function useMapLinks(): MapLinks | null {
  return useContext(MapLinksContext)
}

/** A code after a full label: « Rochevent (RVE) ». */
export function CodeTag({ code }: { code: string }) {
  return <span className="font-mono text-[0.9em] text-ink-500">({code})</span>
}

const LINK_CLASS =
  'inline cursor-pointer rounded-sm p-0 text-left text-inherit underline decoration-brass-700 decoration-dotted underline-offset-2 hover:decoration-solid focus-visible:decoration-solid'

function MapLink({
  territory,
  noble,
  children,
}: {
  territory: string
  noble?: string
  children: ReactNode
}) {
  const links = useMapLinks()
  // Whether this link lit its territory, to put it out if it unmounts lit.
  const lit = useRef(false)
  const linksRef = useRef(links)
  linksRef.current = links
  useEffect(
    () => () => {
      if (lit.current) linksRef.current?.highlight(null)
    },
    [],
  )
  if (!links) return <span>{children}</span>
  const lightOn = () => {
    lit.current = true
    links.highlight(territory)
  }
  const lightOff = () => {
    lit.current = false
    links.highlight(null)
  }
  return (
    <button
      type="button"
      data-map-link={territory}
      className={LINK_CLASS}
      onClick={() => {
        lit.current = false
        links.focus(territory, noble)
      }}
      onMouseEnter={lightOn}
      onMouseLeave={lightOff}
      onFocus={lightOn}
      onBlur={lightOff}
    >
      {children}
    </button>
  )
}

/** A territory written in full, « Rochevent (RVE) », linked to the map. */
export function TerritoryRef({
  id,
  name,
  className,
}: {
  id: string
  /** Commune name; the code alone when unknown. */
  name?: string
  className?: string
}) {
  const links = useMapLinks()
  const fullName = name ?? links?.territoryName(id)
  const label = (
    <span className={className}>
      {fullName ? (
        <>
          {fullName} <CodeTag code={id} />
        </>
      ) : (
        id
      )}
    </span>
  )
  if (!links?.hasTerritory(id)) return label
  return <MapLink territory={id}>{label}</MapLink>
}

/**
 * A noble written in full, « Rainaud de Rochevent (RAI) », linked to the
 * territory where it stands. A dead noble, or one whose position the viewer
 * does not know, stays plain text with a tooltip.
 */
export function NobleRef({
  code,
  name,
  className,
}: {
  code: string
  name?: string
  className?: string
}) {
  const { t } = useLanguage()
  const links = useMapLinks()
  const fullName = name ?? links?.nobleName(code)
  const label = fullName ? (
    <>
      {fullName} <CodeTag code={code} />
    </>
  ) : (
    code
  )
  if (!links) return <span className={className}>{label}</span>
  const territory = links.nobleTerritory(code)
  if (links.isDeceased(code) || !territory) {
    return (
      <span
        className={cn(className, 'cursor-help')}
        title={t(links.isDeceased(code) ? 'links.deceased' : 'links.positionUnknown')}
      >
        {label}
      </span>
    )
  }
  return (
    <MapLink territory={territory} noble={code}>
      <span className={className}>{label}</span>
    </MapLink>
  )
}
