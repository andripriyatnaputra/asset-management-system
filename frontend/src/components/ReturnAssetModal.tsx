// File: src/components/ReturnAssetModal.tsx
import { useState, useEffect } from 'react';
import apiClient from '../services/api';
import toast from 'react-hot-toast';

import { Button } from "../components/ui/button";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from "../components/ui/dialog";
import { Textarea } from '../components/ui/textarea';
import { Label } from '../components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../components/ui/select";

// Interface props yang sudah diperbaiki dan disederhanakan
interface ReturnAssetModalProps {
  isOpen: boolean;
  onClose: () => void;
  assetId: number | null;
  onSuccess: () => void;
}

export default function ReturnAssetModal({ isOpen, onClose, assetId, onSuccess }: ReturnAssetModalProps) {
  const [nextStatus, setNextStatus] = useState('In Stock');
  const [notes, setNotes] = useState('');

  // Reset state saat modal dibuka atau ditutup
  useEffect(() => {
    if (!isOpen) {
        setNextStatus('In Stock');
        setNotes('');
    }
  }, [isOpen]);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!assetId) return;

    const promise = apiClient.post(`/assets/${assetId}/return`, {
      next_status: nextStatus,
      notes: notes,
    });
    
    toast.promise(promise, {
        loading: 'Memproses pengembalian...',
        success: () => {
            onSuccess();
            return 'Aset berhasil dikembalikan!';
        },
        error: (err) => err.response?.data?.error || 'Gagal melakukan pengembalian.'
    })
  };

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Return Asset</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4 py-2">
            <div>
                <Label htmlFor="next-status">Status Aset Selanjutnya</Label>
                <Select value={nextStatus} onValueChange={setNextStatus}>
                    <SelectTrigger id="next-status">
                        <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                        <SelectItem value="In Stock">In Stock</SelectItem>
                        <SelectItem value="In Repair">In Repair</SelectItem>
                        <SelectItem value="Broken">Broken</SelectItem>
                    </SelectContent>
                </Select>
            </div>
            <div>
                <Label htmlFor="return-notes">Catatan Pengembalian</Label>
                <Textarea
                    id="return-notes"
                    value={notes}
                    onChange={(e) => setNotes(e.target.value)}
                    placeholder="Contoh: Dikembalikan oleh Budi, kondisi baik."
                />
            </div>
            <DialogFooter>
                <Button type="button" variant="outline" onClick={onClose}>Batal</Button>
                <Button type="submit">Konfirmasi Pengembalian</Button>
            </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}