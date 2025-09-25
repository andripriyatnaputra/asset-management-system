// File: src/components/Layout.tsx
import { useEffect, useState, useRef } from 'react';
import { Outlet } from 'react-router-dom'; // <-- Periksa impor ini
import { Toaster, toast } from 'react-hot-toast';

import Sidebar from './Sidebar';
import Header from './Header';

export default function Layout() {
  const [isCollapsed, setIsCollapsed] = useState(false);
  const wsRef = useRef<WebSocket | null>(null);

  useEffect(() => {
    // Hanya jalankan jika koneksi belum ada
    if (!wsRef.current) {
      const token = localStorage.getItem('authToken');
      if (!token) return;

      //const wsUrl = `ws://localhost:8080/api/v1/ws?token=${token}`;
      const wsUrl = `ws://202.50.203.142:8080/api/v1/ws?token=${token}`;
      const ws = new WebSocket(wsUrl);
      wsRef.current = ws; // Simpan koneksi di ref

      ws.onopen = () => console.log('WebSocket Connected');
      
      ws.onmessage = (event) => {
        const notification = JSON.parse(event.data);
        toast.custom((t) => (
        <div
          className={`${
            t.visible ? 'animate-enter' : 'animate-leave'
          } max-w-md w-full bg-white shadow-lg rounded-lg pointer-events-auto flex ring-1 ring-black ring-opacity-5`}
        >
          <div className="flex-1 w-0 p-4">
            <p className="font-medium text-gray-900">Notifikasi Baru</p>
            <p className="mt-1 text-sm text-gray-500">{notification.message}</p>
          </div>
        </div>
      ));
      };

      ws.onclose = () => console.log('WebSocket Disconnected');
    }

    // Cleanup: Tutup koneksi saat komponen di-unmount
    return () => {
      if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
        wsRef.current.close();
      }
      wsRef.current = null;
    };
  }, []);

  return (
    <div className={`grid min-h-screen w-full transition-all duration-300 
      ${isCollapsed ? 'md:grid-cols-[56px_1fr]' : 'md:grid-cols-[220px_1fr] lg:grid-cols-[280px_1fr]'}`}
    >
      <Toaster position="top-right" />
      <Sidebar isCollapsed={isCollapsed} />
      <div className="flex flex-col h-screen">
        <Header setIsCollapsed={setIsCollapsed} />
        <main className="flex-1 overflow-auto p-4 lg:p-6">
          <Outlet />
        </main>
      </div>
    </div>
  );
}