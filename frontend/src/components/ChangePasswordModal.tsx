// File: src/components/ChangePasswordModal.tsx
import { useState } from 'react';
import apiClient from '../services/api';
import toast from 'react-hot-toast';

import { Button } from "../components/ui/button";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from "../components/ui/dialog";
import { Input } from "../components/ui/input";
import { Label } from "../components/ui/label";

interface ChangePasswordModalProps {
    isOpen: boolean;
    onClose: () => void;
}

export default function ChangePasswordModal({ isOpen, onClose }: ChangePasswordModalProps) {
    const [oldPassword, setOldPassword] = useState('');
    const [newPassword, setNewPassword] = useState('');
    const [confirmPassword, setConfirmPassword] = useState('');

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        if (newPassword.length < 8) {
      toast.error('Password baru harus minimal 8 karakter.');
      return; // Hentikan proses jika password terlalu pendek
    }
    // ----------------------------------------

    if (newPassword !== confirmPassword) {
      toast.error('Password baru dan konfirmasi tidak cocok.');
      return;
    }

    const promise = apiClient.put('/employees/me/change-password', {
      old_password: oldPassword,
      new_password: newPassword,
    });

    toast.promise(promise, {
      loading: 'Memperbarui password...',
      success: (res) => {
        // Reset form
        setOldPassword('');
        setNewPassword('');
        setConfirmPassword('');
        return res.data.message || 'Password berhasil diperbarui!';
      },
      error: (err) => err.response?.data?.error || 'Gagal memperbarui password.',
    });
    };


    return (
        <Dialog open={isOpen} onOpenChange={onClose}>
            <DialogContent>
                <DialogHeader>
                    <DialogTitle>Ganti Password</DialogTitle>
                    <DialogDescription>Perbarui password Anda secara berkala untuk menjaga keamanan.</DialogDescription>
                </DialogHeader>
                <form onSubmit={handleSubmit} className="space-y-4 py-2">
                    <div className="space-y-2">
                        <Label htmlFor="old-password">Password Lama</Label>
                        <Input id="old-password" type="password" value={oldPassword} onChange={e => setOldPassword(e.target.value)} required />
                        </div>
                        <div className="space-y-2">
                        <Label htmlFor="new-password">Password Baru</Label>
                        <Input id="new-password" type="password" value={newPassword} onChange={e => setNewPassword(e.target.value)} required />
                        </div>
                        <div className="space-y-2">
                        <Label htmlFor="confirm-password">Konfirmasi Password Baru</Label>
                        <Input id="confirm-password" type="password" value={confirmPassword} onChange={e => setConfirmPassword(e.target.value)} required />
                        </div>
                    <Button type="submit" className="w-full">Simpan Perubahan</Button>
                </form>
            </DialogContent>
        </Dialog>
    );
}