// File: src/pages/AuditSessionPage.tsx
import { useEffect, useState, useCallback } from 'react';
import { useParams, useNavigate  } from 'react-router-dom';
import apiClient from '../src/services/api';
import toast from 'react-hot-toast';
import type { AuditSession } from '../src/types';

import { Button } from "../src/components/ui/button";
import { Input } from "../src/components/ui/input";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "../src/components/ui/table";
import { Badge } from '../src/components/ui/badge';
import { Card, CardContent, CardHeader, CardTitle } from "../src/components/ui/card";
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "../src/components/ui/alert-dialog";


interface AuditedAssetInfo {
  asset_name: string;
  asset_tag: string;
  audit_status: string;
}

export default function AuditSessionPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate()
  const [session, setSession] = useState<AuditSession | null>(null);
  const [items, setItems] = useState<AuditedAssetInfo[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [scanInput, setScanInput] = useState('');
  const [isConfirmOpen, setIsConfirmOpen] = useState(false);

  const fetchDetails = useCallback(() => {
    if (!id) return;
    setIsLoading(true);
    apiClient.get(`/audits/${id}`)
      .then(res => {
        setSession(res.data.session);
        setItems(res.data.items);
      })
      .catch(() => toast.error('Gagal memuat detail sesi audit.'))
      .finally(() => setIsLoading(false));
  }, [id]);

  useEffect(() => {
    fetchDetails();
  }, [fetchDetails]);

  const handleScan = (e: React.FormEvent) => {
    e.preventDefault();
    if (!scanInput.trim()) return;

    const promise = apiClient.post(`/audits/${id}/scan`, { asset_tag: scanInput });
    toast.promise(promise, {
      loading: 'Memindai...',
      success: () => {
        setScanInput('');
        fetchDetails(); // Refresh data
        return `Aset ${scanInput} berhasil ditemukan!`;
      },
      error: (err) => err.response?.data?.error || 'Gagal memindai aset.',
    });
  };

  const handleCompleteAudit = () => {
    const promise = apiClient.put(`/audits/${id}/complete`);
    toast.promise(promise, {
      loading: 'Menyelesaikan sesi...',
      success: () => {
        navigate('/audits'); // Go back to the list after completion
        return 'Sesi audit telah selesai!';
      },
      error: 'Gagal menyelesaikan sesi.',
    });
  };

  const foundCount = items.filter(item => item.audit_status === 'Found').length;
  const missingCount = items.length - foundCount;
  const totalCount = items.length;

  return (
    <div className="container mx-auto py-8">
      <div className="mb-6 flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold">{session?.name}</h1>
          <Badge variant={session?.status === 'Completed' ? 'secondary' : 'default'}>{session?.status}</Badge>
        </div>
        {session?.status !== 'Completed' && (
          <Button variant="destructive" onClick={() => setIsConfirmOpen(true)}>Selesaikan Audit</Button>
        )}
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Kolom Kanan: Scan & Progress */}
        <div className="lg:col-span-1 space-y-4">
          {/* Hide scan form if session is completed */}
          {session?.status !== 'Completed' && (
            <Card>
              <CardHeader><CardTitle>Pindai Aset</CardTitle></CardHeader>
              <CardContent><form onSubmit={handleScan} className="space-y-2"><Input placeholder="Ketik Asset Tag..." value={scanInput} onChange={e => setScanInput(e.target.value)} /><Button type="submit" className="w-full">Pindai</Button></form></CardContent>
            </Card>
          )}
          <Card>
            <CardHeader><CardTitle>Laporan Ringkas</CardTitle></CardHeader>
            <CardContent className="grid grid-cols-3 gap-2 text-center">
              <div><p className="text-2xl font-bold">{foundCount}</p><p className="text-sm text-muted-foreground">Ditemukan</p></div>
              <div><p className="text-2xl font-bold text-destructive">{missingCount}</p><p className="text-sm text-muted-foreground">Hilang</p></div>
              <div><p className="text-2xl font-bold">{totalCount}</p><p className="text-sm text-muted-foreground">Total</p></div>
            </CardContent>
          </Card>
        </div>

        {/* Kolom Kiri: Daftar Periksa Aset */}
        <div className="lg:col-span-2 bg-white p-4 border rounded-lg">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Nama Aset</TableHead>
                <TableHead>Tag Aset</TableHead>
                <TableHead>Status Audit</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {isLoading ? (
                <TableRow><TableCell colSpan={3} className="text-center h-24">Loading...</TableCell></TableRow>
              ) : (
                items.map((item, index) => (
                  <TableRow key={index}>
                    <TableCell className="font-medium">{item.asset_name}</TableCell>
                    <TableCell className="font-mono">{item.asset_tag}</TableCell>
                    <TableCell>
                      <Badge variant={item.audit_status === 'Found' ? 'default' : 'destructive'}>
                        {item.audit_status}
                      </Badge>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </div>
      </div>

       <AlertDialog open={isConfirmOpen} onOpenChange={setIsConfirmOpen}>
        <AlertDialogContent>
          <AlertDialogHeader><AlertDialogTitle>Selesaikan Sesi Audit?</AlertDialogTitle><AlertDialogDescription>Setelah diselesaikan, Anda tidak bisa lagi memindai aset di sesi ini. Apakah Anda yakin?</AlertDialogDescription></AlertDialogHeader>
          <AlertDialogFooter><AlertDialogCancel>Batal</AlertDialogCancel><AlertDialogAction onClick={handleCompleteAudit}>Ya, Selesaikan</AlertDialogAction></AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}