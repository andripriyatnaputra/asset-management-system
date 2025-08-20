// File: src/components/BudgetFormModal.tsx
import { useState, useEffect } from 'react';
import apiClient from '../services/api';
import toast from 'react-hot-toast';
import type { Budget, Department } from '../types';

import { Button } from "../components/ui/button";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from "../components/ui/dialog";
import { Input } from "../components/ui/input";
import { Label } from "../components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../components/ui/select";

interface BudgetFormModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSuccess: () => void;
  budget: Budget | null;
}

const formatDateForInput = (dateStr?: string | null) => {
  if (!dateStr) return '';
  return new Date(dateStr).toISOString().split('T')[0];
};

export default function BudgetFormModal({ isOpen, onClose, onSuccess, budget }: BudgetFormModalProps) {
  const [formData, setFormData] = useState({
    name: '', department_id: '', start_date: '', end_date: '', total_amount: 0,
  });
  const [departments, setDepartments] = useState<Department[]>([]);
  const isEditMode = budget !== null;

  useEffect(() => {
    // Ambil daftar departemen untuk dropdown
    apiClient.get('/departments').then(res => setDepartments(res.data));

    if (isEditMode) {
      setFormData({
        name: budget.name,
        department_id: budget.department_id?.toString() || '',
        start_date: formatDateForInput(budget.start_date),
        end_date: formatDateForInput(budget.end_date),
        total_amount: budget.total_amount,
      });
    } else {
      setFormData({ name: '', department_id: '', start_date: '', end_date: '', total_amount: 0 });
    }
  }, [budget, isOpen]);

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setFormData({ ...formData, [e.target.name]: e.target.value });
  };

  const handleSelectChange = (value: string) => {
    setFormData({ ...formData, department_id: value });
  };

  const handleSubmit = () => {
    if (!formData.name || !formData.start_date || !formData.end_date) {
      toast.error("Nama, Tanggal Mulai, dan Tanggal Selesai wajib diisi.");
      return;
    }

    const payload = {
      name: formData.name,
      department_id: formData.department_id === 'null' || formData.department_id === '' ? null : Number(formData.department_id),
      total_amount: Number(formData.total_amount),
      // Ubah tanggal menjadi format ISO String lengkap yang dimengerti Go
      start_date: new Date(formData.start_date).toISOString(),
      end_date: new Date(formData.end_date).toISOString(),
    };

    const promise = isEditMode
      ? apiClient.put(`/budgets/${budget.id}`, payload) // Kita perlu endpoint PUT nanti
      : apiClient.post('/budgets', payload);

    toast.promise(promise, {
      loading: 'Menyimpan anggaran...',
      success: () => {
        onSuccess();
        return `Anggaran berhasil ${isEditMode ? 'diperbarui' : 'ditambahkan'}!`;
      },
      error: (err) => err.response?.data?.error || `Gagal menyimpan anggaran.`,
    });
  };

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent>
        <DialogHeader><DialogTitle>{isEditMode ? 'Edit' : 'Tambah'} Anggaran</DialogTitle></DialogHeader>
        <div className="grid gap-4 py-4">
          <div><Label>Nama Anggaran</Label><Input name="name" value={formData.name} onChange={handleChange} /></div>
          <div>
            <Label>Departemen (Opsional)</Label>
            <Select value={formData.department_id} onValueChange={handleSelectChange}>
              <SelectTrigger><SelectValue placeholder="Pilih Departemen..." /></SelectTrigger>
              <SelectContent>
                <SelectItem value="null">Umum (Tanpa Departemen)</SelectItem>
                {departments.map(dept => <SelectItem key={dept.id} value={dept.id.toString()}>{dept.name}</SelectItem>)}
              </SelectContent>
            </Select>
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div><Label>Tanggal Mulai</Label><Input name="start_date" type="date" value={formData.start_date} onChange={handleChange} /></div>
            <div><Label>Tanggal Selesai</Label><Input name="end_date" type="date" value={formData.end_date} onChange={handleChange} /></div>
          </div>
          <div><Label>Jumlah Total (IDR)</Label><Input name="total_amount" type="number" value={formData.total_amount} onChange={handleChange} /></div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={onClose}>Batal</Button>
          <Button onClick={handleSubmit}>Simpan</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}