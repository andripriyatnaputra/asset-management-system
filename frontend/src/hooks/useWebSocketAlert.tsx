import { useEffect, useState } from "react"
import { toast } from "sonner"

export interface WSMessage {
  type: string
  action?: string
  data?: any
  timestamp?: string
}

export function useWebSocket() {
  const [connected, setConnected] = useState(false)
  const [lastMessage, setLastMessage] = useState<WSMessage | null>(null)

  useEffect(() => {
    const token = localStorage.getItem("authToken")
    if (!token) {
      console.warn("❌ WebSocket: missing token")
      return
    }

    // 🧠 pastikan base URL benar
    const base =
      import.meta.env.VITE_API_BASE?.replace(/^http/, "ws") ||
      window.location.origin.replace(/^http/, "ws")

    const wsUrl = `${base}/ws?token=${token}`
    console.log("🔌 WS connecting to:", wsUrl)

    const ws = new WebSocket(wsUrl)

    ws.onopen = () => {
      setConnected(true)
      console.log("✅ WS connected")
    }

    ws.onmessage = (event) => {
      try {
        const msg: WSMessage = JSON.parse(event.data)
        setLastMessage(msg)

        // tampilkan alert realtime
        if (msg.type === "alert") {
          const sev = msg.data?.severity || "info"
          toast(msg.data?.message || "🔔 Notifikasi", {
            className:
              sev === "critical"
                ? "bg-red-600 text-white"
                : sev === "warning"
                ? "bg-yellow-500 text-black"
                : "bg-blue-600 text-white",
          })
        }
      } catch (err) {
        console.error("WS message parse error:", err)
      }
    }

    ws.onclose = () => {
      console.warn("⚠️ WS closed, retry in 5s...")
      setConnected(false)
      setTimeout(() => window.location.reload(), 5000)
    }

    ws.onerror = (err) => {
      console.error("❌ WS error:", err)
    }

    return () => ws.close()
  }, [])

  return { connected, lastMessage }
}
