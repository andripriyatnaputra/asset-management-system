// File: src/components/CreateAuditModal.tsx
import { useState } from 'react';
import apiClient from '../services/api';
import { toast } from 'sonner'

import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter, DialogDescription } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

interface CreateAuditModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSuccess: () => void;
}

export default function CreateAuditModal({ isOpen, onClose, onSuccess }: CreateAuditModalProps) {
  const [sessionName, setSessionName] = useState('');

  const handleSubmit = () => {
    if (!sessionName.trim()) {
      toast.error("Nama sesi audit tidak boleh kosong.");
      return;
    }
    const promise = apiClient.post('/audits', { name: sessionName });
    toast.promise(promise, {
      loading: 'Memulai sesi audit baru...',
      success: () => {
        onSuccess();
        return 'Sesi audit baru berhasil dibuat!';
      },
      error: (err) => err.response?.data?.error || 'Gagal memulai sesi audit.',
    });
  };

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Mulai Sesi Audit Baru</DialogTitle>
          <DialogDescription>
            Sistem akan membuat snapshot dari semua aset aktif saat ini untuk diaudit.
          </DialogDescription>
        </DialogHeader>
        <div className="py-4">
          <Label htmlFor="name">Nama Sesi</Label>
          <Input 
            id="name" 
            value={sessionName}
            onChange={e => setSessionName(e.target.value)}
            placeholder="Contoh: Audit Aset Lantai 5 - Q4 2025"
          />
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={onClose}>Batal</Button>
          <Button onClick={handleSubmit}>Mulai Sesi</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}