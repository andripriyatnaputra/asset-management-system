// File: src/config/nav.ts
import { Home, Package, Users, Building, Tag, KeyRound, FileText, HelpCircle, Wallet, ClipboardCheck } from 'lucide-react';
import { jwtDecode } from 'jwt-decode';

interface NavItem {
  path: string;
  label: string;
  icon: React.ElementType;
  adminOnly?: boolean;
  employeeOnly?: boolean;
}

interface DecodedToken {
  role: string;
}

// Definisikan semua kemungkinan item navigasi
export const allNavItems: NavItem[] = [
  { path: '/', label: 'Dashboard', icon: Home },
  { path: '/assets', label: 'Manajemen Aset', icon: Package, adminOnly: true },
  { path: '/my-assets', label: 'Aset Saya', icon: Package, employeeOnly: true },
  { path: '/employees', label: 'Employees', icon: Users, adminOnly: true },
  { path: '/departments', label: 'Departments', icon: Building, adminOnly: true },
  { path: '/asset-types', label: 'Asset Types', icon: Tag, adminOnly: true },
  { path: '/reports', label: 'Reports', icon: FileText, adminOnly: true },
  { path: '/licenses', label: 'Licenses', icon: KeyRound, adminOnly: true },
  { path: '/tickets', label: 'Help Desk', icon: HelpCircle },
  { path: '/budgets', label: 'Budgets', icon: Wallet, adminOnly: true },
  { path: '/audits', label: 'Audits', icon: ClipboardCheck, adminOnly: true }
];

// Fungsi untuk mendapatkan item navigasi yang sudah difilter berdasarkan peran
export const getNavItems = (): NavItem[] => {
  const token = localStorage.getItem('authToken');
  let userRole: string | null = null;
  if (token) {
    try {
      const decoded: DecodedToken = jwtDecode(token);
      userRole = decoded.role;
    } catch (error) {
      console.error("Invalid token:", error);
    }
  }

  return allNavItems.filter(item => {
    if (userRole === 'super_admin') {
      return !item.employeeOnly;
    } else {
      return !item.adminOnly;
    }
  });
};