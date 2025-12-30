import { useEffect, useState } from 'react'
import apiClient from '@/services/api'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { toast } from 'sonner'

interface Props { isOpen: boolean; onClose: () => void }

export default function BudgetReportModal({ isOpen, onClose }: Props) {
  const [data, setData] = useState<any[]>([])
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    if (isOpen) {
      setLoading(true)
      apiClient.get('/budgets/report')
        .then(res => setData(Array.isArray(res.data?.report) ? res.data.report : []))
        .catch(() => toast.error('Gagal memuat laporan'))
        .finally(() => setLoading(false))
    }
  }, [isOpen])

  return (
    <Dialog open={isOpen} onOpenChange={(o) => { if (!o) onClose() }}>
      <DialogContent className="sm:max-w-4xl">
        <DialogHeader>
          <DialogTitle>Laporan Realisasi Anggaran per Cost Center</DialogTitle>
        </DialogHeader>

        {loading ? (
          <p className="text-center text-muted-foreground">Memuat laporan...</p>
        ) : data.length === 0 ? (
          <p className="text-center text-muted-foreground">Tidak ada data laporan.</p>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Cost Center</TableHead>
                <TableHead>Kategori</TableHead>
                <TableHead>Periode</TableHead>
                <TableHead>Total Terpakai (IDR)</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {data.map((r, i) => (
                <TableRow key={i}>
                  <TableCell>{r.cost_center}</TableCell>
                  <TableCell>{r.category || '-'}</TableCell>
                  <TableCell>{new Date(r.periode).toLocaleDateString('id-ID', { month: 'long', year: 'numeric' })}</TableCell>
                  <TableCell>{r.total_spent.toLocaleString('id-ID', { style: 'currency', currency: 'IDR' })}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </DialogContent>
    </Dialog>
  )
}
