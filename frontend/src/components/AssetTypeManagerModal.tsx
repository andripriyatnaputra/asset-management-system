// File: src/components/AssetTypeManagerModal.tsx
import { useState, useEffect } from 'react';
import apiClient from '../services/api';
import toast from 'react-hot-toast';

import { Button } from "../components/ui/button";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from "../components/ui/dialog";
import { Input } from "../components/ui/input";
import { Table, TableBody, TableCell, TableRow } from "../components/ui/table";

interface AssetType { id: number; name: string; }

interface AssetTypeManagerModalProps {
  isOpen: boolean;
  onClose: () => void;
}

export default function AssetTypeManagerModal({ isOpen, onClose }: AssetTypeManagerModalProps) {
  const [assetTypes, setAssetTypes] = useState<AssetType[]>([]);
  const [newTypeName, setNewTypeName] = useState('');

  const fetchAssetTypes = () => {
    apiClient.get('/asset-types').then(response => {
      setAssetTypes(response.data);
    });
  };

  useEffect(() => {
    if (isOpen) {
      fetchAssetTypes();
    }
  }, [isOpen]);

  const handleAddType = async () => {
    if (!newTypeName.trim()) {
      toast.error('Nama tipe tidak boleh kosong.');
      return;
    }

    const promise = apiClient.post('/asset-types', { name: newTypeName });

    toast.promise(promise, {
      loading: 'Menyimpan tipe baru...',
      success: () => {
        setNewTypeName('');
        fetchAssetTypes(); // Refresh daftar
        return 'Tipe aset berhasil ditambahkan!';
      },
      error: 'Gagal menambahkan tipe aset.',
    });
  };

  return (
    <Dialog open={isOpen} onOpenChange={(open) => { if (!open) onClose(); }}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Manajemen Tipe Aset</DialogTitle>
          <DialogDescription>Tambah atau lihat tipe aset yang sudah ada.</DialogDescription>
        </DialogHeader>
        <div className="py-4">
          <div className="flex w-full max-w-sm items-center space-x-2 mb-4">
            <Input 
              type="text" 
              placeholder="Nama tipe baru..." 
              value={newTypeName}
              onChange={e => setNewTypeName(e.target.value)}
            />
            <Button type="submit" onClick={handleAddType}>Tambah</Button>
          </div>
          <div className="border rounded-md max-h-60 overflow-y-auto">
            <Table>
              <TableBody>
                {assetTypes.map((type) => (
                  <TableRow key={type.id}>
                    <TableCell>{type.name}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}