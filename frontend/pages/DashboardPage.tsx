// File: src/pages/DashboardPage.tsx
import { useEffect, useState } from 'react';
import apiClient from '../src/services/api';
import toast from 'react-hot-toast';

import { Card, CardContent, CardHeader, CardTitle } from "../src/components/ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "../src/components/ui/table";

import AssetTypeChart from '../src/components/AssetTypeChart';
import EmployeeDeptChart from '../src/components/EmployeeDeptChart';

interface StatCard {
  title: string;
  value: number;
}

interface RecentActivity {
  asset_name: string;
  employee_name: string;
  assigned_at: string;
}

interface ChartData {
  name: string;
  value: number;
}

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
        console.log("Data Dashboard Diterima:", response.data);
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

  {/*return (
    <div className="container mx-auto py-8 space-y-8">
      <div>
        <h1 className="text-3xl font-bold">Dashboard Overview</h1>
      </div>

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        {stats?.stat_cards.map((card, index) => (
          <Card key={index}>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">{card.title}</CardTitle>
              {/* Anda bisa menambahkan ikon di sini nanti 
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{card.value}</div>
            </CardContent>
          </Card>
        ))}
      </div>

      <div className="grid gap-6 md:grid-cols-2">
        <Card style={{ minHeight: "350px" }}>
          <CardHeader><CardTitle>Aset per Tipe</CardTitle></CardHeader>
          <CardContent style={{ height: "300px" }}>
            <div style={{ width: '100%', height: '100%' }}>
              {stats && <AssetTypeChart data={stats.assets_by_type} />}
            </div>
          </CardContent>
        </Card>

        <Card style={{ minHeight: "350px" }}>
          <CardHeader><CardTitle>Karyawan per Departemen</CardTitle></CardHeader>
          <CardContent style={{ height: "300px" }}>
            <div style={{ width: '100%', height: '100%' }}>
              {stats && <EmployeeDeptChart data={stats.employees_by_dept} />}
            </div>
          </CardContent>
        </Card>
      </div>

      <div>
        <h2 className="text-2xl font-bold mb-4">Aktivitas Terakhir</h2>
        <div className="bg-white p-4 border rounded-lg">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Nama Aset</TableHead>
                <TableHead>Diberikan Kepada</TableHead>
                <TableHead>Tanggal</TableHead>
              </TableRow>
            </TableHeader>
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
*/}

  return(

  <div>
    <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4 mb-6">
      {stats?.stat_cards.map((card, index) => (
        <Card key={index}>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">{card.title}</CardTitle>
            {/* Anda bisa menambahkan ikon di sini nanti */}
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{card.value}</div>
            </CardContent>
        </Card>
      ))}
    </div>
    

    <div className="grid gap-6 md:grid-cols-2">
        <Card style={{ minHeight: "350px" }}>
          <CardHeader><CardTitle>Aset per Tipe</CardTitle></CardHeader>
          <CardContent style={{ height: "300px" }}>
            <div style={{ width: '100%', height: '100%' }}>
              {stats && <AssetTypeChart data={stats.assets_by_type} />}
            </div>
          </CardContent>
        </Card>

        <Card style={{ minHeight: "350px" }}>
          <CardHeader><CardTitle>Karyawan per Departemen</CardTitle></CardHeader>
          <CardContent style={{ height: "300px" }}>
            <div style={{ width: '100%', height: '100%' }}>
              {stats && <EmployeeDeptChart data={stats.employees_by_dept} />}
            </div>
          </CardContent>
        </Card>
      </div>

      <div>
        <h2 className="text-2xl font-bold mb-4">Aktivitas Terakhir</h2>
        <div className="bg-white p-4 border rounded-lg">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Nama Aset</TableHead>
                <TableHead>Diberikan Kepada</TableHead>
                <TableHead>Tanggal</TableHead>
              </TableRow>
            </TableHeader>
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
  

  
)

}