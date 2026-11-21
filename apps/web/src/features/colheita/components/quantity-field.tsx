import { FieldInput, FormField } from '@/features/painel/components/page-ui'
import type { BoxType } from '../types'

type Props = {
  type: BoxType
  value: string
  onChange: (value: string) => void
  hint?: string
  required?: boolean
}

export function QuantityField({ type, value, hint, required, onChange }: Props) {
  return (
    <FormField label={type === 'normal' ? 'Caixas normais' : 'Caixas de moagem'} hint={hint}>
      <FieldInput
        type="number"
        inputMode="numeric"
        min={required ? 1 : 0}
        max={2147483647}
        step={1}
        value={value}
        placeholder="0"
        required={required}
        onChange={(event) => onChange(event.target.value)}
      />
    </FormField>
  )
}
