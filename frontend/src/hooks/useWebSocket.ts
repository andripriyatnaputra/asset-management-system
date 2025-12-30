import { useEffect, useState } from "react"
import { toast } from "sonner"

export interface WSMessage {
  type: string
  action?: string
  data?: any
  timestamp?: string
}

export interface AlertMessage {
  message: string
  severity: "info" | "warning" | "critical"
  timestamp: string
}

export function useWebSocket() {
  const [connected, setConnected] = useState(false)
  const [lastMessage, setLastMessage] = useState<WSMessage | null>(null)
  const [alerts, setAlerts] = useState<AlertMessage[]>([])

  useEffect(() => {
    const token = localStorage.getItem("authToken")
    if (!token) {
      console.warn("⚠️ WebSocket: Token tidak ditemukan.")
      return
    }

    // 🧠 Pastikan URL menuju /api/v1/ws
    const httpBase =
      import.meta.env.VITE_API_BASE || window.location.origin + "/api/v1"
    const wsBase = httpBase.replace(/^http/, "ws")
    const wsUrl = `${wsBase}/ws?token=${token}`

    console.log("🔌 [WS] Connecting to:", wsUrl)

    let ws: WebSocket | null = null
    let reconnectTimer: NodeJS.Timeout | null = null

    const connect = () => {
      ws = new WebSocket(wsUrl)

      ws.onopen = () => {
        console.log("✅ [WS] Connected")
        setConnected(true)
      }

      ws.onmessage = (event) => {
        try {
          const msg: WSMessage = JSON.parse(event.data)
          setLastMessage(msg)

          if (msg.type === "alert") {
            const sev = msg.data?.severity || "info"
            const text = msg.data?.message || "⚠️ Alert baru diterima"
            const alert: AlertMessage = {
              message: text,
              severity:
                sev === "critical"
                  ? "critical"
                  : sev === "warning"
                  ? "warning"
                  : "info",
              timestamp: new Date().toLocaleString("id-ID"),
            }
            setAlerts((p) => [alert, ...p])
            toast(text, {
              className:
                alert.severity === "critical"
                  ? "bg-red-600 text-white"
                  : alert.severity === "warning"
                  ? "bg-yellow-400 text-black"
                  : "bg-blue-600 text-white",
              duration: 6000,
              position: "top-right",
            })
          }
        } catch (err) {
          console.error("❌ [WS] Parse error:", err)
        }
      }

      ws.onclose = () => {
        console.warn("⚠️ [WS] Disconnected, retrying in 5s...")
        setConnected(false)
        if (reconnectTimer) clearTimeout(reconnectTimer)
        reconnectTimer = setTimeout(connect, 5000)
      }

      ws.onerror = (err) => {
        console.error("❌ [WS] Error:", err)
      }
    }

    connect()
    return () => {
      if (reconnectTimer) clearTimeout(reconnectTimer)
      ws?.close()
    }
  }, [])

  return { connected, lastMessage, alerts }
}
