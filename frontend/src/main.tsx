// File: src/main.tsx
import React from 'react'
import ReactDOM from 'react-dom/client'
import { BrowserRouter, Routes, Route } from 'react-router-dom'
import Modal from 'react-modal'

import ProtectedRoute from './components/ProtectedRoute.tsx';
import Layout from './components/Layout.tsx';
import LoginPage from '../pages/LoginPage.tsx';
import EmployeesPage from '../pages/EmployeesPage.tsx';
import AssetsPage from '../pages/AssetsPage.tsx';
import DepartmentsPage from '../pages/DepartmentsPage.tsx';
import AssetTypesPage from '../pages/AssetTypesPage.tsx';
import DashboardPage from '../pages/DashboardPage.tsx';
import ReportsPage from '../pages/ReportsPage.tsx';
import LicensesPage from '../pages/LicensesPage.tsx';
import TicketsPage from '../pages/TicketsPage.tsx';
import TicketDetailPage from '../pages/TicketDetailPage.tsx';
import MyAssetsPage from '../pages/MyAssetsPage.tsx';
import BudgetsPage from '../pages/BudgetsPage.tsx';
import AuditsPage from '../pages/AuditsPage.tsx';
import AuditSessionPage from '../pages/AuditSessionPage.tsx';

import './index.css'

Modal.setAppElement('#root');

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <BrowserRouter>
      <Routes>
        {/* Rute publik untuk login */}
        <Route path="/login" element={<LoginPage />} />

        {/* Rute-rute yang diproteksi */}
        <Route element={<ProtectedRoute />}>
          <Route element={<Layout />}>
            <Route path="/" element={<DashboardPage />} /> 
            <Route path="/employees" element={<EmployeesPage />} />
            <Route path="/assets" element={<AssetsPage />} />
            <Route path="/departments" element={<DepartmentsPage />} />
            <Route path="/asset-types" element={<AssetTypesPage />} />
            <Route path="/reports" element={<ReportsPage />} />
            <Route path="/licenses" element={<LicensesPage />} />
            <Route path="/tickets" element={<TicketsPage />} />
            <Route path="/tickets/:id" element={<TicketDetailPage />} />
            <Route path="/my-assets" element={<MyAssetsPage />} />
            <Route path="/budgets" element={<BudgetsPage />} />
            <Route path="/budgets" element={<BudgetsPage />} />
            <Route path="/audits" element={<AuditsPage />} />
            <Route path="/audits" element={<AuditsPage />} />
            <Route path="/audits/:id" element={<AuditSessionPage />} />
        </Route>
        </Route>
      </Routes>
    </BrowserRouter>
  </React.StrictMode>,
)