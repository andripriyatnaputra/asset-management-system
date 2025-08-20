// File: src/components/LicenseFormModal.tsx
import { useState, useEffect } from 'react';
import apiClient from '../services/api';
import toast from 'react-hot-toast';
import type { SoftwareLicense } from '../types';

import { Button } from "../components/ui/button";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from "../components/ui/dialog";
import { Input } from "../components/ui/input";
import { Label } from "../components/ui/label";

interface LicenseFormModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSuccess: () => void;
  license: SoftwareLicense | null;
}

// Helper to format date for input type="date"
const formatDateForInput = (dateStr?: string | null) => {
  if (!dateStr) return '';
  return new Date(dateStr).toISOString().split('T')[0];
};

export default function LicenseFormModal({ isOpen, onClose, onSuccess, license }: LicenseFormModalProps) {
  const [formData, setFormData] = useState({
    name: '', license_key: '', total_seats: 1,
    purchase_date: '', expiration_date: '', cost: 0,
  });
  const isEditMode = license !== null;
  

  useEffect(() => {
    if (isEditMode) {
      setFormData({
        name: license.name,
        license_key: license.license_key || '',
        total_seats: license.total_seats,
        purchase_date: formatDateForInput(license.purchase_date),
        expiration_date: formatDateForInput(license.expiration_date),
        cost: license.cost || 0,
      });
    } else {
      setFormData({ name: '', license_key: '', total_seats: 1, purchase_date: '', expiration_date: '', cost: 0 });
    }
  }, [license, isOpen]);

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setFormData({ ...formData, [e.target.name]: e.target.value });
  };

  const handleSubmit = () => {

    if (!isEditMode && !formData.name) {
      toast.error("Nama Software tidak boleh kosong.");
      return;
    }
    
    const payload = {
      ...formData,
      total_seats: Number(formData.total_seats),
      cost: Number(formData.cost),
      purchase_date: formData.purchase_date ? new Date(formData.purchase_date).toISOString() : null,
      expiration_date: formData.expiration_date ? new Date(formData.expiration_date).toISOString() : null,
    };

    const promise = isEditMode
      ? apiClient.put(`/licenses/${license.id}`, payload)
      : apiClient.post('/licenses', payload);

    toast.promise(promise, {
      loading: 'Menyimpan data lisensi...',
      success: () => {
        onSuccess();
        return `Lisensi berhasil ${isEditMode ? 'diperbarui' : 'ditambahkan'}!`;
      },
      error: (err) => err.response?.data?.error || `Gagal menyimpan data.`,
    });
  };

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader><DialogTitle>{isEditMode ? 'Edit' : 'Tambah'} Lisensi Software</DialogTitle></DialogHeader>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4 py-4">
          <div><Label>Nama Software</Label><Input name="name" value={formData.name} onChange={handleChange} /></div>
          <div><Label>Kunci Lisensi</Label><Input name="license_key" value={formData.license_key} onChange={handleChange} /></div>
          <div><Label>Jumlah Pengguna</Label><Input name="total_seats" type="number" value={formData.total_seats} onChange={handleChange} /></div>
          <div><Label>Biaya (IDR)</Label><Input name="cost" type="number" value={formData.cost} onChange={handleChange} /></div>
          <div><Label>Tanggal Pembelian</Label><Input name="purchase_date" type="date" value={formData.purchase_date} onChange={handleChange} /></div>
          <div><Label>Tanggal Kedaluwarsa</Label><Input name="expiration_date" type="date" value={formData.expiration_date} onChange={handleChange} /></div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={onClose}>Batal</Button>
          <Button onClick={handleSubmit}>Simpan</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}