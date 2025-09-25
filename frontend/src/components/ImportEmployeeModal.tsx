// File: src/components/ImportEmployeeModal.tsx
import { useState } from 'react';
import apiClient from '@/services/api';
import toast from 'react-hot-toast';

import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

interface ImportEmployeeModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSuccess: () => void;
}

export default function ImportEmployeeModal({ isOpen, onClose, onSuccess }: ImportEmployeeModalProps) {
  const [selectedFile, setSelectedFile] = useState<File | null>(null);

  const handleFileChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    if (event.target.files) {
      setSelectedFile(event.target.files[0]);
    }
  };

  const handleUpload = () => {
    if (!selectedFile) {
      toast.error("Silakan pilih file CSV terlebih dahulu.");
      return;
    }

    const formData = new FormData();
    formData.append("file", selectedFile);

    const promise = apiClient.post('/employees/import', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    });

    toast.promise(promise, {
      loading: 'Mengimpor data karyawan...',
      success: (res) => {
        onSuccess();
        return res.data.message || 'Proses impor selesai!';
      },
      error: 'Gagal mengimpor data.',
    });
  };

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Impor Karyawan dari CSV</DialogTitle>
          <DialogDescription>
            Pilih file CSV dengan kolom: No, Nama, NIK, Email, Departemen.
          </DialogDescription>
        </DialogHeader>
        <div className="py-4">
          <Label htmlFor="csv-file">File CSV</Label>
          <Input id="csv-file" type="file" accept=".csv" onChange={handleFileChange} />
        </div>
        <Button onClick={handleUpload} className="w-full">Unggah dan Impor</Button>
      </DialogContent>
    </Dialog>
  );
}