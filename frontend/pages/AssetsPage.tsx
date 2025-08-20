// File: src/pages/AssetsPage.tsx
import { useState, useEffect } from 'react';
import { useAssetLogic } from '../src/hooks/useAssetLogic';
import apiClient from '../src/services/api';
import toast from 'react-hot-toast';
import type { AssetType } from '../src/types';
import { useMemo } from 'react'; 
import { jwtDecode } from 'jwt-decode';

// Impor komponen UI
import { Button } from '../src/components/ui/button';
import { Input } from "../src/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../src/components/ui/select";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "../src/components/ui/table";
import { Badge } from "../src/components/ui/badge";
import { Pagination } from '../src/components/ui/pagination';
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "../src/components/ui/alert-dialog";

// Impor semua modal yang dibutuhkan
import AddAssetModal from '../src/components/AddAssetModal';
import EditAssetModal from '../src/components/EditAssetModal';
import AssignAssetModal from '../src/components/AssignAssetModal';
import ReturnAssetModal from '../src/components/ReturnAssetModal';
import AssetHistoryModal from '../src/components/AssetHistoryModal';
import AssetDetailsModal from '../src/components/AssetDetailsModal';
import MaintenanceLogModal from '../src/components/MaintenanceLogModal';
import AssetTypeManagerModal from '../src/components/AssetTypeManagerModal';

interface DecodedToken { role: string; }

export default function AssetsPage() {
  const userRole = useMemo(() => {
      const token = localStorage.getItem('authToken');
      if (!token) return null;
      try {
        const decoded: DecodedToken = jwtDecode(token);
        return decoded.role;
      } catch (error) { 
        console.error("Invalid token:", error);
        return null; 
      }
    }, []);

  // --- Menggunakan Hook untuk Logika Utama ---
  const {
    assets, pagination, isLoading, searchTerm, setSearchTerm,
    selectedType, setSelectedType, setCurrentPage, handleSort,
    fetchAssets // Kita ambil fungsi fetchAssets untuk me-refresh data
  } = useAssetLogic();
  
  // --- State Lokal hanya untuk UI (Modal & Data Tambahan) ---
  const [assetTypes, setAssetTypes] = useState<AssetType[]>([]);
  const [selectedAssetId, setSelectedAssetId] = useState<number | null>(null);
  
  // State untuk setiap modal
  const [isAddModalOpen, setIsAddModalOpen] = useState(false);
  const [isEditModalOpen, setIsEditModalOpen] = useState(false);
  const [isDeleteConfirmOpen, setIsDeleteConfirmOpen] = useState(false);
  const [isAssignModalOpen, setIsAssignModalOpen] = useState(false);
  const [isReturnModalOpen, setIsReturnModalOpen] = useState(false);
  const [isHistoryModalOpen, setIsHistoryModalOpen] = useState(false);
  const [isDetailsModalOpen, setIsDetailsModalOpen] = useState(false);
  const [isMaintenanceModalOpen, setIsMaintenanceModalOpen] = useState(false);
  const [isAssetTypeModalOpen, setIsAssetTypeModalOpen] = useState(false);
  
  // Ambil data tipe aset untuk dropdown filter
  //useEffect(() => {
  //  console.log("Data Tipe Aset Diterima:", response.data);
  //  apiClient.get('/asset-types').then(response => setAssetTypes(response.data));
  //}, []);
  useEffect(() => {
    apiClient.get('/asset-types').then(response => {
      console.log("Data Tipe Aset Diterima:", response.data); // Untuk debugging
      
      // Perbaikan: Cek jika data ada di dalam properti 'data' atau tidak
      const typesData = Array.isArray(response.data) ? response.data : response.data.data;
      
      // Pastikan kita selalu mengatur state dengan sebuah array
      setAssetTypes(Array.isArray(typesData) ? typesData : []);
    });
  }, []);

  // --- Fungsi Handle untuk Interaksi UI ---
  const handleCloseModals = () => {
    setIsAddModalOpen(false);
    setIsEditModalOpen(false);
    setIsAssignModalOpen(false);
    setIsReturnModalOpen(false);
    setIsHistoryModalOpen(false);
    setIsDetailsModalOpen(false);
    setIsMaintenanceModalOpen(false);
    setIsAssetTypeModalOpen(false);
    setSelectedAssetId(null);
  };

  const handleSuccess = (message: string) => {
    handleCloseModals();
    fetchAssets(pagination?.current_page || 1);
    toast.success(message);
  };

  // Fungsi handle untuk membuka setiap modal
  const handleEditClick = (id: number) => { setSelectedAssetId(id); setIsEditModalOpen(true); };
  const handleDeleteClick = (id: number) => { setSelectedAssetId(id); setIsDeleteConfirmOpen(true); };
  const handleAssignClick = (id: number) => { setSelectedAssetId(id); setIsAssignModalOpen(true); };
  const handleReturnClick = (id: number) => { setSelectedAssetId(id); setIsReturnModalOpen(true); };
  const handleHistoryClick = (id: number) => { setSelectedAssetId(id); setIsHistoryModalOpen(true); };
  const handleDetailClick = (id: number) => { setSelectedAssetId(id); setIsDetailsModalOpen(true); };
  const handleMaintenanceClick = (id: number) => { setSelectedAssetId(id); setIsMaintenanceModalOpen(true); };

  // Fungsi untuk konfirmasi delete
  const handleConfirmDelete = () => {
    if (!selectedAssetId) return;
    const promise = apiClient.delete(`/assets/${selectedAssetId}`);
    toast.promise(promise, {
        loading: 'Menghapus aset...',
        success: () => {
            fetchAssets(1);
            return 'Aset berhasil dihapus!';
        },
        error: 'Gagal menghapus aset.',
    });
    handleCloseModals();
  };


  return (
    <div className="container mx-auto py-8">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-3xl font-bold">Manajemen Aset</h1>
        {/* Tombol ini hanya muncul untuk admin */}
        {userRole === 'super_admin' && (
          <Button onClick={() => setIsAddModalOpen(true)}>+ Tambah Aset Baru</Button>
        )} </div>

      <div className="bg-white p-4 mb-6 border rounded-lg flex flex-wrap items-center justify-between gap-4">
        <div className="flex-grow sm:flex-grow-0">
          <Input placeholder="Cari berdasarkan nama aset..." value={searchTerm} onChange={(e) => setSearchTerm(e.target.value)} className="w-full sm:w-64" />
        </div>
        <div className="flex items-center space-x-2">
          <Select value={selectedType} onValueChange={setSelectedType}>
            <SelectTrigger className="w-[180px]"><SelectValue placeholder="Semua Tipe Aset" /></SelectTrigger>
            <SelectContent>
              <SelectItem value="all">Semua Tipe Aset</SelectItem>
              {assetTypes.map(type => (<SelectItem key={type.id} value={type.id.toString()}>{type.name}</SelectItem>))}
            </SelectContent>
          </Select>
          <Button variant="outline" onClick={() => setIsAssetTypeModalOpen(true)}>Kelola Tipe</Button>
        </div>
      </div>
      
      <div className="bg-white p-4 border rounded-lg">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead onClick={() => handleSort('asset_tag')} className="cursor-pointer hover:bg-gray-100">Tag Aset</TableHead>
              <TableHead onClick={() => handleSort('name')} className="cursor-pointer hover:bg-gray-100">Nama</TableHead>
              <TableHead>Tipe</TableHead>
              <TableHead onClick={() => handleSort('status')} className="cursor-pointer hover:bg-gray-100">Status</TableHead>
              <TableHead onClick={() => handleSort('initial_price')} className="cursor-pointer hover:bg-gray-100 text-right">Harga</TableHead>
              {userRole === 'super_admin' && <TableHead className="text-center w-[320px]">Aksi</TableHead>}
             </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <TableRow><TableCell colSpan={6} className="text-center h-24">Loading...</TableCell></TableRow>
            ) : (
              assets.map((asset) => (
                <TableRow key={asset.id}>
                  <TableCell className="font-mono">{asset.asset_tag}</TableCell>
                  <TableCell className="font-medium">{asset.name}</TableCell>
                  <TableCell>{asset.asset_type_name || '-'}</TableCell>
                  <TableCell><Badge variant={asset.status === 'In Stock' ? 'default' : 'secondary'}>{asset.status}</Badge></TableCell>
                  <TableCell className="text-right">{new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR' }).format(asset.initial_price)}</TableCell>
                  {userRole === 'super_admin' && (
                  <TableCell className="flex items-center justify-center space-x-1">
                    {asset.status === 'In Stock' && (<Button variant="outline" size="sm" onClick={() => handleAssignClick(asset.id)}>Assign</Button>)}
                    {asset.status === 'Assigned' && (<Button variant="outline" size="sm" onClick={() => handleReturnClick(asset.id)}>Return</Button>)}
                    <Button variant="ghost" size="sm" onClick={() => handleHistoryClick(asset.id)}>History</Button>
                    <Button variant="ghost" size="sm" onClick={() => handleDetailClick(asset.id)}>Detail</Button>
                    <Button variant="ghost" size="sm" onClick={() => handleMaintenanceClick(asset.id)}>Logs</Button>
                    <Button variant="outline" size="sm" onClick={() => handleEditClick(asset.id)}>Edit</Button>
                    <Button variant="destructive" size="sm" onClick={() => handleDeleteClick(asset.id)}>Delete</Button>
                  </TableCell>
                  )}
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>

      {pagination && <Pagination currentPage={pagination.current_page} totalPages={pagination.total_pages} onPageChange={setCurrentPage} />}

      {/* --- Render Semua Modal --- */}
      <AddAssetModal isOpen={isAddModalOpen} onClose={handleCloseModals} onSuccess={() => handleSuccess('Aset baru berhasil ditambahkan!')} />
      <EditAssetModal isOpen={isEditModalOpen} onClose={handleCloseModals} onSuccess={() => handleSuccess('Aset berhasil diperbarui!')} assetId={selectedAssetId} />
      <AssignAssetModal isOpen={isAssignModalOpen} onClose={handleCloseModals} onSuccess={() => handleSuccess('Aset berhasil di-assign!')} assetId={selectedAssetId} />
      <ReturnAssetModal isOpen={isReturnModalOpen} onClose={handleCloseModals} onSuccess={() => handleSuccess('Aset berhasil dikembalikan!')} assetId={selectedAssetId} />
      <AssetHistoryModal isOpen={isHistoryModalOpen} onClose={handleCloseModals} assetId={selectedAssetId} />
      <AssetDetailsModal isOpen={isDetailsModalOpen} onClose={handleCloseModals} assetId={selectedAssetId} />
      <MaintenanceLogModal isOpen={isMaintenanceModalOpen} onClose={handleCloseModals} assetId={selectedAssetId} />
      <AssetTypeManagerModal isOpen={isAssetTypeModalOpen} onClose={handleCloseModals} />
      
      <AlertDialog open={isDeleteConfirmOpen} onOpenChange={setIsDeleteConfirmOpen}>
          <AlertDialogContent>
              <AlertDialogHeader><AlertDialogTitle>Apakah Anda Yakin?</AlertDialogTitle><AlertDialogDescription>Tindakan ini akan menghapus data aset secara permanen (soft delete).</AlertDialogDescription></AlertDialogHeader>
              <AlertDialogFooter><AlertDialogCancel>Batal</AlertDialogCancel><AlertDialogAction onClick={handleConfirmDelete}>Ya, Hapus</AlertDialogAction></AlertDialogFooter>
          </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}