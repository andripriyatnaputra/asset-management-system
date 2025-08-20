// File: src/pages/TicketDetailPage.tsx
import { useEffect, useState, type FormEvent, useMemo } from 'react';
import { useParams } from 'react-router-dom';
import apiClient from '../src/services/api';
import toast from 'react-hot-toast';
import { jwtDecode } from 'jwt-decode';
import type { TicketDetail } from '../src/types';

import { Card, CardContent, CardHeader, CardTitle } from "../src/components/ui/card";
import { Badge } from "../src/components/ui/badge";
import { Button } from '../src/components/ui/button';
import { Textarea } from '../src/components/ui/textarea';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../src/components/ui/select";
import { Label } from '../src/components/ui/label';
import { Avatar, AvatarFallback } from "../src/components/ui/avatar";

// Tipe untuk data yang di-decode dari token JWT
interface DecodedToken {
  role: string;
}

export default function TicketDetailPage() {
  const { id } = useParams<{ id: string }>();
  const [ticket, setTicket] = useState<TicketDetail | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [newComment, setNewComment] = useState('');

  // Ambil peran user dari token untuk menampilkan/menyembunyikan kontrol admin
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
  
  const fetchTicketDetails = () => {
    if (id) {
      setIsLoading(true);
      apiClient.get(`/tickets/${id}`)
        .then(res => setTicket(res.data))
        .catch(() => toast.error('Gagal memuat detail tiket.'))
        .finally(() => setIsLoading(false));
    }
  };

  useEffect(() => {
    fetchTicketDetails();
  }, [id]);

  const handleCommentSubmit = (e: FormEvent) => {
    e.preventDefault();
    if (!newComment.trim()) return;
    const promise = apiClient.post(`/tickets/${id}/comments`, { comment: newComment });
    toast.promise(promise, {
        loading: 'Mengirim komentar...',
        success: () => {
            setNewComment('');
            fetchTicketDetails();
            return 'Komentar berhasil ditambahkan!';
        },
        error: 'Gagal menambahkan komentar.'
    });
  };
  
  const handleTicketUpdate = (field: 'status' | 'priority', value: string) => {
    if (!ticket) return;

    const payload = {
      status: ticket.status,
      priority: ticket.priority,
      [field]: value,
    };

    const promise = apiClient.put(`/tickets/${id}`, payload);
    toast.promise(promise, {
      loading: 'Memperbarui tiket...',
      success: () => {
        fetchTicketDetails();
        return 'Tiket berhasil diperbarui!';
      },
      error: 'Gagal memperbarui tiket.'
    });
  };

  // --- FUNGSI YANG HILANG DITAMBAHKAN DI SINI ---
  const getStatusVariant = (status: string) => {
    switch (status.toLowerCase()) {
      case 'open': return 'default';
      case 'in progress': return 'secondary';
      case 'closed': return 'outline';
      default: return 'secondary';
    }
  };

  if (isLoading) { return <div className="p-8">Loading ticket details...</div>; }
  if (!ticket) { return <div className="p-8">Ticket not found.</div>; }

  return (
    <div className="container mx-auto py-8">
      {/* Header Halaman */}
      <div>
        <p className="text-muted-foreground">Ticket #{ticket.id}</p>
        <h1 className="text-3xl font-bold">{ticket.subject}</h1>
        <div className="flex items-center space-x-4 mt-2">
          <Badge variant={getStatusVariant(ticket.status)}>{ticket.status}</Badge>
          <span className="text-sm text-muted-foreground">
            Dibuat oleh {ticket.created_by_employee_name}
          </span>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 lg:grid-cols-4 gap-6 mt-6">

        {/* Kolom Kiri: Alur Percakapan */}
        <div className="md:col-span-2 lg:col-span-3 space-y-6">
          <Card>
            <CardHeader><CardTitle>Deskripsi Masalah</CardTitle></CardHeader>
            <CardContent><p className="whitespace-pre-wrap text-sm">{ticket.description || "Tidak ada deskripsi."}</p></CardContent>
          </Card>
          
          <div className="space-y-6">
            {ticket.comments.map(comment => (
              <div key={comment.id} className="flex items-start space-x-3">
                <Avatar><AvatarFallback>{comment.employee_name.charAt(0)}</AvatarFallback></Avatar>
                <div className="flex-1">
                  <div className="bg-white p-4 border rounded-lg">
                    <div className="flex justify-between items-center mb-1">
                      <p className="font-semibold text-sm">{comment.employee_name}</p>
                      <p className="text-xs text-muted-foreground">{new Date(comment.created_at).toLocaleString('id-ID')}</p>
                    </div>
                    <p className="text-sm">{comment.comment}</p>
                  </div>
                </div>
              </div>
            ))}
          </div>

          <Card>
            <CardHeader><CardTitle>Tambah Komentar</CardTitle></CardHeader>
            <CardContent>
              <form onSubmit={handleCommentSubmit} className="space-y-4">
                <Textarea value={newComment} onChange={e => setNewComment(e.target.value)} placeholder="Tulis balasan..." required />
                <Button type="submit">Kirim Komentar</Button>
              </form>
            </CardContent>
          </Card>
        </div>

        {/* Kolom Kanan: Panel Detail */}
        <div className="md:col-span-1 lg:col-span-1 space-y-4">
          <Card>
            <CardHeader><CardTitle>Detail Tiket</CardTitle></CardHeader>
            <CardContent className="space-y-4 text-sm">
              <div>
                <Label className="text-xs">Status</Label>
                {userRole === 'super_admin' ? (
                  <Select value={ticket.status} onValueChange={(value) => handleTicketUpdate('status', value)}>
                    <SelectTrigger><SelectValue /></SelectTrigger>
                    <SelectContent>
                      <SelectItem value="Open">Open</SelectItem>
                      <SelectItem value="In Progress">In Progress</SelectItem>
                      <SelectItem value="Closed">Closed</SelectItem>
                    </SelectContent>
                  </Select>
                ) : <p className="font-semibold">{ticket.status}</p>}
              </div>
              <div>
                <Label className="text-xs">Prioritas</Label>
                {userRole === 'super_admin' ? (
                  <Select value={ticket.priority} onValueChange={(value) => handleTicketUpdate('priority', value)}>
                    <SelectTrigger><SelectValue /></SelectTrigger>
                    <SelectContent>
                       <SelectItem value="Low">Low</SelectItem>
                       <SelectItem value="Medium">Medium</SelectItem>
                       <SelectItem value="High">High</SelectItem>
                       <SelectItem value="Critical">Critical</SelectItem>
                    </SelectContent>
                  </Select>
                ) : <p className="font-semibold">{ticket.priority}</p>}
              </div>
              <div>
                <Label className="text-xs">Dibuat Pada</Label>
                <p className="font-semibold">{new Date(ticket.created_at).toLocaleString('id-ID')}</p>
              </div>
               <div>
                <Label className="text-xs">Update Terakhir</Label>
                <p className="font-semibold">{new Date(ticket.updated_at).toLocaleString('id-ID')}</p>
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardHeader><CardTitle>Log Perbaikan Terkait</CardTitle></CardHeader>
            <CardContent className="space-y-2 text-sm">
              {ticket.maintenance_logs && ticket.maintenance_logs.length > 0 ? (
                ticket.maintenance_logs.map(log => (
                  <div key={log.id} className="border-b pb-1">
                    <p className="font-medium">{log.log_type}: {new Date(log.log_date).toLocaleDateString('id-ID')}</p>
                    <p className="text-muted-foreground">{log.description}</p>
                  </div>
                ))
              ) : <p className="text-muted-foreground">Belum ada log perbaikan.</p>}
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}