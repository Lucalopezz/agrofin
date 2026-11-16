import type { ComponentProps } from 'react'

type AuthFieldProps = ComponentProps<'input'> & {
  label: string
}

export function AuthField({ label, ...inputProps }: AuthFieldProps) {
  return (
    <label>
      {label}
      <input {...inputProps} />
    </label>
  )
}
