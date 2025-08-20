// File: src/pages/TicketsPage.tsx
import { useEffect, useState, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import apiClient from '@/services/api';
import toast from 'react-hot-toast';
import type { TicketInfo, PaginationData } from '@/types';

import { Button } from "@/components/ui/button";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Pagination } from '@/components/ui/pagination';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import CreateTicketModal from '@/components/CreateTicketModal';

export default function TicketsPage() {
  const [tickets, setTickets] = useState<TicketInfo[]>([]);
  const [pagination, setPagination] = useState<PaginationData | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [currentPage, setCurrentPage] = useState(1);
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const navigate = useNavigate();

  // State untuk filter
  const [statusFilter, setStatusFilter] = useState('all');
  const [priorityFilter, setPriorityFilter] = useState('all');

  // Gunakan useCallback untuk efisiensi
  const fetchTickets = useCallback((pageToFetch: number) => {
    setIsLoading(true);
    const params = {
      page: pageToFetch.toString(),
      limit: '10',
      ...(statusFilter !== 'all' && { status: statusFilter }),
      ...(priorityFilter !== 'all' && { priority: priorityFilter }),
    };
    const queryString = new URLSearchParams(params).toString();

    apiClient.get(`/tickets?${queryString}`)
      .then(res => {
        setTickets(res.data.data);
        setPagination(res.data.pagination);
      })
      .catch(() => toast.error('Gagal memuat data tiket.'))
      .finally(() => setIsLoading(false));
  }, [statusFilter, priorityFilter]); // Dependensi hanya pada filter

  // useEffect untuk memantau perubahan filter
  useEffect(() => {
    const handler = setTimeout(() => {
      if (currentPage !== 1) {
        setCurrentPage(1);
      } else {
        fetchTickets(1);
      }
    }, 500); // Debounce
    return () => clearTimeout(handler);
  }, [statusFilter, priorityFilter]);

  // useEffect untuk memantau perubahan halaman
  useEffect(() => {
    fetchTickets(currentPage);
  }, [currentPage, fetchTickets]);

  const getStatusVariant = (status: string) => {
    switch (status.toLowerCase()) {
      case 'open': return 'default';
      case 'in progress': return 'secondary';
      case 'closed': return 'outline';
      default: return 'secondary';
    }
  };

  const handleSuccess = () => {
    setIsCreateModalOpen(false);
    fetchTickets(currentPage);
  };

  return (
    <div className="container mx-auto py-8">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-3xl font-bold">Help Desk Tiket</h1>
        <Button onClick={() => setIsCreateModalOpen(true)}>+ Buat Tiket Baru</Button>
      </div>

      <div className="bg-white p-4 mb-6 border rounded-lg flex items-center space-x-4">
        <Select value={statusFilter} onValueChange={setStatusFilter}>
          <SelectTrigger className="w-[180px]"><SelectValue placeholder="Semua Status" /></SelectTrigger>
          <SelectContent>
            <SelectItem value="all">Semua Status</SelectItem>
            <SelectItem value="Open">Open</SelectItem>
            <SelectItem value="In Progress">In Progress</SelectItem>
            <SelectItem value="Closed">Closed</SelectItem>
          </SelectContent>
        </Select>
        <Select value={priorityFilter} onValueChange={setPriorityFilter}>
          <SelectTrigger className="w-[180px]"><SelectValue placeholder="Semua Prioritas" /></SelectTrigger>
          <SelectContent>
            <SelectItem value="all">Semua Prioritas</SelectItem>
            <SelectItem value="Low">Low</SelectItem>
            <SelectItem value="Medium">Medium</SelectItem>
            <SelectItem value="High">High</SelectItem>
            <SelectItem value="Critical">Critical</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <div className="bg-white p-4 border rounded-lg">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>ID</TableHead>
              <TableHead>Subjek</TableHead>
              <TableHead>Dilaporkan Oleh</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Prioritas</TableHead>
              <TableHead>Update Terakhir</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <TableRow><TableCell colSpan={6} className="text-center h-24">Loading...</TableCell></TableRow>
            ) : (
              tickets.map((ticket) => (
                <TableRow 
                  key={ticket.id} 
                  onClick={() => navigate(`/tickets/${ticket.id}`)}
                  className="cursor-pointer hover:bg-muted/50"
                >
                  <TableCell className="font-mono">#{ticket.id}</TableCell>
                  <TableCell className="font-medium">{ticket.subject}</TableCell>
                  <TableCell>{ticket.created_by_employee_name}</TableCell>
                  <TableCell>
                    <Badge variant={getStatusVariant(ticket.status)}>{ticket.status}</Badge>
                  </TableCell>
                  <TableCell>{ticket.priority}</TableCell>
                  <TableCell>{new Date(ticket.updated_at).toLocaleString('id-ID')}</TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>

      {pagination && <Pagination currentPage={pagination.current_page} totalPages={pagination.total_pages} onPageChange={setCurrentPage} />}
      
      <CreateTicketModal
        isOpen={isCreateModalOpen}
        onClose={() => setIsCreateModalOpen(false)}
        onSuccess={handleSuccess}
      />
    </div>
  );
}