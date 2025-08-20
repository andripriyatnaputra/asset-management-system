// File: src/pages/AuditsPage.tsx
import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import apiClient from '../src/services/api';
import toast from 'react-hot-toast';
import type { AuditSession } from '../src/types';

import { Button } from "../src/components/ui/button";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "../src/components/ui/table";
import { Badge } from "../src/components/ui/badge";
import CreateAuditModal from '../src/components/CreateAuditModal';

export default function AuditsPage() {
  const [sessions, setSessions] = useState<AuditSession[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const navigate = useNavigate();

  const fetchSessions = () => {
    setIsLoading(true);
    apiClient.get('/audits')
      .then(res => setSessions(res.data))
      .catch(() => toast.error('Gagal memuat sesi audit.'))
      .finally(() => setIsLoading(false));
  };

  useEffect(() => {
    fetchSessions();
  }, []);

  const handleSuccess = () => {
    setIsModalOpen(false);
    fetchSessions(); // Refresh daftar sesi audit
  };

  return (
    <div className="container mx-auto py-8">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-3xl font-bold">Sesi Audit Aset</h1>
        <Button onClick={() => setIsModalOpen(true)}>+ Mulai Sesi Baru</Button>
      </div>

      <div className="bg-white p-4 border rounded-lg">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Nama Sesi</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Tanggal Dibuat</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <TableRow><TableCell colSpan={3} className="text-center h-24">Loading...</TableCell></TableRow>
            ) : (
              sessions && sessions.map((session) => (
                <TableRow 
                  key={session.id} 
                  onClick={() => navigate(`/audits/${session.id}`)}
                  className="cursor-pointer hover:bg-muted/50"
                >
                  <TableCell className="font-medium">{session.name}</TableCell>
                  <TableCell><Badge variant={session.status === 'Completed' ? 'secondary' : 'default'}>{session.status}</Badge></TableCell>
                  <TableCell>{new Date(session.created_at).toLocaleDateString('id-ID')}</TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>
      
      <CreateAuditModal
        isOpen={isModalOpen}
        onClose={() => setIsModalOpen(false)}
        onSuccess={handleSuccess}
      />
    </div>
  );
}