import { useEffect, useState } from 'react'
import apiClient from '@/services/api'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { toast } from 'sonner'

// ✅ Definisikan tipe data lisensi
interface License {
  id: number
  name: string
  vendor?: string
  license_type?: string
  license_model?: string
  total_seats?: number
  cost?: number
  expiration_date?: string
  compliance_status?: string
}

// ✅ Definisikan tipe props komponen
interface Props {
  isOpen: boolean
  onClose: () => void
  contractId: number | null
}

export default function ContractLicensesModal({ isOpen, onClose, contractId }: Props) {
  const [licenses, setLicenses] = useState<License[]>([])

  useEffect(() => {
    if (isOpen && contractId) {
      apiClient
        .get(`/contracts/${contractId}/licenses`)
        .then(r => {
          const data = Array.isArray(r.data?.licenses) ? r.data.licenses : []
          setLicenses(data)
        })
        .catch(() => {
          toast.error('Gagal memuat data lisensi.')
          setLicenses([])
        })
    }
  }, [isOpen, contractId])

  return (
    <Dialog open={isOpen} onOpenChange={(o) => { if (!o) onClose() }}>
      <DialogContent className="sm:max-w-3xl">
        <DialogHeader>
          <DialogTitle>Daftar Lisensi untuk Kontrak #{contractId}</DialogTitle>
        </DialogHeader>

        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Nama</TableHead>
              <TableHead>Vendor</TableHead>
              <TableHead>Jenis</TableHead>
              <TableHead>Seats</TableHead>
              <TableHead>Kedaluwarsa</TableHead>
              <TableHead>Status</TableHead>
            </TableRow>
          </TableHeader>

          <TableBody>
            {licenses.length === 0 ? (
              <TableRow>
                <TableCell colSpan={6} className="text-center text-muted-foreground">
                  Tidak ada lisensi terkait.
                </TableCell>
              </TableRow>
            ) : (
              licenses.map((l) => (
                <TableRow key={l.id}>
                  <TableCell>{l.name}</TableCell>
                  <TableCell>{l.vendor || '-'}</TableCell>
                  <TableCell>{l.license_type || '-'}</TableCell>
                  <TableCell>{l.total_seats ?? '-'}</TableCell>
                  <TableCell>
                    {l.expiration_date
                      ? new Date(l.expiration_date).toLocaleDateString('id-ID')
                      : '-'}
                  </TableCell>
                  <TableCell>
                    {l.compliance_status ? (
                      <span
                        className={
                          l.compliance_status === 'compliant'
                            ? 'px-2 py-1 text-xs rounded-full bg-green-100 text-green-800'
                            : l.compliance_status === 'non-compliant'
                            ? 'px-2 py-1 text-xs rounded-full bg-red-100 text-red-800'
                            : 'px-2 py-1 text-xs rounded-full bg-gray-100 text-gray-800'
                        }
                      >
                        {l.compliance_status}
                      </span>
                    ) : (
                      '-'
                    )}
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </DialogContent>
    </Dialog>
  )
}
