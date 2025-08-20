// File: src/components/CreateTicketModal.tsx
import { useState, useEffect } from 'react';
import apiClient from '../services/api';
import toast from 'react-hot-toast';
import type { Asset } from '../types';

import { Button } from "../components/ui/button";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from "../components/ui/dialog";
import { Input } from "../components/ui/input";
import { Label } from "../components/ui/label";
import { Textarea } from "../components/ui/textarea";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../components/ui/select";

interface CreateTicketModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSuccess: () => void;
}

export default function CreateTicketModal({ isOpen, onClose, onSuccess }: CreateTicketModalProps) {
  const [subject, setSubject] = useState('');
  const [description, setDescription] = useState('');
  const [priority, setPriority] = useState('Medium');
  const [relatedAssetId, setRelatedAssetId] = useState<string | undefined>(undefined);
  const [assignedAssets, setAssignedAssets] = useState<Asset[]>([]);

  useEffect(() => {
    if (isOpen) {
      // Ambil daftar aset milik pengguna
      apiClient.get('/employees/me/assets')
        .then(res => setAssignedAssets(res.data))
        .catch(() => toast.error("Gagal memuat daftar aset Anda."));
    } else {
      // Reset form saat modal ditutup
      setSubject(''); setDescription(''); setPriority('Medium'); setRelatedAssetId(undefined);
    }
  }, [isOpen]);

  const handleSubmit = () => {
    const promise = apiClient.post('/tickets', {
      subject,
      description,
      priority,
      related_asset_id: relatedAssetId ? Number(relatedAssetId) : null,
    });

    toast.promise(promise, {
      loading: 'Mengirim tiket...',
      success: () => {
        onSuccess();
        return 'Tiket berhasil dibuat!';
      },
      error: (err) => err.response?.data?.error || 'Gagal membuat tiket.',
    });
  };

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent>
        <DialogHeader><DialogTitle>Buat Tiket Bantuan Baru</DialogTitle></DialogHeader>
        <div className="py-4 space-y-4">
          <div><Label>Subjek</Label><Input value={subject} onChange={e => setSubject(e.target.value)} placeholder="Contoh: Laptop tidak bisa menyala" required /></div>
          <div><Label>Deskripsi Masalah</Label><Textarea value={description} onChange={e => setDescription(e.target.value)} placeholder="Jelaskan masalah Anda secara detail..." /></div>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <Label>Prioritas</Label>
              <Select value={priority} onValueChange={setPriority}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="Low">Low</SelectItem>
                  <SelectItem value="Medium">Medium</SelectItem>
                  <SelectItem value="High">High</SelectItem>
                  <SelectItem value="Critical">Critical</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div>
              <Label>Aset Terkait (Opsional)</Label>
              <Select value={relatedAssetId} onValueChange={setRelatedAssetId}>
                <SelectTrigger><SelectValue placeholder="Pilih aset..." /></SelectTrigger>
                <SelectContent>
                  {assignedAssets && assignedAssets.map(asset => (
                    <SelectItem key={asset.id} value={asset.id.toString()}>
                      {/* Tampilkan tipe aset di sini */}
                      {asset.name} ({asset.asset_type_name}) - {asset.asset_tag}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={onClose}>Batal</Button>
          <Button onClick={handleSubmit}>Kirim Tiket</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}