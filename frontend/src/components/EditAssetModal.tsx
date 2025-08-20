// File: src/components/EditAssetModal.tsx
import { useState, useEffect } from 'react';
import apiClient from '../services/api';
import toast from 'react-hot-toast';

import { Button } from "../components/ui/button";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from "../components/ui/dialog";
import { Input } from "../components/ui/input";
import { Label } from "../components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../components/ui/select";
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "../components/ui/alert-dialog";


interface AssetType { id: number; name: string; }

interface EditAssetModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSuccess: () => void;
  assetId: number | null;
}

export default function EditAssetModal({ isOpen, onClose, onSuccess, assetId }: EditAssetModalProps) {
  const [formData, setFormData] = useState({
    name: '', asset_tag: '', status: '', asset_type_id: '',
    purchase_date: '', initial_price: 0,
  });
  const [assetTypes, setAssetTypes] = useState<AssetType[]>([]);
  const [isConfirmOpen, setIsConfirmOpen] = useState(false);
  const [isLoading, setIsLoading] = useState(false);

  useEffect(() => {
    if (isOpen && assetId) {
      setIsLoading(true);
      // Ambil data aset yang akan diedit dan tipe aset untuk dropdown
      Promise.all([
        apiClient.get(`/assets/${assetId}`),
        apiClient.get('/asset-types')
      ]).then(([assetRes, typesRes]) => {
        const asset = assetRes.data;
        // Format tanggal agar sesuai dengan input type="date" (YYYY-MM-DD)
        const formattedDate = asset.purchase_date ? new Date(asset.purchase_date).toISOString().split('T')[0] : '';
        
        setFormData({
          name: asset.name,
          asset_tag: asset.asset_tag,
          status: asset.status,
          asset_type_id: asset.asset_type_id?.toString() || '',
          purchase_date: formattedDate,
          initial_price: asset.initial_price,
        });
        setAssetTypes(typesRes.data);
      }).catch(err => {
        toast.error("Gagal memuat data aset.");
        console.error(err);
      }).finally(() => {
        setIsLoading(false);
      });
    }
  }, [isOpen, assetId]);

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setFormData({ ...formData, [e.target.name]: e.target.value });
  };

  const handleSelectChange = (name: string, value: string) => {
    setFormData({ ...formData, [name]: value });
  };

  const handleSubmit = async () => {
    if (!assetId) return;

    const updatedAsset = {
      ...formData,
      asset_type_id: Number(formData.asset_type_id),
      initial_price: Number(formData.initial_price),
      purchase_date: new Date(formData.purchase_date).toISOString(),
    };

    const promise = apiClient.put(`/assets/${assetId}`, updatedAsset);
    toast.promise(promise, {
      loading: 'Menyimpan perubahan...',
      success: () => {
        onSuccess();
        return 'Aset berhasil diperbarui!';
      },
      error: 'Gagal memperbarui aset.',
    });
  };

  return (
    <>
      <Dialog open={isOpen} onOpenChange={(open) => !open && onClose()}>
        <DialogContent>
          <DialogHeader><DialogTitle>Edit Aset</DialogTitle></DialogHeader>
          {isLoading ? <p>Loading data...</p> : (
            <div className="grid gap-4 py-4">
              {/* Form fields */}
              <Label>Nama</Label> <Input name="name" value={formData.name} onChange={handleChange} />
              <Label>Tag Aset</Label> <Input name="asset_tag" value={formData.asset_tag} onChange={handleChange} />
              <Label>Tipe</Label>
              <Select name="asset_type_id" value={formData.asset_type_id} onValueChange={(v) => handleSelectChange('asset_type_id', v)}>
                <SelectTrigger><SelectValue/></SelectTrigger>
                <SelectContent>
                  {assetTypes.map(type => <SelectItem key={type.id} value={type.id.toString()}>{type.name}</SelectItem>)}
                </SelectContent>
              </Select>
              {/* Add other fields similarly: Status, Purchase Date, Initial Price */}
            </div>
          )}
          <DialogFooter>
            <Button variant="outline" onClick={onClose}>Batal</Button>
            <Button onClick={() => setIsConfirmOpen(true)} disabled={isLoading}>Simpan Perubahan</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
      {/* AlertDialog for confirmation */}
      <AlertDialog open={isConfirmOpen} onOpenChange={setIsConfirmOpen}>
        <AlertDialogContent>
          <AlertDialogHeader><AlertDialogTitle>Konfirmasi</AlertDialogTitle></AlertDialogHeader>
          <AlertDialogDescription>Apakah Anda yakin ingin menyimpan perubahan ini?</AlertDialogDescription>
          <AlertDialogFooter>
            <AlertDialogCancel>Batal</AlertDialogCancel>
            <AlertDialogAction onClick={() => { handleSubmit(); setIsConfirmOpen(false); }}>Yakin</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}