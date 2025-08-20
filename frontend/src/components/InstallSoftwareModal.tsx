// File: src/components/InstallSoftwareModal.tsx
import { useState, useEffect } from 'react';
import apiClient from '../services/api';
import toast from 'react-hot-toast';
import type { SoftwareLicense } from '../types';

import { Button } from "../components/ui/button";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from "..//components/ui/dialog";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "..//components/ui/select";

interface InstallSoftwareModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSuccess: () => void;
  assetId: number | null;
}

export default function InstallSoftwareModal({ isOpen, onClose, onSuccess, assetId }: InstallSoftwareModalProps) {
  const [licenses, setLicenses] = useState<SoftwareLicense[]>([]);
  const [selectedLicenseId, setSelectedLicenseId] = useState('');
  const [isLoading, setIsLoading] = useState(false);

  useEffect(() => {
    if (isOpen) {
      setIsLoading(true);
      // Ambil daftar semua lisensi yang tersedia
      apiClient.get('/licenses')
        .then(res => setLicenses(res.data))
        .catch(() => toast.error("Gagal memuat daftar lisensi."))
        .finally(() => setIsLoading(false));
    }
  }, [isOpen]);

  const handleSubmit = () => {
    if (!assetId || !selectedLicenseId) {
      toast.error("Silakan pilih lisensi terlebih dahulu.");
      return;
    }

    const promise = apiClient.post(`/assets/${assetId}/installations`, {
      license_id: Number(selectedLicenseId),
    });

    toast.promise(promise, {
      loading: 'Menginstal software...',
      success: () => {
        onSuccess();
        return 'Software berhasil diinstal!';
      },
      error: (err) => err.response?.data?.error || 'Gagal menginstal software.',
    });
  };

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Install Software Baru</DialogTitle>
        </DialogHeader>
        <div className="py-4">
          <p className="mb-2 text-sm text-muted-foreground">Pilih lisensi software yang akan diinstal pada aset ini.</p>
          <Select onValueChange={setSelectedLicenseId}>
            <SelectTrigger>
              <SelectValue placeholder={isLoading ? "Memuat lisensi..." : "Pilih lisensi..."} />
            </SelectTrigger>
            <SelectContent>
              {!isLoading && licenses && licenses.map(license => (
                <SelectItem key={license.id} value={license.id.toString()}>
                  {license.name} ({license.total_seats} seats)
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={onClose}>Batal</Button>
          <Button onClick={handleSubmit}>Install</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}