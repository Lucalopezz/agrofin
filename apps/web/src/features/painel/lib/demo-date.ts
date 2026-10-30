export function formatDemoDate(value: string) {
  if (!value) return 'data não informada'
  // O horário local no meio do dia evita que a conversão do fuso mude o dia escolhido no formulário.
  return new Date(`${value}T12:00:00`).toLocaleDateString('pt-BR', { day: 'numeric', month: 'short' }).replace('.', '')
}
