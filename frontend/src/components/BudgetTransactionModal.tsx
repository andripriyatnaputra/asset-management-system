import { useEffect, useState } from "react"
import apiClient from "@/services/api"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { toast } from "sonner"

interface Props {
  isOpen: boolean
  onClose: () => void
  budgetId: number | null
}

export default function BudgetTransactionsModal({ isOpen, onClose, budgetId }: Props) {
  const [data, setData] = useState<any[]>([])
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    let active = true
    if (isOpen && budgetId) {
      setLoading(true)
      apiClient
        .get(`/budgets/${budgetId}/transactions`)
        .then((res) => {
          if (!active) return
          const txs = Array.isArray(res.data?.transactions)
            ? res.data.transactions
            : []
          setData(txs)
        })
        .catch((err) => {
          console.error("Load budget transactions error:", err)
          toast.error(
            err?.response?.data?.error || "Gagal memuat transaksi anggaran."
          )
        })
        .finally(() => active && setLoading(false))
    }
    return () => {
      active = false
    }
  }, [isOpen, budgetId])

  const totalAmount = data.reduce(
    (sum, t) => sum + (Number(t.amount) || 0),
    0
  )

  return (
    <Dialog open={isOpen} onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="sm:max-w-5xl">
        <DialogHeader>
          <DialogTitle>Audit Trail Anggaran #{budgetId}</DialogTitle>
        </DialogHeader>

        {loading ? (
          <p className="text-center text-muted-foreground py-6 animate-pulse">
            Memuat transaksi...
          </p>
        ) : data.length === 0 ? (
          <p className="text-center text-muted-foreground py-6">
            Tidak ada transaksi untuk anggaran ini.
          </p>
        ) : (
          <>
            {/* 🔹 Ringkasan */}
            <div className="text-sm text-right text-muted-foreground mb-2">
              Total transaksi: {data.length} | Total nilai:{" "}
              <span className="font-medium text-foreground">
                {totalAmount.toLocaleString("id-ID", {
                  style: "currency",
                  currency: "IDR",
                })}
              </span>
            </div>

            {/* 🔹 Tabel */}
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Tanggal</TableHead>
                  <TableHead>Entitas</TableHead>
                  <TableHead>Nama</TableHead>
                  <TableHead>Nilai (IDR)</TableHead>
                  <TableHead>Cost Center</TableHead>
                  <TableHead>Catatan</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {data.map((t) => (
                  <TableRow key={t.id}>
                    <TableCell>
                      {new Date(t.transaction_date).toLocaleString("id-ID")}
                    </TableCell>
                    <TableCell>
                      {t.contract_id
                        ? "Kontrak"
                        : t.license_id
                        ? "Lisensi"
                        : t.asset_id
                        ? "Aset"
                        : "-"}
                    </TableCell>
                    <TableCell>
                      {t.contract_number ||
                        t.license_name ||
                        t.asset_name ||
                        "-"}
                    </TableCell>
                    <TableCell>
                      {t.amount.toLocaleString("id-ID", {
                        style: "currency",
                        currency: "IDR",
                      })}
                    </TableCell>
                    <TableCell>{t.cost_center || "-"}</TableCell>
                    <TableCell>{t.notes || "-"}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </>
        )}
      </DialogContent>
    </Dialog>
  )
}
