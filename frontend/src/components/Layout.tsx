import { Outlet } from 'react-router-dom'
import Header from './Header'
import Sidebar from './Sidebar'
import { Toaster } from './ui/sonner'           // sesuai hasil shadcn add sonner
import { TooltipProvider } from './ui/tooltip'  // ⬅️ tambahkan ini

const wsUrl = import.meta.env.VITE_WS_URL || null

export default function Layout() {
  if (wsUrl) console.log('WS enabled:', wsUrl)

  return (
    <TooltipProvider delayDuration={200}>
      <div className="flex h-screen flex-col">
        <Header />
        <div className="flex flex-1 overflow-hidden">
          <Sidebar />
          <main className="flex-1 overflow-y-auto p-4">
            <Outlet />
          </main>
        </div>
        <Toaster />
      </div>
    </TooltipProvider>
  )
}
