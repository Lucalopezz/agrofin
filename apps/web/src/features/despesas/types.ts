export type ExpenseInput = {
  paid_at: string
  category: string
  description: string
  amount_cents: number
}

export type Expense = ExpenseInput & { id: string }
