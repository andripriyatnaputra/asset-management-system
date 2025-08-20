// File: src/components/AssetDetailsModal.tsx
import { useState, useEffect } from 'react';
import apiClient from '../services/api';
import toast from 'react-hot-toast';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from "../components/ui/dialog";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Button } from "../components/ui/button";
import type { DepreciationInfo, InstalledSoftwareInfo } from '../types'; // Pastikan tipe ini ada

import InstallSoftwareModal from './InstallSoftwareModal';

const formatCurrency = (value: number) => new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR' }).format(value);

export default function AssetDetailsModal({ isOpen, onClose, assetId }: { isOpen: boolean; onClose: () => void; assetId: number | null; }) {
  const [details, setDetails] = useState<DepreciationInfo | null>(null);
  const [installedSoftware, setInstalledSoftware] = useState<InstalledSoftwareInfo[] | null>(null);
  const [isInstallModalOpen, setIsInstallModalOpen] = useState(false);
  const [isLoading, setIsLoading] = useState(false);

  const fetchAllDetails = () => {
    if (!assetId) return;
    setIsLoading(true);
    Promise.all([
      apiClient.get(`/assets/${assetId}/depreciation`),
      apiClient.get(`/assets/${assetId}/software`)
    ]).then(([depreciationRes, softwareRes]) => {
      setDetails(depreciationRes.data);
      setInstalledSoftware(softwareRes.data);
    }).catch(() => toast.error('Gagal memuat detail aset.'))
      .finally(() => setIsLoading(false));
  };

  useEffect(() => {
    if (isOpen) fetchAllDetails();
  }, [isOpen, assetId]);

  const handleInstallSuccess = () => {
    setIsInstallModalOpen(false);
    fetchAllDetails(); 
  };

  return (
    <>
      <Dialog open={isOpen} onOpenChange={onClose}>
        <DialogContent className="sm:max-w-3xl">
          <DialogHeader>
            <DialogTitle>Detail Aset: {details?.asset_name}</DialogTitle>
            <DialogDescription>Tag: {details?.asset_tag}</DialogDescription>
          </DialogHeader>
          {isLoading ? <div className="py-8 text-center">Loading...</div> : (
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6 py-4">
              <Card>
                <CardHeader><CardTitle>Informasi Finansial</CardTitle></CardHeader>
                <CardContent className="space-y-2 text-sm">
                  <p><strong>Tgl Beli:</strong> {details?.purchase_date}</p>
                  <p><strong>Harga Awal:</strong> {formatCurrency(details?.initial_price || 0)}</p>
                  <p><strong>Nilai Buku Saat Ini:</strong> <span className="font-bold">{formatCurrency(details?.current_book_value || 0)}</span></p>
                </CardContent>
              </Card>
              <Card>
                <CardHeader>
                  <div className="flex justify-between items-center">
                    <CardTitle>Software Terinstal</CardTitle>
                    <Button size="sm" onClick={() => setIsInstallModalOpen(true)}>+ Install</Button>
                  </div>
                </CardHeader>
                <CardContent className="space-y-2 max-h-40 overflow-y-auto">
                  {installedSoftware && installedSoftware.length > 0 ? (
                    installedSoftware.map(sw => (
                      <div key={sw.installation_id} className="text-sm">
                        <p className="font-medium">{sw.license_name}</p>
                        <p className="text-xs text-muted-foreground">Diinstal: {new Date(sw.installation_date).toLocaleDateString('id-ID')}</p>
                      </div>
                    ))
                  ) : <p className="text-sm text-muted-foreground">Tidak ada software.</p>}
                </CardContent>
              </Card>
            </div>
          )}
        </DialogContent>
      </Dialog>

      <InstallSoftwareModal
        isOpen={isInstallModalOpen}
        onClose={() => setIsInstallModalOpen(false)}
        onSuccess={handleInstallSuccess}
        assetId={assetId}
      />
    </>
  );
}