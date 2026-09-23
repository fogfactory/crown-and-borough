import { useEffect, useMemo, useState } from 'react'

import { hasSupplySource } from '@/lib/supply'
import { transferTargetsForTerritory } from '@/lib/transfer-preview'
import type {
  PlayerId,
  StateData,
  SupplyLine,
  TerritoryState,
  TransferLine,
} from '@/types'

export type SupplyFetcher = <T>(path: string, signal: AbortSignal) => Promise<T>

export interface UseSupplyAndTransferOptions {
  selectedId: string | null
  state: StateData | null
  selectedState: TerritoryState | null | undefined
  chainDrafts: Record<string, string>
  /** Player whose drafts are shown; transfer targets require owning the army. */
  ownerId: PlayerId | null
  /** Resource root for the supply endpoint ('/api' hotseat, '/api/games/{id}' online). */
  basePath?: string
  /**
   * Performs the GET for a supply path. Throws on failure; an error with
   * `status === 401` triggers `onAuthError` (online sessions only).
   */
  fetcher: SupplyFetcher
  /** Translated message used when the request fails with a non-401 error. */
  networkErrorMessage: string
  onAuthError?: () => void
}

export interface SupplyAndTransferState {
  supplyLine: SupplyLine | null
  supplyLoading: boolean
  supplyError: string | null
  transferLine: TransferLine | null
  transferLoading: boolean
  transferError: string | null
  selectedSupplyLine: SupplyLine | null
  /** Origin of the selected supply line, to resolve against the map. */
  sourceTerritoryId: string | null
  transferTargets: string[]
  transferTarget: string | null
  setTransferTarget: (target: string | null) => void
}

function isUnauthorized(error: unknown): boolean {
  return (
    typeof error === 'object' &&
    error !== null &&
    (error as { status?: unknown }).status === 401
  )
}

/**
 * Shared supply and transfer previews for the hotseat and online screens.
 * The fetcher abstracts the transport (plain hotseat fetch vs authenticated
 * online API call); everything else behaves identically in both modes.
 */
export function useSupplyAndTransfer({
  selectedId,
  state,
  selectedState,
  chainDrafts,
  ownerId,
  basePath = '/api',
  fetcher,
  networkErrorMessage,
  onAuthError,
}: UseSupplyAndTransferOptions): SupplyAndTransferState {
  const [supplyLine, setSupplyLine] = useState<SupplyLine | null>(null)
  const [supplyLoading, setSupplyLoading] = useState(false)
  const [supplyError, setSupplyError] = useState<string | null>(null)
  const [transferLine, setTransferLine] = useState<TransferLine | null>(null)
  const [transferLoading, setTransferLoading] = useState(false)
  const [transferError, setTransferError] = useState<string | null>(null)
  const [selectedTransferTarget, setSelectedTransferTarget] = useState<string | null>(
    null,
  )

  const transferTargets = useMemo(
    () =>
      selectedState?.army?.owner === ownerId
        ? transferTargetsForTerritory(chainDrafts, selectedId)
        : [],
    [chainDrafts, ownerId, selectedId, selectedState?.army?.owner],
  )
  const transferTarget = transferTargets.includes(selectedTransferTarget ?? '')
    ? selectedTransferTarget
    : (transferTargets[0] ?? null)

  useEffect(() => {
    if (
      !selectedId ||
      !state ||
      (!selectedState?.army && !hasSupplySource(selectedState ?? undefined)) ||
      state.season === 'winter'
    ) {
      setSupplyLine(null)
      setSupplyError(null)
      setSupplyLoading(false)
      return
    }
    const controller = new AbortController()
    setSupplyLoading(true)
    setSupplyError(null)
    fetcher<SupplyLine>(
      `${basePath}/supply?territory=${encodeURIComponent(selectedId)}`,
      controller.signal,
    )
      .then((line) => {
        if (!controller.signal.aborted) setSupplyLine(line)
      })
      .catch((supplyFailure: unknown) => {
        if (controller.signal.aborted) return
        if (isUnauthorized(supplyFailure)) {
          onAuthError?.()
          return
        }
        setSupplyError(
          supplyFailure instanceof Error ? supplyFailure.message : networkErrorMessage,
        )
      })
      .finally(() => {
        if (!controller.signal.aborted) setSupplyLoading(false)
      })
    return () => controller.abort()
  }, [
    basePath,
    fetcher,
    networkErrorMessage,
    onAuthError,
    selectedId,
    selectedState,
    state,
  ])

  useEffect(() => {
    if (
      !selectedId ||
      !state ||
      !selectedState?.army ||
      !transferTarget ||
      state.season === 'winter'
    ) {
      setTransferLine(null)
      setTransferError(null)
      setTransferLoading(false)
      return
    }

    const controller = new AbortController()
    setTransferLine(null)
    setTransferError(null)
    setTransferLoading(true)
    fetcher<TransferLine>(
      `${basePath}/supply?territory=${encodeURIComponent(selectedId)}&target=${encodeURIComponent(transferTarget)}`,
      controller.signal,
    )
      .then((line) => {
        if (!controller.signal.aborted) setTransferLine(line)
      })
      .catch((transferFailure: unknown) => {
        if (controller.signal.aborted) return
        if (isUnauthorized(transferFailure)) {
          onAuthError?.()
          return
        }
        setTransferError(
          transferFailure instanceof Error
            ? transferFailure.message
            : networkErrorMessage,
        )
      })
      .finally(() => {
        if (!controller.signal.aborted) setTransferLoading(false)
      })
    return () => controller.abort()
  }, [
    basePath,
    fetcher,
    networkErrorMessage,
    onAuthError,
    selectedId,
    selectedState,
    state,
    transferTarget,
  ])

  const supplySelectionAllowed =
    (supplyLine?.kind === 'army' && Boolean(selectedState?.army)) ||
    (supplyLine?.kind === 'source' &&
      !selectedState?.army &&
      hasSupplySource(selectedState ?? undefined))
  const selectedSupplyLine =
    supplySelectionAllowed && supplyLine?.territory === selectedId ? supplyLine : null

  return {
    supplyLine,
    supplyLoading,
    supplyError,
    transferLine,
    transferLoading,
    transferError,
    selectedSupplyLine,
    sourceTerritoryId: selectedSupplyLine?.source ?? null,
    transferTargets,
    transferTarget,
    setTransferTarget: setSelectedTransferTarget,
  }
}
