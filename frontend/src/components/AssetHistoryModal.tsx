// File: src/components/AssetHistoryModal.tsx
import { useState, useEffect } from 'react';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from "../components/ui/dialog";
import apiClient from '../services/api';
import type { AssetHistoryResponse } from '../types'; // Pastikan tipe ini ada di src/types/index.ts
import toast from 'react-hot-toast';

export default function AssetHistoryModal({ isOpen, onClose, assetId }: { isOpen: boolean; onClose: () => void; assetId: number | null; }) {
  const [history, setHistory] = useState<AssetHistoryResponse[]>([]);
  const [isLoading, setIsLoading] = useState(false);

  useEffect(() => {
    if (isOpen && assetId) {
      setIsLoading(true);
      apiClient.get(`/assets/${assetId}/history`)
        .then(res => setHistory(res.data))
        .catch(() => toast.error('Gagal memuat riwayat aset.'))
        .finally(() => setIsLoading(false));
    }
  }, [isOpen, assetId]);

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="sm:max-w-xl">
        <DialogHeader>
          <DialogTitle>Riwayat Aset</DialogTitle>
          <DialogDescription>Menampilkan semua riwayat perpindahan dan penugasan aset.</DialogDescription>
        </DialogHeader>
        <div className="py-4 max-h-[60vh] overflow-y-auto">
          {isLoading ? <p>Loading...</p> : (
            <div className="space-y-4">
              {history.length > 0 ? (
                history.map((record) => (
                  <div key={record.assignment_id} className="p-3 border rounded-md bg-muted/50">
                    <p className="font-semibold">{record.employee_name} ({record.employee_nik})</p>
                    <p className="text-sm text-muted-foreground">
                      <span className="font-medium">Assign:</span> {new Date(record.assigned_at).toLocaleString('id-ID')}
                    </p>
                    {record.returned_at && (
                      <p className="text-sm text-muted-foreground">
                        <span className="font-medium">Return:</span> {new Date(record.returned_at).toLocaleString('id-ID')}
                      </p>
                    )}
                    {record.notes && (
                      <p className="text-sm text-foreground mt-1 border-l-2 border-slate-300 pl-2">
                        <i>"{record.notes}"</i>
                      </p>
                    )}
                  </div>
                ))
              ) : <p>Tidak ada riwayat untuk aset ini.</p>}
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}