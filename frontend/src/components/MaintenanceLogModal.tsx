// File: src/components/MaintenanceLogModal.tsx
import { useState, useEffect } from 'react';
import apiClient from '../services/api';
import toast from 'react-hot-toast';

import { Button } from "../components/ui/button";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "../components/ui/dialog";
import { Input } from "../components/ui/input";
import { Label } from "../components/ui/label";
import { Textarea } from "../components/ui/textarea";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../components/ui/select";
import type { AssetMaintenanceLog, TicketInfo } from '../types';

interface MaintenanceLogModalProps {
  isOpen: boolean;
  onClose: () => void;
  assetId: number | null;
}

export default function MaintenanceLogModal({ isOpen, onClose, assetId }: MaintenanceLogModalProps) {
  const [logs, setLogs] = useState<AssetMaintenanceLog[]>([]);
  const [openTickets, setOpenTickets] = useState<TicketInfo[]>([]);
  
  // State untuk form
  const [logType, setLogType] = useState('Maintenance');
  const [description, setDescription] = useState('');
  const [cost, setCost] = useState(0);
  const [logDate, setLogDate] = useState(new Date().toISOString().split('T')[0]);
  const [selectedTicketId, setSelectedTicketId] = useState<string | undefined>(undefined);

  const [isLoading, setIsLoading] = useState(false);

  const fetchLogsAndTickets = () => {
    if (!assetId) return;
    setIsLoading(true);
    
    // Ambil riwayat log yang sudah ada & tiket yang masih terbuka secara bersamaan
    Promise.all([
      apiClient.get(`/assets/${assetId}/maintenance-logs`),
      apiClient.get(`/tickets?status=Open&limit=100`) // Ambil tiket yang masih 'Open'
    ]).then(([logsRes, ticketsRes]) => {
      setLogs(logsRes.data);
      // Filter tiket yang relevan dengan aset ini (jika ada)
      const relevantTickets = ticketsRes.data.data.filter((ticket: TicketInfo) => ticket.related_asset_id === assetId);
      setOpenTickets(relevantTickets);
    }).catch(() => {
      toast.error("Gagal memuat data.");
    }).finally(() => {
      setIsLoading(false);
    });
  };

  useEffect(() => {
    if (isOpen) {
      fetchLogsAndTickets();
    }
  }, [isOpen, assetId]);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!assetId) return;

    const promise = apiClient.post(`/assets/${assetId}/maintenance-logs`, {
      log_type: logType,
      description,
      cost: Number(cost),
      log_date: new Date(logDate).toISOString(),
      ticket_id: selectedTicketId ? Number(selectedTicketId) : null,
    });

    toast.promise(promise, {
      loading: 'Menyimpan log...',
      success: () => {
        // Reset form
        setDescription(''); setCost(0); setSelectedTicketId(undefined);
        fetchLogsAndTickets(); // Refresh list log dan tiket
        return 'Log berhasil ditambahkan!';
      },
      error: 'Gagal menambahkan log.',
    });
  };
  
  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="sm:max-w-3xl">
        <DialogHeader><DialogTitle>Riwayat Maintenance & Perbaikan</DialogTitle></DialogHeader>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6 py-4">
          {/* Kolom Kiri: Daftar Log */}
          <div className="md:border-r pr-6 max-h-96 overflow-y-auto space-y-2">
            <h3 className="font-semibold mb-2">Riwayat</h3>
            {isLoading && <p>Loading...</p>}
            {!isLoading && logs && logs.length > 0 ? (
              logs.map(log => (
                <div key={log.id} className="text-sm border-b py-2">
                  <p className="font-bold">{log.log_type} - {new Date(log.log_date).toLocaleDateString('id-ID')}</p>
                  <p>{log.description}</p>
                  <p className="text-xs text-muted-foreground">Biaya: {new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR' }).format(log.cost)}</p>
                  {log.ticket_id && <p className="text-xs text-blue-600">Terkait Tiket #{log.ticket_id}</p>}
                </div>
              ))
            ) : !isLoading && (
              <p className="text-sm text-muted-foreground">Belum ada riwayat.</p>
            )}
          </div>

          {/* Kolom Kanan: Form Tambah Log Baru */}
          <form onSubmit={handleSubmit} className="space-y-4">
            <h3 className="font-semibold">Tambah Log Baru</h3>
            <div><Label>Tanggal</Label><Input type="date" value={logDate} onChange={e => setLogDate(e.target.value)} required /></div>
            <div><Label>Tipe Log</Label><Select value={logType} onValueChange={setLogType}><SelectTrigger><SelectValue/></SelectTrigger><SelectContent><SelectItem value="Maintenance">Maintenance</SelectItem><SelectItem value="Repair">Repair</SelectItem><SelectItem value="Upgrade">Upgrade</SelectItem></SelectContent></Select></div>
            <div><Label>Deskripsi</Label><Textarea value={description} onChange={e => setDescription(e.target.value)} required /></div>
            <div><Label>Biaya</Label><Input type="number" value={cost} onChange={e => setCost(Number(e.target.value))} /></div>
            <div>
              <Label>Tautkan ke Tiket (Opsional)</Label>
              <Select value={selectedTicketId} onValueChange={setSelectedTicketId}>
                  <SelectTrigger><SelectValue placeholder="Pilih tiket terkait..." /></SelectTrigger>
                  <SelectContent>
                    {openTickets && openTickets.length > 0 ? (
                      openTickets.map(ticket => (
                        <SelectItem key={ticket.id} value={ticket.id.toString()}>#{ticket.id} - {ticket.subject}</SelectItem>
                      ))
                    ) : (
                      <SelectItem value="none" disabled>Tidak ada tiket terbuka untuk aset ini</SelectItem>
                    )}
                  </SelectContent>
              </Select>
            </div>
            <Button type="submit" className="w-full">Simpan Log</Button>
          </form>
        </div>
      </DialogContent>
    </Dialog>
  );
}