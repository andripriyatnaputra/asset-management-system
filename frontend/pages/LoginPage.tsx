import { useState } from "react"
import { Link } from "react-router-dom"
import { Package, ShieldCheck, Eye, EyeOff } from "lucide-react"
import { toast } from "sonner"
import apiClient from "@/services/api"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Button } from "@/components/ui/button"
import { useAuthContext } from "@/context/AuthContext"

export default function LoginPage() {
  const [email, setEmail] = useState("")
  const [password, setPassword] = useState("")
  const [showPassword, setShowPassword] = useState(false)
  const [isLoading, setIsLoading] = useState(false) // ✅ aktifkan lagi untuk tombol
  const { login } = useAuthContext()

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault()
    setIsLoading(true)
    try {
      const res = await apiClient.post("/auth/login", { email, password })
      const { token, refresh_token } = res.data

      if (!token) {
        toast.error("Login gagal: token tidak diterima dari server.")
        return
      }

      login(token, refresh_token)
      toast.success("Login berhasil")

      // 🔁 Redirect setelah token tersimpan
      setTimeout(() => {
        window.location.replace("/")
      }, 300)
    } catch (err: any) {
      toast.error(err?.response?.data?.error || "Login gagal")
    } finally {
      setIsLoading(false)
    }
  }


  return (
    <div className="min-h-screen grid grid-cols-1 lg:grid-cols-2">
      {/* KIRI — HERO / BRAND */}
      <section
        className="
          relative hidden lg:flex items-center justify-center overflow-hidden
          bg-[radial-gradient(1200px_600px_at_-10%_-10%,hsl(var(--primary)/0.08),transparent_60%)]
          dark:bg-[radial-gradient(1200px_600px_at_-10%_-10%,hsl(var(--primary)/0.12),transparent_60%)]
        "
      >
        <div className="pointer-events-none absolute inset-0 [mask-image:radial-gradient(circle_at_center,black,transparent_70%)]">
          <div className="absolute -top-24 -left-24 h-72 w-72 rounded-full bg-primary/15 blur-3xl" />
          <div className="absolute -bottom-28 -right-20 h-80 w-80 rounded-full bg-primary/10 blur-3xl" />
        </div>

        <div className="relative z-10 mx-auto max-w-xl px-12 text-center">
          <span className="mx-auto mb-7 inline-flex h-16 w-16 items-center justify-center rounded-2xl bg-primary/10 ring-1 ring-primary/20">
            <Package className="h-8 w-8 text-primary" />
          </span>

          <h2 className="text-4xl font-extrabold leading-tight tracking-tight">
            IT Asset & Service
            <span className="block">Management System</span>
          </h2>

          <p className="mt-3 text-base text-muted-foreground">
            Solusi terintegrasi untuk pengelolaan aset & layanan IT—aman, cepat,
            dan mudah dioperasikan.
          </p>

          <ul className="mt-8 grid grid-cols-3 gap-6 text-sm text-muted-foreground">
            <li className="flex flex-col items-center gap-2">
              <ShieldCheck className="h-4 w-4" />
              <span className="text-center leading-tight">
                Keamanan<br />Data
              </span>
            </li>
            <li className="flex flex-col items-center gap-2">
              <ShieldCheck className="h-4 w-4" />
              <span className="text-center leading-tight">
                Real-time<br />Insight
              </span>
            </li>
            <li className="flex flex-col items-center gap-2">
              <ShieldCheck className="h-4 w-4" />
              <span className="text-center leading-tight">
                Multi-Role<br />Access
              </span>
            </li>
          </ul>
        </div>
      </section>

      {/* KANAN — FORM LOGIN */}
      <section className="flex items-center justify-center px-6 py-12">
        <div className="w-full max-w-sm">
          <div className="rounded-2xl border bg-card/50 p-6 shadow-sm">
            <div className="mb-5 text-center">
              <h1 className="text-2xl font-bold tracking-tight">Masuk</h1>
              <p className="mt-1 text-sm text-muted-foreground">
                Masukkan email dan password untuk mengakses dashboard
              </p>
            </div>

            <form onSubmit={handleLogin} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="email">Email</Label>
                <Input
                  id="email"
                  type="email"
                  placeholder="admin@example.com"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  required
                  disabled={isLoading}
                  autoComplete="username"
                />
              </div>

              <div className="space-y-2">
                <div className="flex items-center">
                  <Label htmlFor="password">Password</Label>
                  <Link
                    to="/forgot-password"
                    className="ml-auto inline-block text-xs underline"
                  >
                    Lupa Password?
                  </Link>
                </div>

                <div className="relative">
                  <Input
                    id="password"
                    type={showPassword ? "text" : "password"}
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    required
                    disabled={isLoading}
                    autoComplete="current-password"
                  />
                  <button
                    type="button"
                    onClick={() => setShowPassword((v) => !v)}
                    className="absolute inset-y-0 right-2 flex items-center rounded-md px-2 text-muted-foreground hover:text-foreground"
                    tabIndex={-1}
                  >
                    {showPassword ? (
                      <EyeOff className="h-5 w-5" />
                    ) : (
                      <Eye className="h-5 w-5" />
                    )}
                  </button>
                </div>
              </div>

              <Button type="submit" className="w-full" disabled={isLoading}>
                {isLoading ? "Memverifikasi…" : "Login"}
              </Button>
            </form>
          </div>

          <p className="mt-6 text-center text-xs text-muted-foreground">
            © {new Date().getFullYear()} IT-ASMS. All rights reserved.
          </p>
        </div>
      </section>
    </div>
  )
}
