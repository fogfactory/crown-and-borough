import { useState, type ReactNode } from 'react'
import { Dialog } from 'radix-ui'

import { Button } from '@/components/ui/button'
import { useLanguage } from '@/i18n/LanguageContext'

export interface OrderFieldOption {
  value: string
  label: string
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
  const [chosen, setValues] = useState(() => initialValues(fields))
  // A select whose options changed with another field falls back to its first
  // option when its previous value is no longer offered.
  const values = { ...chosen }
  for (const field of fields) {
    const options = fieldOptions(field, values)
    if (options && !field.multi && !options.some((option) => option.value === values[field.key])) {
      values[field.key] = options[0]?.value ?? ''
    }
  }
  const order = buildOrder(values)

  return (
    <Dialog.Portal>
      <Dialog.Overlay className="fixed inset-0 z-50 bg-black/40" />
      <Dialog.Content className="fixed left-1/2 top-1/2 z-50 w-[min(92vw,26rem)] -translate-x-1/2 -translate-y-1/2 space-y-3 rounded-xl border border-[#9bbbd3] bg-[#f7fbff] p-4 text-[#263f52] shadow-xl">
        <Dialog.Title className="font-serif text-base font-bold text-[#2c5b7d]">
          {title}
        </Dialog.Title>
        <Dialog.Description className="text-xs leading-relaxed text-[#55738a]">
          {description ?? title}
        </Dialog.Description>
        {fields.map((field) => {
          const options = fieldOptions(field, values)
          if (field.multi && options) {
            const chosen = values[field.key].split(' ').filter(Boolean)
            return (
              <fieldset key={field.key} className="space-y-1 text-xs font-semibold">
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
                      {option.label}
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
                  onChange={(event) =>
                    setValues({ ...values, [field.key]: event.target.value })
                  }
                  className="block w-full rounded border border-[#9bbbd3] bg-white px-2 py-1 font-normal"
                >
                  {options.map((option) => (
                    <option key={option.value} value={option.value}>
                      {option.label}
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
                  className="block w-full rounded border border-[#9bbbd3] bg-white px-2 py-1 font-normal uppercase"
                />
              )}
            </label>
          )
        })}
        {hint && hint(values) && (
          <p className="rounded border border-[#c9dcea] bg-white/70 px-2 py-1 text-xs text-[#2c5b7d]">
            {hint(values)}
          </p>
        )}
        <p className="rounded bg-white/70 px-2 py-1 font-mono text-xs text-[#55738a]">
          {order ? `${order.line} # ${order.comment}` : '—'}
        </p>
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
              onConfirm(`${order.line} # ${order.comment}`)
              onClose()
            }}
          >
            {t('orders.dialogConfirm')}
          </Button>
        </div>
      </Dialog.Content>
    </Dialog.Portal>
  )
}

interface OrderLauncherProps extends Omit<OrderDialogProps, 'onClose'> {
  /** Text of the button that opens the dialog. */
  label: ReactNode
  ariaLabel?: string
  disabled?: boolean
  title: string
}

/**
 * A button opening a modal that configures one order (target, destination,
 * effect) and appends its line, with an inline comment, to the sheet.
 */
export function OrderLauncher({
  label,
  ariaLabel,
  disabled,
  ...dialog
}: OrderLauncherProps) {
  const [open, setOpen] = useState(false)
  return (
    <Dialog.Root open={open} onOpenChange={setOpen}>
      <Dialog.Trigger asChild>
        <Button
          type="button"
          variant="outline"
          size="sm"
          disabled={disabled}
          aria-label={ariaLabel}
        >
          {label}
        </Button>
      </Dialog.Trigger>
      {open && <OrderDialogBody {...dialog} onClose={() => setOpen(false)} />}
    </Dialog.Root>
  )
}
