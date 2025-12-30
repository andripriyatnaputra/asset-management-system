import { createContext, useContext, useEffect, useMemo, useState } from 'react'

type Theme = 'light' | 'dark' | 'system'
type Ctx = { theme: Theme; setTheme: (t: Theme) => void; resolved: 'light'|'dark' }

const ThemeCtx = createContext<Ctx | null>(null)
const STORAGE_KEY = 'theme'

function applyTheme(theme: Theme) {
  const root = document.documentElement
  const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches
  const willBeDark = theme === 'dark' || (theme === 'system' && prefersDark)
  root.classList.toggle('dark', willBeDark)
}

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const [theme, setTheme] = useState<Theme>(() => (localStorage.getItem(STORAGE_KEY) as Theme) || 'system')
  const [prefersDark, setPrefersDark] = useState(
    typeof window !== 'undefined' && window.matchMedia('(prefers-color-scheme: dark)').matches
  )

  // react to user selection
  useEffect(() => {
    localStorage.setItem(STORAGE_KEY, theme)
    applyTheme(theme)
  }, [theme])

  // react to OS theme changes when on "system"
  useEffect(() => {
    const mq = window.matchMedia('(prefers-color-scheme: dark)')
    const onChange = () => {
      setPrefersDark(mq.matches)
      if ((localStorage.getItem(STORAGE_KEY) as Theme) === 'system') applyTheme('system')
    }
    mq.addEventListener('change', onChange)
    return () => mq.removeEventListener('change', onChange)
  }, [])

  const resolved: 'light'|'dark' = theme === 'dark' || (theme === 'system' && prefersDark) ? 'dark' : 'light'
  const value = useMemo(() => ({ theme, setTheme, resolved }), [theme, resolved])
  return <ThemeCtx.Provider value={value}>{children}</ThemeCtx.Provider>
}

export function useTheme() {
  const ctx = useContext(ThemeCtx)
  if (!ctx) throw new Error('useTheme must be used within ThemeProvider')
  return ctx
}
