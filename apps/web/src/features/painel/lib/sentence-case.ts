export function sentenceCase(value: string) {
  return value.replace(/^(\s*\p{Ll})/u, firstLetter => firstLetter.toLocaleUpperCase('pt-BR'))
}
