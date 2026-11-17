import { useState } from 'react'

type Picker = (territory: string) => void

interface OrderPickerOptions {
  desktop: boolean
  /** Whether the orders are on screen (the drawer's Orders bookmark, or the phone's Orders tab). */
  ordersShown: boolean
  /** Brings the orders on screen (the phone's Orders tab). */
  showOrders: () => void
  /** The territory selected on the map, which the card's button fills the field with. */
  selectedId: string | null
}

/**
 * The order field waiting for a territory, shared by the map and the territory
 * card: a Ctrl/Cmd+click on the map and the card's « Use for the order »
 * button both fill it.
 */
export function useOrderPicker({
  desktop,
  ordersShown,
  showOrders,
  selectedId,
}: OrderPickerOptions) {
  const [picker, setPicker] = useState<Picker | null>(null)

  return {
    /** For `OrderFormProvider`. */
    onPickerChange: (next: Picker | null) => setPicker(() => next),
    /**
     * For the map's `onPick`: only while the orders are on screen, so that a
     * form left open behind another bookmark or tab does not swallow clicks.
     */
    pickOnMap: (id: string) => {
      if (!picker || !ordersShown) return false
      picker(id)
      return true
    },
    /**
     * For the card's button, undefined when no field waits or nothing is
     * selected. A phone offers it from the map tab too, then goes back to the
     * orders.
     */
    fillWithSelection:
      picker && selectedId && (!desktop || ordersShown)
        ? () => {
            picker(selectedId)
            if (!desktop) showOrders()
          }
        : undefined,
  }
}
