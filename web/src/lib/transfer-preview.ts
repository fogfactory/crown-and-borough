import { parseChainDraft } from '@/lib/chain-parse'

export function transferTargetsForTerritory(
  drafts: Record<string, string>,
  territoryID: string | null,
): string[] {
  if (!territoryID) return []

  const targets = new Set<string>()
  for (const draft of Object.values(drafts)) {
    for (const order of parseChainDraft(draft)) {
      if (order.type !== 'transfer' || order.position !== territoryID) continue
      const target = order.targets?.[0]
      if (target) targets.add(target)
    }
  }
  return [...targets]
}
