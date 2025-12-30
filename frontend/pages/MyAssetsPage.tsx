// File: src/pages/MyAssetsPage.tsx
import { useEffect, useState } from 'react';
import apiClient from '@/services/api';
import { toast } from 'sonner';
import type { Asset } from '@/types'; // Kita bisa gunakan tipe Asset yang sudah ada

import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "../src/components/ui/table";
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';

export default function MyAssetsPage() {
  const [myAssets, setMyAssets] = useState<Asset[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    setIsLoading(true);
    apiClient.get('/employees/me/assets')
      .then(res => {
        setMyAssets(res.data);
      })
      .catch(() => toast.error('Gagal memuat data aset Anda.'))
      .finally(() => setIsLoading(false));
  }, []);

  return (
    <div className="space-y-6 p-4 md:p-6">
      <h1 className="text-2xl font-semibold">Aset Saya</h1>
      <Card>
        <CardHeader>
          <CardTitle>Daftar Aset yang Ditugaskan</CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Tag Aset</TableHead>
                <TableHead>Nama Aset</TableHead>
                <TableHead>Tipe</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {isLoading ? (
                <TableRow><TableCell colSpan={3} className="text-center h-24">Memuat data…</TableCell></TableRow>
              ) : myAssets.length > 0 ? (
                myAssets.map((a) => (
                  <TableRow key={a.id}>
                    <TableCell className="font-mono">{a.asset_tag}</TableCell>
                    <TableCell>{a.name}</TableCell>
                    <TableCell>{a.asset_type_name || '-'}</TableCell>
                  </TableRow>
                ))
              ) : (
                <TableRow><TableCell colSpan={3} className="text-center h-24">Anda belum memiliki aset aktif.</TableCell></TableRow>
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  )
}