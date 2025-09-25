// File: src/hooks/useEmployeeLogic.ts
import { useState, useEffect, useCallback } from 'react';
import apiClient from '@/services/api';
import toast from 'react-hot-toast';
import type { Employee, PaginationData } from '@/types';

export function useEmployeeLogic() {
  const [employees, setEmployees] = useState<Employee[]>([]);
  const [pagination, setPagination] = useState<PaginationData | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  
  // State untuk filter, sort, pagination
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedDept, setSelectedDept] = useState('all');
  const [sortBy, setSortBy] = useState('name');
  const [sortOrder, setSortOrder] = useState('asc');
  const [currentPage, setCurrentPage] = useState(1);
  
  // State untuk modal dan delete
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingEmployee, setEditingEmployee] = useState<Employee | null>(null);
  const [isConfirmOpen, setIsConfirmOpen] = useState(false);

  const fetchEmployees = useCallback(() => {
    setIsLoading(true);
    const params = {
      page: currentPage.toString(), limit: '10', q: searchTerm,
      sort_by: sortBy, sort_order: sortOrder,
      ...(selectedDept !== 'all' && { department_id: selectedDept }),
    };
    const queryString = new URLSearchParams(params).toString();

    apiClient.get(`/employees?${queryString}`)
      .then(res => {
        setEmployees(res.data.data);
        setPagination(res.data.pagination);
      })
      .catch(() => toast.error('Gagal memuat data karyawan.'))
      .finally(() => setIsLoading(false));
  }, [currentPage, searchTerm, selectedDept, sortBy, sortOrder]);

  useEffect(() => {
    const handler = setTimeout(() => {
      if (currentPage !== 1) setCurrentPage(1);
      else fetchEmployees();
    }, 500);
    return () => clearTimeout(handler);
  }, [searchTerm, selectedDept, sortBy, sortOrder]);

  useEffect(() => {
    fetchEmployees();
  }, [currentPage, fetchEmployees]);

  const handleOpenModal = (employee: Employee | null) => {
    setEditingEmployee(employee);
    setIsModalOpen(true);
  };

  const handleCloseModal = () => {
    setIsModalOpen(false);
    setEditingEmployee(null);
  };

  const handleSuccess = () => {
    handleCloseModal();
    fetchEmployees();
  };
  
  const handleDeleteClick = (employee: Employee) => {
    setEditingEmployee(employee);
    setIsConfirmOpen(true);
  };
  
  const handleConfirmDelete = () => {
    if (!editingEmployee) return;
    
    const promise = apiClient.delete(`/employees/${editingEmployee.id}`);
    toast.promise(promise, {
      loading: 'Menghapus karyawan...',
      success: () => {
        fetchEmployees();
        return 'Karyawan berhasil dihapus!';
      },
      error: (err) => err.response?.data?.error || `Gagal menghapus karyawan.`,
    });
    
    setIsConfirmOpen(false);
    setEditingEmployee(null);
  };

  const handleSort = (column: string) => {
    const newSortOrder = sortBy === column && sortOrder === 'asc' ? 'desc' : 'asc';
    setSortBy(column);
    setSortOrder(newSortOrder);
  };

  return {
    employees, pagination, isLoading, isModalOpen, editingEmployee, isConfirmOpen, setIsConfirmOpen,
    searchTerm, setSearchTerm, selectedDept, setSelectedDept,
    handleOpenModal, handleCloseModal, handleSuccess, handleDeleteClick, handleConfirmDelete, handleSort,
    setCurrentPage, fetchEmployees,
  };
}