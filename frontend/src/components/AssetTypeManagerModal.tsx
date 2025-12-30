import { useState, useEffect } from 'react'
import apiClient from '../services/api'
import { toast } from 'sonner'

import { Button } from './ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from './ui/dialog'
import { Input } from './ui/input'
import { Table, TableBody, TableCell, TableRow } from './ui/table'

interface AssetType { id: number; name: string }

interface AssetTypeManagerModalProps {
  isOpen: boolean
  onClose: () => void
}

export default function AssetTypeManagerModal({ isOpen, onClose }: AssetTypeManagerModalProps) {
  const [assetTypes, setAssetTypes] = useState<AssetType[]>([])
  const [newTypeName, setNewTypeName] = useState('')

  const fetchAssetTypes = async () => {
    try {
      const res = await apiClient.get('/asset-types')
      const list = Array.isArray(res.data) ? res.data : res.data?.data
      setAssetTypes(Array.isArray(list) ? list : [])
    } catch {
      toast.error('Gagal memuat tipe aset.')
    }
  }

  useEffect(() => { if (isOpen) fetchAssetTypes() }, [isOpen])

  const handleAddType = async () => {
    if (!newTypeName.trim()) {
      toast.error('Nama tipe tidak boleh kosong.')
      return
    }
    const promise = apiClient.post('/asset-types', { name: newTypeName })
    toast.promise(promise, {
      loading: 'Menyimpan tipe baru...',
      success: () => {
        setNewTypeName('')
        fetchAssetTypes()
        return 'Tipe aset berhasil ditambahkan!'
      },
      error: 'Gagal menambahkan tipe aset.',
    })
  }

  return (
    <Dialog open={isOpen} onOpenChange={(open) => { if (!open) onClose() }}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Manajemen Tipe Aset</DialogTitle>
          <DialogDescription>Tambah atau lihat tipe aset yang sudah ada.</DialogDescription>
        </DialogHeader>
        <div className="py-4">
          <div className="mb-4 flex w-full max-w-sm items-center space-x-2">
            <Input
              type="text"
              placeholder="Nama tipe baru..."
              value={newTypeName}
              onChange={e => setNewTypeName(e.target.value)}
            />
            <Button type="button" onClick={handleAddType}>Tambah</Button>
          </div>
          <div className="max-h-60 overflow-y-auto rounded-md border">
            <Table>
              <TableBody>
                {assetTypes.map((type) => (
                  <TableRow key={type.id}>
                    <TableCell>{type.name}</TableCell>
                  </TableRow>
                ))}
                {assetTypes.length === 0 && (
                  <TableRow>
                    <TableCell className="text-center text-muted-foreground">Belum ada tipe aset.</TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}
