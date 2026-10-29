import { clsx, type ClassValue } from 'clsx'
import { twMerge } from 'tailwind-merge'

// Resolve conflitos entre classes padrão dos componentes e ajustes de cada tela.
export function cn(...inputs: ClassValue[]) { return twMerge(clsx(inputs)) }
