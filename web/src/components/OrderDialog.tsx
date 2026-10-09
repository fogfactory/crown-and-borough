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
  /** Absent for a free text field. */
  options?: OrderFieldOption[]
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
  fields: OrderField[]
  /** Returns null while the values do not make a valid order. */
  buildOrder: (values: Record<string, string>) => BuiltOrder | null
  /** Receives the order line followed by its `# comment`. */
  onConfirm: (orderLine: string) => void
  onClose: () => void
}

function initialValues(fields: OrderField[]): Record<string, string> {
  return Object.fromEntries(
    fields.map((field) => [field.key, field.options?.[0]?.value ?? '']),
  )
}

function OrderDialogBody({
  title,
  description,
  fields,
  buildOrder,
  onConfirm,
  onClose,
}: OrderDialogProps) {
  const { t } = useLanguage()
  const [values, setValues] = useState(() => initialValues(fields))
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
        {fields.map((field) => (
          <label key={field.key} className="block space-y-1 text-xs font-semibold">
            <span>{field.label}</span>
            {field.options ? (
              <select
                value={values[field.key]}
                onChange={(event) =>
                  setValues({ ...values, [field.key]: event.target.value })
                }
                className="block w-full rounded border border-[#9bbbd3] bg-white px-2 py-1 font-normal"
              >
                {field.options.map((option) => (
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
        ))}
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
