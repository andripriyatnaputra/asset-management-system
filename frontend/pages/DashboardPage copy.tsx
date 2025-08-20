// File: src/pages/DashboardPage.tsx
import React, { useEffect, useState, Suspense } from 'react'; // <-- Tambahkan Suspense
import apiClient from '../src/services/api';
import toast from 'react-hot-toast';

import { Card, CardContent, CardHeader, CardTitle } from "../src/components/ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "../src/components/ui/table";

// Gunakan dynamic import dengan React.lazy
const AssetTypeChart = React.lazy(() => import('../src/components/AssetTypeChart'));
const EmployeeDeptChart = React.lazy(() => import('../src/components/EmployeeDeptChart'));


interface StatCard { title: string; value: number; }
interface RecentActivity { asset_name: string; employee_name: string; assigned_at: string; }
interface ChartData { name: string; value: number; }
interface DashboardStats {
  stat_cards: StatCard[];
  recent_activity: RecentActivity[];
  assets_by_type: ChartData[];
  employees_by_dept: ChartData[];
}

export default function DashboardPage() {
  const [stats, setStats] = useState<DashboardStats | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    const fetchStats = async () => {
      try {
        const response = await apiClient.get('/dashboard/stats');
        setStats(response.data);
        console.log("Data Dashboard Diterima:", response.data); // Untuk debugging
      } catch (err) {
        toast.error('Gagal memuat data dashboard.');
      } finally {
        setIsLoading(false);
      }
    };
    fetchStats();
  }, []);

  if (isLoading) {
    return <div className="p-8">Loading dashboard...</div>;
  }

  return (
    <div className="container mx-auto py-8 space-y-8">
      <div>
        <h1 className="text-3xl font-bold">Dashboard Overview</h1>
      </div>

      <div className="grid gap-6 md:grid-cols-2 xl:grid-cols-4">
        {stats?.stat_cards.map((card, index) => (
          <Card key={index}>
            <CardHeader>
              <CardTitle className="text-sm font-medium text-muted-foreground">{card.title}</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-3xl font-bold">{card.value}</div>
            </CardContent>
          </Card>
        ))}
      </div>

     <div className="grid gap-6 md:grid-cols-2">
        <Card>
          <CardHeader><CardTitle>Aset per Tipe</CardTitle></CardHeader>
          <CardContent className="h-[72px]">
            {/* Bungkus chart dengan Suspense */}
            <Suspense fallback={<div>Loading chart...</div>}>
              {stats && <AssetTypeChart data={stats.assets_by_type} />}
            </Suspense>
          </CardContent>
        </Card>
        <Card>
          <CardHeader><CardTitle>Karyawan per Departemen</CardTitle></CardHeader>
          <CardContent className="h-[72px]">
             {/* Bungkus chart dengan Suspense */}
            <Suspense fallback={<div>Loading chart...</div>}>
              {stats && <EmployeeDeptChart data={stats.employees_by_dept} />}
            </Suspense>
          </CardContent>
        </Card>
      </div>
      
      <div>
        <h2 className="text-2xl font-bold mb-4">Aktivitas Terakhir</h2>
        <div className="bg-white p-4 border rounded-lg">
          <Table>
            <TableHeader><TableRow><TableHead>Nama Aset</TableHead><TableHead>Diberikan Kepada</TableHead><TableHead>Tanggal</TableHead></TableRow></TableHeader>
            <TableBody>
              {stats?.recent_activity.map((activity, index) => (
                <TableRow key={index}>
                  <TableCell className="font-medium">{activity.asset_name}</TableCell>
                  <TableCell>{activity.employee_name}</TableCell>
                  <TableCell>{new Date(activity.assigned_at).toLocaleDateString('id-ID')}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      </div>
    </div>
  );
}