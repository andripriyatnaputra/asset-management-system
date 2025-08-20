// File: src/components/ProtectedRoute.tsx
import { Navigate, Outlet } from 'react-router-dom';

const ProtectedRoute = () => {
  const token = localStorage.getItem('authToken');

  // Jika token ada, izinkan akses ke halaman yang diminta (direpresentasikan oleh <Outlet />).
  // Jika tidak, lempar pengguna ke halaman login.
  return token ? <Outlet /> : <Navigate to="/login" replace />;
};

export default ProtectedRoute;