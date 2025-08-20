// File: src/pages/MyAssetsPage.tsx
import { useEffect, useState } from 'react';
import apiClient from '../src/services/api';
import toast from 'react-hot-toast';
import type { Asset } from '../src//types'; // Kita bisa gunakan tipe Asset yang sudah ada

import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "../src/components/ui/table";

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
    <div className="container mx-auto py-8">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-3xl font-bold">Aset Saya</h1>
      </div>

      <div className="bg-white p-4 border rounded-lg">
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
              <TableRow><TableCell colSpan={3} className="text-center h-24">Loading...</TableCell></TableRow>
            ) : myAssets && myAssets.length > 0 ? (
              myAssets.map((asset) => (
                <TableRow key={asset.id}>
                  <TableCell className="font-mono">{asset.asset_tag}</TableCell>
                  <TableCell className="font-medium">{asset.name}</TableCell>
                  <TableCell>{asset.asset_type_name || '-'}</TableCell>
                </TableRow>
              ))
            ) : (
                <TableRow><TableCell colSpan={3} className="text-center h-24">Anda tidak memiliki aset yang di-assign.</TableCell></TableRow>
            )}
          </TableBody>
        </Table>
      </div>
    </div>
  );
}