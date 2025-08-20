// File: src/pages/EmployeesPage.tsx
import { useEmployeeLogic } from '../hooks/useEmployeeLogic'; // Perbaikan path import
import { useEffect, useState } from 'react';
import apiClient from '../src/services/api'; // Perbaikan path import

import { Button } from "../src/components/ui/button";
import { Input } from '../src/components/ui/input';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "../src/components/ui/table";
import { Badge } from "../src/components/ui/badge"; // Impor Badge yang terlewat
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../src/components/ui/select';
import { Pagination } from '../src/components/ui/pagination'; // Perbaikan path import
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "../src/components/ui/alert-dialog"; // Perbaikan path import
import EmployeeFormModal from '../src/components/EmployeeFormModal'; // Perbaikan path import
import type { Department } from '../src/types'; // Perbaikan path import

export default function EmployeesPage() {
  const [departments, setDepartments] = useState<Department[]>([]);

  const {
    employees, pagination, isLoading, isModalOpen, editingEmployee, isConfirmOpen, setIsConfirmOpen,
    searchTerm, setSearchTerm, selectedDept, setSelectedDept,
    handleOpenModal, handleCloseModal, handleSuccess, handleDeleteClick, handleConfirmDelete, handleSort,
    setCurrentPage,
  } = useEmployeeLogic();

  useEffect(() => {
    apiClient.get('/departments').then(res => setDepartments(res.data));
  }, [])

  return (
    <div className="container mx-auto py-8">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-3xl font-bold">Manajemen Karyawan</h1>
        <Button onClick={() => handleOpenModal(null)}>+ Tambah Karyawan</Button>
      </div>

      <div className="bg-white p-4 mb-6 border rounded-lg flex flex-wrap items-center justify-between gap-4">
        <div className="flex-grow sm:flex-grow-0">
          <Input
            placeholder="Cari berdasarkan nama..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="w-full sm:w-64"
          />
        </div>
        <div className="flex items-center space-x-2">
          <Select value={selectedDept} onValueChange={setSelectedDept}>
            <SelectTrigger className="w-[180px]"><SelectValue placeholder="Semua Departemen" /></SelectTrigger>
            <SelectContent>
              <SelectItem value="all">Semua Departemen</SelectItem>
              {departments.map(dept => (
                <SelectItem key={dept.id} value={dept.id.toString()}>{dept.name}</SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      <div className="bg-white p-4 border rounded-lg">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead onClick={() => handleSort('employee_nik')} className="cursor-pointer hover:bg-gray-100">NIK</TableHead>
              <TableHead onClick={() => handleSort('name')} className="cursor-pointer hover:bg-gray-100">Nama</TableHead>
              <TableHead onClick={() => handleSort('email')} className="cursor-pointer hover:bg-gray-100">Email</TableHead>
              <TableHead onClick={() => handleSort('department_name')} className="cursor-pointer hover:bg-gray-100">Departemen</TableHead>
              <TableHead>Role</TableHead>
              <TableHead className="text-right">Aksi</TableHead>
            </TableRow>
          </TableHeader>
          {/* --- BAGIAN YANG HILANG & DIPERBAIKI ADA DI SINI --- */}
          <TableBody>
            {isLoading ? (
              <TableRow><TableCell colSpan={6} className="text-center h-24">Loading...</TableCell></TableRow>
            ) : (
              employees.map((employee) => (
                <TableRow key={employee.id}>
                  <TableCell>{employee.employee_nik}</TableCell>
                  <TableCell className="font-medium">{employee.name}</TableCell>
                  <TableCell>{employee.email}</TableCell>
                  <TableCell>{employee.department_name || '-'}</TableCell>
                  <TableCell>
                    <Badge variant={employee.role === 'super_admin' ? 'destructive' : 'default'}>
                      {employee.role}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-right space-x-2">
                    <Button variant="outline" size="sm" onClick={() => handleOpenModal(employee)}>Edit</Button>
                    <Button variant="destructive" size="sm" onClick={() => handleDeleteClick(employee)}>Delete</Button>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>

      {pagination && <Pagination currentPage={pagination.current_page} totalPages={pagination.total_pages} onPageChange={setCurrentPage} />}

      <EmployeeFormModal
        isOpen={isModalOpen}
        onClose={handleCloseModal}
        onSuccess={handleSuccess}
        employee={editingEmployee}
      />

      <AlertDialog open={isConfirmOpen} onOpenChange={setIsConfirmOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Apakah Anda Yakin?</AlertDialogTitle>
            <AlertDialogDescription>
              Tindakan ini akan menghapus karyawan "{editingEmployee?.name}".
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Batal</AlertDialogCancel>
            <AlertDialogAction onClick={handleConfirmDelete}>Ya, Hapus</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}