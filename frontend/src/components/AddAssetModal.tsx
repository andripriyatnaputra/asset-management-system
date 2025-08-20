// File: src/components/AddAssetModal.tsx
import { useState, useEffect } from 'react';
import apiClient from '../services/api';
import toast from 'react-hot-toast';

import { Button } from "../components/ui/button";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from "../components/ui/dialog";
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "../components/ui/alert-dialog";
import { Input } from "../components/ui/input";
import { Label } from "../components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../components/ui/select";

interface AssetType { id: number; name: string; }

interface AddAssetModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSuccess: () => void;
}

export default function AddAssetModal({ isOpen, onClose, onSuccess }: AddAssetModalProps) {
  const [assetTag, setAssetTag] = useState('');
  const [name, setName] = useState('');
  const [assetTypeId, setAssetTypeId] = useState('');
  const [status, setStatus] = useState('In Stock');
  const [purchaseDate, setPurchaseDate] = useState('');
  const [initialPrice, setInitialPrice] = useState(0);
  
  const [assetTypes, setAssetTypes] = useState<AssetType[]>([]);
  const [isConfirmOpen, setIsConfirmOpen] = useState(false);

  // Ambil data tipe aset untuk dropdown
  useEffect(() => {
    if (isOpen) {
      apiClient.get('/asset-types').then(response => setAssetTypes(response.data));
    }
  }, [isOpen]);

  const handleSubmit = async () => {
    const newAsset = {
      name,
      asset_tag: assetTag,
      asset_type_id: Number(assetTypeId),
      status,
      purchase_date: new Date(purchaseDate).toISOString(),
      initial_price: Number(initialPrice),
    };

    const promise = apiClient.post('/assets', newAsset);

    toast.promise(promise, {
      loading: 'Menyimpan aset baru...',
      success: () => {
        onSuccess();
        return 'Aset berhasil ditambahkan!';
      },
      error: 'Gagal menambahkan aset.',
    });
  };
  
  const resetForm = () => {
    setName(''); setAssetTag(''); setAssetTypeId('');
    setStatus('In Stock'); setPurchaseDate(''); setInitialPrice(0);
  }

  return (
    <>
      <Dialog open={isOpen} onOpenChange={(open) => { if (!open) { resetForm(); onClose(); } }}>
        <DialogContent className="sm:max-w-[425px]">
          <DialogHeader>
            <DialogTitle>Tambah Aset Baru</DialogTitle>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid grid-cols-4 items-center gap-4">
              <Label htmlFor="name" className="text-right">Nama</Label>
              <Input id="name" value={name} onChange={e => setName(e.target.value)} className="col-span-3" />
            </div>
            <div className="grid grid-cols-4 items-center gap-4">
              <Label htmlFor="tag" className="text-right">Tag Aset</Label>
              <Input id="tag" value={assetTag} onChange={e => setAssetTag(e.target.value)} className="col-span-3" />
            </div>
            <div className="grid grid-cols-4 items-center gap-4">
              <Label htmlFor="type" className="text-right">Tipe</Label>
              <Select onValueChange={setAssetTypeId}>
                  <SelectTrigger className="col-span-3">
                    <SelectValue placeholder="Pilih tipe aset..." />
                  </SelectTrigger>
                  <SelectContent>
                    {assetTypes.map(type => (
                      <SelectItem key={type.id} value={type.id.toString()}>{type.name}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
            </div>
             <div className="grid grid-cols-4 items-center gap-4">
              <Label htmlFor="date" className="text-right">Tgl Beli</Label>
              <Input id="date" type="date" value={purchaseDate} onChange={e => setPurchaseDate(e.target.value)} className="col-span-3" />
            </div>
             <div className="grid grid-cols-4 items-center gap-4">
              <Label htmlFor="price" className="text-right">Harga</Label>
              <Input id="price" type="number" value={initialPrice} onChange={e => setInitialPrice(Number(e.target.value))} className="col-span-3" />
            </div>
          </div>
          <DialogFooter>
            <Button onClick={() => setIsConfirmOpen(true)}>Simpan Aset</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <AlertDialog open={isConfirmOpen} onOpenChange={setIsConfirmOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Konfirmasi</AlertDialogTitle>
            <AlertDialogDescription>
              Apakah Anda yakin ingin menyimpan aset baru ini?
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Batal</AlertDialogCancel>
            <AlertDialogAction onClick={() => { handleSubmit(); setIsConfirmOpen(false); }}>Yakin</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}