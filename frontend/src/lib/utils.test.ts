import { describe, it, expect } from 'vitest'
import { cn } from './utils'

describe('cn()', () => {
  it('menggabungkan class string biasa', () => {
    expect(cn('px-2', 'py-1')).toBe('px-2 py-1')
  })

  it('mengabaikan falsy value', () => {
    expect(cn('a', undefined, null as any, false as any, 'b')).toBe('a b')
  })

  it('mendukung object conditional class', () => {
    expect(cn('base', { active: true, disabled: false })).toBe('base active')
  })

  it('tailwind-merge: class terakhir menang untuk properti yang sama', () => {
    expect(cn('px-2', 'px-4')).toBe('px-4')
    expect(cn('text-sm font-bold', 'text-lg')).toBe('font-bold text-lg')
  })

  it('mengembalikan string kosong jika tidak ada argumen', () => {
    expect(cn()).toBe('')
  })
})
