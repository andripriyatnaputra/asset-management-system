// File: src/pages/LicensesPage.tsx
import { useEffect, useState } from 'react';
import apiClient from '../src/services/api';
import toast from 'react-hot-toast';
import type { SoftwareLicense } from '../src/types';

import { Button } from "../src/components/ui/button";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "../src/components/ui/table";
import LicenseFormModal from '../src/components/LicenseFormModal';

export default function LicensesPage() {
  const [licenses, setLicenses] = useState<SoftwareLicense[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingLicense, setEditingLicense] = useState<SoftwareLicense | null>(null);

  const fetchLicenses = () => {
    setIsLoading(true);
    apiClient.get('/licenses')
      .then(res => setLicenses(res.data))
      .catch(() => toast.error('Gagal memuat data lisensi.'))
      .finally(() => setIsLoading(false));
  };

  useEffect(() => {
    fetchLicenses();
  }, []);

  const handleOpenModal = (license: SoftwareLicense | null) => {
    setEditingLicense(license);
    setIsModalOpen(true);
  };

  const handleCloseModal = () => {
    setIsModalOpen(false);
    setEditingLicense(null);
  };

  const handleSuccess = () => {
    handleCloseModal();
    fetchLicenses();
  };

  return (
    <div className="container mx-auto py-8">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-3xl font-bold">Manajemen Lisensi Software</h1>
        <Button onClick={() => handleOpenModal(null)}>+ Tambah Lisensi</Button>
      </div>

      <div className="bg-white p-4 border rounded-lg">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Nama Software</TableHead>
              <TableHead>Jumlah Pengguna</TableHead>
              <TableHead>Tanggal Kedaluwarsa</TableHead>
              <TableHead className="text-right">Aksi</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <TableRow><TableCell colSpan={4} className="text-center h-24">Loading...</TableCell></TableRow>
            ) : (
              licenses && licenses.map((license) => (
                <TableRow key={license.id}>
                  <TableCell className="font-medium">{license.name}</TableCell>
                  <TableCell>{license.total_seats}</TableCell>
                  <TableCell>
                    {license.expiration_date ? new Date(license.expiration_date).toLocaleDateString('id-ID') : '-'}
                  </TableCell>
                  <TableCell className="text-right">
                    <Button variant="outline" size="sm" onClick={() => handleOpenModal(license)}>Edit</Button>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>
      
      <LicenseFormModal
        isOpen={isModalOpen}
        onClose={handleCloseModal}
        onSuccess={handleSuccess}
        license={editingLicense}
      />
    </div>
  );
}