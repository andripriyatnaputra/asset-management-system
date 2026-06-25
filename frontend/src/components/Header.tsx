// =========================
// 📁 File: src/components/layout/Header.tsx
// =========================
import { useMemo } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { jwtDecode } from 'jwt-decode'
import { toast } from 'sonner'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuLabel,
  DropdownMenuItem,
  DropdownMenuSeparator,
} from '@/components/ui/dropdown-menu'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Sun, Moon, Search, LogOut, KeyRound, User } from 'lucide-react'
import { useAuthContext } from '@/context/AuthContext'
import NotificationBell from '@/components/NotificationBell'

type Decoded = { name?: string; email?: string; role?: string }

export default function Header() {
  const navigate = useNavigate()
  const { logout, token } = useAuthContext()

  const user = useMemo<Decoded>(() => {
    if (!token) return {}
    try {
      return jwtDecode(token) as Decoded
    } catch {
      return {}
    }
  }, [token])

  const initials = (user.name || user.email || 'U')
    .split(' ')
    .map((s) => s[0])
    .join('')
    .slice(0, 2)
    .toUpperCase()

  const handleLogout = () => {
    logout()
    toast.success('Anda telah logout.')
    navigate('/login', { replace: true })
  }

  const handleChangePassword = () => navigate('/change-password')

  const toggleTheme = () => {
    const root = document.documentElement
    const isDark = root.classList.contains('dark')
    root.classList.toggle('dark', !isDark)
    localStorage.setItem('theme', !isDark ? 'dark' : 'light')
  }

  return (
    <header className="sticky top-0 z-40 border-b bg-background/60 backdrop-blur supports-[backdrop-filter]:bg-background/60">
      <div className="flex h-14 w-full items-center px-4 sm:px-6 lg:px-8">
        <Link to="/" className="font-semibold text-sm md:text-base mr-2 whitespace-nowrap">
          Dashboard IT Asset & Service Management System
        </Link>

        <div className="relative ml-auto hidden sm:block w-60">
          <Search className="absolute left-2 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input placeholder="Search..." className="pl-8" />
        </div>

        <div className="ml-2 flex items-center gap-2">
          <Button variant="ghost" size="icon" onClick={toggleTheme} aria-label="Toggle theme">
            <Sun className="h-5 w-5 rotate-0 scale-100 transition-all dark:-rotate-90 dark:scale-0" />
            <Moon className="absolute h-5 w-5 rotate-90 scale-0 transition-all dark:rotate-0 dark:scale-100" />
          </Button>

          <NotificationBell />

          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" className="h-9 w-9 rounded-full p-0" aria-label="User menu">
                <Avatar className="h-9 w-9">
                  <AvatarFallback className="text-xs">{initials}</AvatarFallback>
                </Avatar>
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-56">
              <DropdownMenuLabel className="space-y-0.5">
                <div className="font-medium leading-none">{user.name || 'User'}</div>
                <div className="text-xs text-muted-foreground truncate">{user.email || '-'}</div>
                {user.role && (
                  <div className="text-[10px] text-muted-foreground uppercase">{user.role}</div>
                )}
              </DropdownMenuLabel>
              <DropdownMenuSeparator />
              <DropdownMenuItem onClick={() => navigate('/')} className="cursor-pointer">
                <User className="mr-2 h-4 w-4" /> Profil (coming soon)
              </DropdownMenuItem>
              <DropdownMenuItem onClick={handleChangePassword} className="cursor-pointer">
                <KeyRound className="mr-2 h-4 w-4" /> Ganti Password
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem
                onClick={handleLogout}
                className="cursor-pointer text-destructive focus:text-destructive"
              >
                <LogOut className="mr-2 h-4 w-4" /> Logout
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>
    </header>
  )
}