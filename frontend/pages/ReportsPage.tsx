// File: src/pages/ReportsPage.tsx
import { useState, useEffect } from 'react';
import apiClient from '../src/services/api';
import toast from 'react-hot-toast';

import { Button } from "../src/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "../src/components/ui/card";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../src/components/ui/select";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "../src/components/ui/table";
import { BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer } from 'recharts';

import type { Department } from '../src/types';
import type { Employee } from '../src/types';

interface AssetReportRow {
  asset_name: string;
  asset_tag: string;
  asset_type: string;
  employee_name: string;
  employee_nik: string;
  assigned_at: string;
}

interface TicketReportRow {
  asset_type: string;
  ticket_count: number;
}

export default function ReportsPage() {
  const [departments, setDepartments] = useState<Department[]>([]);
  const [employees, setEmployees] = useState<Employee[]>([]);
  
  const [selectedDept, setSelectedDept] = useState<string>('');
  const [deptReportData, setDeptReportData] = useState<AssetReportRow[]>([]);
  const [isDeptLoading, setIsDeptLoading] = useState(false);

  const [selectedEmployee, setSelectedEmployee] = useState<string>('');
  const [employeeReportData, setEmployeeReportData] = useState<AssetReportRow[]>([]);
  const [isEmployeeLoading, setIsEmployeeLoading] = useState(false);

  const [ticketReport, setTicketReport] = useState<TicketReportRow[]>([]);
  

  useEffect(() => {
    // Ambil data awal untuk semua dropdown
    apiClient.get('/departments').then(res => setDepartments(res.data));
    apiClient.get('/employees?limit=500').then(res => setEmployees(res.data.data));
     apiClient.get('/reports/tickets-by-asset-type').then(res => setTicketReport(res.data));
  }, []);

  const handleGenerateDeptReport = () => {
    if (!selectedDept) {
      toast.error('Silakan pilih departemen terlebih dahulu.');
      return;
    }
    setIsDeptLoading(true);
    apiClient.get(`/reports/assets-by-department?department_id=${selectedDept}`)
      .then(res => {
        setDeptReportData(res.data);
        toast.success('Laporan per departemen berhasil dibuat!');
      })
      .catch(() => toast.error('Gagal membuat laporan.'))
      .finally(() => setIsDeptLoading(false));
  };

  const handleGenerateEmployeeReport = () => {
    if (!selectedEmployee) {
      toast.error('Silakan pilih karyawan terlebih dahulu.');
      return;
    }
    setIsEmployeeLoading(true);
    apiClient.get(`/reports/assets-by-employee?employee_id=${selectedEmployee}`)
      .then(res => {
        setEmployeeReportData(res.data);
        toast.success('Laporan per karyawan berhasil dibuat!');
      })
      .catch(() => toast.error('Gagal membuat laporan per karyawan.'))
      .finally(() => setIsEmployeeLoading(false));
  }

  const handleExportCSV = () => {
    if (!selectedDept) {
      toast.error('Silakan pilih departemen terlebih dahulu.');
      return;
    }
    const promise = apiClient.get(`/reports/assets-by-department?department_id=${selectedDept}&export=csv`, {
      responseType: 'blob', // Penting: Minta data sebagai file (blob)
    });// Buka URL di tab baru untuk memicu download

    toast.promise(promise, {
      loading: 'Mengekspor CSV...',
      success: (response) => {
        // 1. Buat URL sementara dari data blob yang diterima dari backend
        const url = window.URL.createObjectURL(new Blob([response.data]));
        
        // 2. Buat elemen link 'a' palsu di memori
        const link = document.createElement('a');
        link.href = url;
        
        // 3. Set nama file yang akan di-download
        link.setAttribute('download', `report-aset-departemen-${selectedDept}.csv`);
        
        // 4. Tambahkan link ke body dan "klik" secara otomatis untuk memicu download
        document.body.appendChild(link);
        link.click();
        
        // 5. Hapus link setelah selesai untuk kebersihan
        document.body.removeChild(link);
        window.URL.revokeObjectURL(url);
        
        return 'Ekspor berhasil!';
      },
      error: 'Gagal mengekspor CSV.',
    });

  };

return (
    <div className="container mx-auto py-8 space-y-6">
      <h1 className="text-3xl font-bold">Pelaporan</h1>

      <Card>
        <CardHeader>
          <CardTitle>Laporan Aset per Departemen</CardTitle>
          <CardDescription>Pilih departemen untuk melihat daftar aset yang sedang digunakan dan ekspor ke CSV.</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex items-center space-x-2 mb-4">
            <Select onValueChange={setSelectedDept} value={selectedDept}>
              <SelectTrigger className="w-[280px]"><SelectValue placeholder="Pilih Departemen..." /></SelectTrigger>
              <SelectContent>
                {departments.map(dept => (<SelectItem key={dept.id} value={dept.id.toString()}>{dept.name}</SelectItem>))}
              </SelectContent>
            </Select>
            <Button onClick={handleGenerateDeptReport} disabled={isDeptLoading}>
              {isDeptLoading ? 'Memuat...' : 'Tampilkan Laporan'}
            </Button>
            {deptReportData.length > 0 && (
              <Button variant="outline" onClick={handleExportCSV}>Ekspor ke CSV</Button>
            )}
          </div>
          {deptReportData.length > 0 && (
            <div className="border rounded-md">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Nama Karyawan</TableHead>
                    <TableHead>NIK</TableHead>
                    <TableHead>Nama Aset</TableHead>
                    <TableHead>Tag Aset</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {deptReportData.map((row, index) => (
                    <TableRow key={index}>
                      <TableCell>{row.employee_name}</TableCell>
                      <TableCell>{row.employee_nik}</TableCell>
                      <TableCell className="font-medium">{row.asset_name}</TableCell>
                      <TableCell className="font-mono">{row.asset_tag}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </CardContent>
      </Card>

       <Card>
        <CardHeader>
          <CardTitle>Laporan Aset per Karyawan</CardTitle>
          <CardDescription>Pilih karyawan untuk melihat daftar aset yang sedang dipegang.</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex items-center space-x-2 mb-4">
            <Select onValueChange={setSelectedEmployee} value={selectedEmployee}>
              <SelectTrigger className="w-[280px]"><SelectValue placeholder="Pilih Karyawan..." /></SelectTrigger>
              <SelectContent>
                {employees.map(emp => (<SelectItem key={emp.id} value={emp.id.toString()}>{emp.name}</SelectItem>))}
              </SelectContent>
            </Select>
            <Button onClick={handleGenerateEmployeeReport} disabled={isEmployeeLoading}>
              {isEmployeeLoading ? 'Memuat...' : 'Tampilkan Laporan'}
            </Button>
          </div>
          {/* Tambahkan kondisi isLoading di sini */}
          {isEmployeeLoading && <p className="text-sm text-muted-foreground">Loading report data...</p>}
          {!isEmployeeLoading && employeeReportData.length > 0 && (
            <div className="border rounded-md">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Nama Aset</TableHead>
                    <TableHead>Tag Aset</TableHead>
                    <TableHead>Tipe Aset</TableHead>
                    <TableHead>Tanggal Diberikan</TableHead>
                    </TableRow>
                  </TableHeader>
                <TableBody>
                  {employeeReportData.map((row, index) => (
                    <TableRow key={index}>
                      <TableCell className="font-medium">{row.asset_name}</TableCell>
                      <TableCell className="font-mono">{row.asset_tag}</TableCell>
                      <TableCell>{row.asset_type}</TableCell>
                      <TableCell>{new Date(row.assigned_at).toLocaleDateString('id-ID')}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </CardContent>
      </Card>

      {/* --- CARD BARU UNTUK LAPORAN TIKET PER TIPE ASET --- */}
      <Card>
        <CardHeader>
          <CardTitle>Laporan Tiket per Tipe Aset</CardTitle>
          <CardDescription>Menampilkan jumlah tiket masalah berdasarkan tipe aset.</CardDescription>
        </CardHeader>
        <CardContent className="h-[350px]">
          <ResponsiveContainer width="100%" height="100%">
            <BarChart data={ticketReport} layout="vertical">
              <XAxis type="number" />
              <YAxis type="category" dataKey="asset_type" width={120} />
              <Tooltip cursor={{ fill: 'rgba(241, 245, 249, 0.5)' }} />
              <Bar dataKey="ticket_count" fill="#8884d8" barSize={30} />
            </BarChart>
          </ResponsiveContainer>
        </CardContent>
      </Card>

    </div>
  );
}