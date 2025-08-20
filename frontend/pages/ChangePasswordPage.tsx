// File: src/pages/ChangePasswordPage.tsx
import { useState } from 'react';
import apiClient from '../src/services/api';
import toast from 'react-hot-toast';

import { Button } from "../src/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../src/components/ui/card";
import { Input } from "../src/components/ui/input";
import { Label } from "../src/components/ui/label";

export default function ChangePasswordPage() {
  const [oldPassword, setOldPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    
    // --- VALIDASI BARU DITAMBAHKAN DI SINI ---
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
    <div className="container mx-auto py-8">
      <Card className="max-w-lg mx-auto">
        <CardHeader>
          <CardTitle>Ganti Password</CardTitle>
          <CardDescription>Perbarui password Anda secara berkala untuk menjaga keamanan akun.</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
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
        </CardContent>
      </Card>
    </div>
  );
}