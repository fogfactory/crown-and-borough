import {
  createContext,
  useContext,
  useEffect,
  useId,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from 'react'

import { Button } from '@/components/ui/button'
import { useMapLinks, type MapLinks } from '@/components/MapLinks'
import { SectionHeading } from '@/components/SectionHeading'
import { useLanguage } from '@/i18n/LanguageContext'

export interface OrderFieldOption {
  value: string
  label: string
  /** The value is a territory code: the list writes « Ravinval (RVN) ». */
  territory?: boolean
}

/** One input of an order dialog: a choice among options, or a short free code. */
export interface OrderField {
  key: string
  label: string
  /** Absent for a free text field; a function when it depends on other fields. */
  options?: OrderFieldOption[] | ((values: Record<string, string>) => OrderFieldOption[])
  /** Checkboxes instead of a select: the value is the space-joined codes. */
  multi?: boolean
  placeholder?: string
  maxLength?: number
  /** Skips the field, and its value, while the other values make it pointless. */
  hidden?: (values: Record<string, string>) => boolean
}

/** The order a dialog produces: its line and the inline comment that recalls it. */
export interface BuiltOrder {
  line: string
  comment: string
}

interface OrderDialogProps {
  title: string
  description?: string
  /** Extra line under the fields, read from the current values. */
  hint?: (values: Record<string, string>) => string | null
  fields: OrderField[]
  /** Returns null while the values do not make a valid order. */
  buildOrder: (values: Record<string, string>) => BuiltOrder | null
  /** Receives the order line followed by its `# comment`. */
  onConfirm: (orderLine: string) => void
  onClose: () => void
}

function fieldOptions(
  field: OrderField,
  values: Record<string, string>,
): OrderFieldOption[] | undefined {
  return typeof field.options === 'function' ? field.options(values) : field.options
}

function initialValues(fields: OrderField[]): Record<string, string> {
  const values: Record<string, string> = {}
  for (const field of fields) {
    values[field.key] = field.multi ? '' : (fieldOptions(field, values)?.[0]?.value ?? '')
  }
  return values
}

/** Shares the open form and the map picks between the order forms of a page. */
interface OrderFormHub {
  openId: string | null
  setOpenId: (id: string | null) => void
  /** Hands map clicks to the open form's active field, or stops with null. */
  setPicker: (picker: ((territory: string) => void) | null) => void
}

const OrderFormContext = createContext<OrderFormHub | null>(null)

/**
 * Lets the order forms of a page share one opening at a time and receive the
 * territory clicked on the map (charte-graphique.md § Poste de commandement).
 * `onPickerChange` gets the open form's picker, or null when none listens.
 */
export function OrderFormProvider({
  children,
  onPickerChange,
}: {
  children: ReactNode
  onPickerChange: (picker: ((territory: string) => void) | null) => void
}) {
  const [openId, setOpenId] = useState<string | null>(null)
  const pickerChange = useRef(onPickerChange)
  pickerChange.current = onPickerChange
  const hub = useMemo<OrderFormHub>(
    () => ({
      openId,
      setOpenId,
      setPicker: (picker) => pickerChange.current(picker),
    }),
    [openId],
  )
  return <OrderFormContext.Provider value={hub}>{children}</OrderFormContext.Provider>
}

/** A territory code followed by nothing that already names it. */
const BARE_CODE = /(?<![(\w])([A-Z]{3})(?![\w)])/g

/**
 * The comment of an order line in full labels: each bare territory code reads
 * « Rochevent (RVE) » (charte-graphique.md § Identifiants et liens vers la carte).
 */
export function fullLabelComment(comment: string, links: MapLinks | null): string {
  if (!links) return comment
  return comment.replace(BARE_CODE, (code: string) => {
    const name = links.hasTerritory(code) ? links.territoryName(code) : undefined
    return name ? `${name} (${code})` : code
  })
}

function OrderDialogBody({
  title,
  description,
  hint,
  fields,
  buildOrder,
  onConfirm,
  onClose,
}: OrderDialogProps) {
  const { t } = useLanguage()
  const links = useMapLinks()
  const hub = useContext(OrderFormContext)
  // A territory option reads « Ravinval (RVN) », not its bare code
  // (charte-graphique.md § Identifiants et liens vers la carte).
  const optionLabel = (option: OrderFieldOption) => {
    const name = option.territory ? links?.territoryName(option.value) : undefined
    return name ? `${name} (${option.value})` : option.label
  }
  const [chosen, setValues] = useState(() => initialValues(fields))
  // A select whose options changed with another field falls back to its first
  // option when its previous value is no longer offered.
  const values = { ...chosen }
  for (const field of fields) {
    const options = fieldOptions(field, values)
    if (
      options &&
      !field.multi &&
      !options.some((option) => option.value === values[field.key])
    ) {
      values[field.key] = options[0]?.value ?? ''
    }
  }
  const order = buildOrder(values)

  // The option a click on the map's territory stands for: the territory
  // itself, or a noble standing there.
  const pickedOption = (options: OrderFieldOption[], territory: string) =>
    options.find((option) =>
      option.territory
        ? option.value === territory
        : links?.nobleTerritory(option.value) === territory,
    )
  const pickable = (field: OrderField) => {
    if (field.hidden?.(values)) return false
    const options = fieldOptions(field, values)
    return Boolean(
      options?.some((option) => option.territory || links?.nobleTerritory(option.value)),
    )
  }
  const [activeKey, setActiveKey] = useState<string | null>(null)
  const active =
    fields.find((field) => field.key === activeKey && pickable(field)) ??
    fields.find(pickable)

  // The open form takes the map's clicks for its active field.
  const pick = (territory: string) => {
    if (!active) return
    const options = fieldOptions(active, values) ?? []
    const option = pickedOption(options, territory)
    if (!option) return
    if (active.multi) {
      const picked = values[active.key].split(' ').filter(Boolean)
      setValues({
        ...values,
        [active.key]: (picked.includes(option.value)
          ? picked.filter((code) => code !== option.value)
          : [...picked, option.value]
        ).join(' '),
      })
    } else {
      setValues({ ...values, [active.key]: option.value })
    }
  }
  const pickRef = useRef(pick)
  pickRef.current = pick
  useEffect(() => {
    if (!hub) return
    hub.setPicker((territory) => pickRef.current(territory))
    return () => hub.setPicker(null)
  }, [hub])

  const ring = (field: OrderField) =>
    active?.key === field.key ? 'ring-2 ring-brass-700' : ''
  const comment = order ? fullLabelComment(order.comment, links) : ''

  return (
    <div
      role="dialog"
      aria-modal="false"
      aria-label={title}
      data-order-form
      onKeyDown={(event) => {
        if (event.key !== 'Escape') return
        event.stopPropagation()
        onClose()
      }}
      className="basis-full space-y-3 rounded-md border border-line bg-paper-50 p-3 text-ink-700 shadow-panel"
    >
      <SectionHeading
        title={title}
        topic={title}
        help={description ?? title}
        titleClassName="font-display text-sm font-semibold text-ink-900"
      />
      {fields.map((field) => {
        if (field.hidden?.(values)) return null
        const options = fieldOptions(field, values)
        if (field.multi && options) {
          const chosen = values[field.key].split(' ').filter(Boolean)
          return (
            <fieldset
              key={field.key}
              data-active-field={active?.key === field.key || undefined}
              className={`space-y-1 rounded-md p-1 text-xs font-semibold ${ring(field)}`}
              onFocus={() => setActiveKey(field.key)}
            >
              <legend>{field.label}</legend>
              <div className="flex flex-wrap gap-x-3 gap-y-1 font-normal">
                {options.map((option) => (
                  <label key={option.value} className="flex items-center gap-1">
                    <input
                      type="checkbox"
                      checked={chosen.includes(option.value)}
                      onChange={(event) =>
                        setValues({
                          ...values,
                          [field.key]: (event.target.checked
                            ? [...chosen, option.value]
                            : chosen.filter((code) => code !== option.value)
                          ).join(' '),
                        })
                      }
                    />
                    {optionLabel(option)}
                  </label>
                ))}
              </div>
            </fieldset>
          )
        }
        return (
          <label key={field.key} className="block space-y-1 text-xs font-semibold">
            <span>{field.label}</span>
            {options ? (
              <select
                value={values[field.key]}
                data-active-field={active?.key === field.key || undefined}
                onFocus={() => setActiveKey(field.key)}
                onChange={(event) =>
                  setValues({ ...values, [field.key]: event.target.value })
                }
                className={`block w-full rounded border border-line-strong bg-paper-50 px-2 py-1 font-normal ${ring(field)}`}
              >
                {options.map((option) => (
                  <option key={option.value} value={option.value}>
                    {optionLabel(option)}
                  </option>
                ))}
              </select>
            ) : (
              <input
                value={values[field.key]}
                maxLength={field.maxLength}
                placeholder={field.placeholder}
                onChange={(event) =>
                  setValues({ ...values, [field.key]: event.target.value })
                }
                className="block w-full rounded border border-line-strong bg-paper-50 px-2 py-1 font-normal uppercase"
              />
            )}
          </label>
        )
      })}
      {active && <p className="text-2xs text-ink-500">{t('orders.form.mapPick')}</p>}
      {hint && hint(values) && (
        <p className="rounded border border-line bg-paper-100 px-2 py-1 text-xs text-ink-700">
          {hint(values)}
        </p>
      )}
      <div data-order-preview className="rounded bg-paper-100 px-2 py-1 text-xs">
        <p className="font-mono text-ink-900">{order ? order.line : '—'}</p>
        {order && <p className="text-ink-500">{comment}</p>}
      </div>
      <div className="flex justify-end gap-2">
        <Button type="button" variant="outline" size="sm" onClick={onClose}>
          {t('orders.dialogCancel')}
        </Button>
        <Button
          type="button"
          size="sm"
          disabled={order === null}
          onClick={() => {
            if (!order) return
            onConfirm(`${order.line} # ${comment}`)
            onClose()
          }}
        >
          {t('orders.dialogConfirm')}
        </Button>
      </div>
    </div>
  )
}

export interface OrderLauncherProps extends Omit<OrderDialogProps, 'onClose'> {
  /** Text of the button that opens the form. */
  label: ReactNode
  /** Tooltip of the button; the form title by default. */
  tooltip?: string
  ariaLabel?: string
  disabled?: boolean
  title: string
}

/**
 * A button unfolding, in the drawer right under it, the form that configures
 * one order (target, destination, effect) and appends its line, with an
 * inline comment, to the sheet. Inside an `OrderFormProvider` only one form is
 * open at a time. The parent lays the launchers out in a wrapping flex row, so
 * the form takes a line of its own.
 */
export function OrderLauncher({
  label,
  tooltip,
  ariaLabel,
  disabled,
  ...form
}: OrderLauncherProps) {
  const hub = useContext(OrderFormContext)
  const id = useId()
  const [ownOpen, setOwnOpen] = useState(false)
  const open = hub ? hub.openId === id : ownOpen
  const setOpen = (next: boolean) => {
    if (hub) hub.setOpenId(next ? id : null)
    else setOwnOpen(next)
  }
  return (
    <>
      <Button
        type="button"
        variant="outline"
        size="sm"
        disabled={disabled}
        aria-label={ariaLabel}
        aria-expanded={open}
        title={tooltip ?? form.title}
        // A long label (a couple's names) wraps instead of leaving the drawer.
        className="h-auto min-h-8 max-w-full whitespace-normal py-1 text-left"
        onClick={() => setOpen(!open)}
      >
        {label}
      </Button>
      {open && <OrderDialogBody {...form} onClose={() => setOpen(false)} />}
    </>
  )
}
