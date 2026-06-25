import { type ClassValue, clsx } from "clsx"
import { twMerge } from "tailwind-merge"
import apiClient from "@/services/api"
import { toast } from "sonner"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export async function downloadFile(endpoint: string, filename: string) {
  try {
    const res = await apiClient.get(endpoint, { responseType: "blob" })
    const url = URL.createObjectURL(new Blob([res.data]))
    const a = document.createElement("a")
    a.href = url
    a.setAttribute("download", filename)
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
  } catch {
    toast.error(`Gagal mengunduh ${filename}`)
  }
}