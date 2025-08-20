// File: src/components/AssignAssetModal.tsx
import { useState, useEffect } from 'react';
import apiClient from '../services/api';
import toast from 'react-hot-toast';
import type { Employee } from '../types';

import { Button } from "../components/ui/button";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from "../components/ui/dialog";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../components/ui/select";
import { Textarea } from '../components/ui/textarea';
import { Label } from '../components/ui/label';

// Pastikan interface ini memiliki 'onClose'
interface AssignAssetModalProps {
  isOpen: boolean;
  onClose: () => void;
  assetId: number | null;
  onSuccess: () => void;
}

export default function AssignAssetModal({ isOpen, onClose, assetId, onSuccess }: AssignAssetModalProps) {
  const [employees, setEmployees] = useState<Employee[] | null>(null);
  const [selectedEmployee, setSelectedEmployee] = useState('');
  const [notes, setNotes] = useState('');
  const [isLoading, setIsLoading] = useState(false);

  useEffect(() => {
    if (isOpen) {
      setIsLoading(true);
      apiClient.get('/employees?limit=500')
        .then(response => {
          setEmployees(response.data.data);
        })
        .catch(() => toast.error('Gagal memuat daftar karyawan.'))
        .finally(() => setIsLoading(false));
    } else {
      setSelectedEmployee('');
      setNotes('');
    }
  }, [isOpen]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedEmployee || !assetId) {
      toast.error('Silakan pilih karyawan.');
      return;
    }
    
    const promise = apiClient.post(`/assets/${assetId}/assign`, {
      employee_nik: selectedEmployee,
      notes: notes,
    });

    toast.promise(promise, {
        loading: 'Melakukan assignment...',
        success: () => {
            onSuccess(); // Panggil onSuccess dari props
            return 'Aset berhasil di-assign!';
        },
        error: (err) => err.response?.data?.error || 'Gagal melakukan assignment.'
    });
  };

  return (
    // Gunakan <Dialog> dari Shadcn, yang menggunakan onOpenChange
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Assign Asset</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4 py-2">
            <div>
                <Label htmlFor="employee-select">Pilih Karyawan</Label>
                <Select onValueChange={setSelectedEmployee} value={selectedEmployee}>
                    <SelectTrigger id="employee-select">
                    <SelectValue placeholder={isLoading ? "Memuat..." : "Pilih Karyawan..."} />
                    </SelectTrigger>
                    <SelectContent>
                    {employees && employees.map((emp) => (
                        <SelectItem key={emp.id} value={emp.employee_nik}>
                        {emp.name} ({emp.employee_nik})
                        </SelectItem>
                    ))}
                    </SelectContent>
                </Select>
            </div>
            <div>
                <Label htmlFor="notes">Catatan (Opsional)</Label>
                <Textarea
                    id="notes"
                    value={notes}
                    onChange={(e) => setNotes(e.target.value)}
                    placeholder="Contoh: Diberikan untuk proyek X..."
                />
            </div>
            <DialogFooter>
                <Button type="button" variant="outline" onClick={onClose}>Batal</Button>
                <Button type="submit">Assign Aset</Button>
            </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}