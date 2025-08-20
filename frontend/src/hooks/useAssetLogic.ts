// File: src/hooks/useAssetLogic.ts
import { useState, useEffect, useCallback } from 'react';
import apiClient from '../services/api';
import toast from 'react-hot-toast';
import type { Asset, PaginationData } from '../types';

export function useAssetLogic() {
  const [assets, setAssets] = useState<Asset[]>([]);
  const [pagination, setPagination] = useState<PaginationData | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  // State untuk filter & sort
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedType, setSelectedType] = useState('all');
  const [sortBy, setSortBy] = useState('created_at');
  const [sortOrder, setSortOrder] = useState('desc');
  const [currentPage, setCurrentPage] = useState(1);

  const fetchAssets = useCallback((pageToFetch: number) => {
    setIsLoading(true);
    const params = {
      page: pageToFetch.toString(),
      limit: '10',
      q: searchTerm,
      sort_by: sortBy,
      sort_order: sortOrder,
      ...(selectedType !== 'all' && { type_id: selectedType }),
    };
    const queryString = new URLSearchParams(params).toString();

    apiClient.get(`/assets?${queryString}`)
      .then(res => {
        setAssets(res.data.data);
        setPagination(res.data.pagination);
      })
      .catch(() => toast.error('Gagal memuat data aset.'))
      .finally(() => setIsLoading(false));
  }, [searchTerm, selectedType, sortBy, sortOrder]);

  // useEffect untuk memantau perubahan filter
  useEffect(() => {
    const handler = setTimeout(() => {
      if (currentPage !== 1) setCurrentPage(1);
      else fetchAssets(1);
    }, 500);
    return () => clearTimeout(handler);
  }, [searchTerm, selectedType, sortBy, sortOrder]);

  // useEffect untuk memantau perubahan halaman
  useEffect(() => {
    fetchAssets(currentPage);
  }, [currentPage, fetchAssets]);

  const handleSort = (column: string) => {
    const newSortOrder = sortBy === column && sortOrder === 'asc' ? 'desc' : 'asc';
    setSortBy(column);
    setSortOrder(newSortOrder);
  };

  return {
    assets, pagination, isLoading, searchTerm, setSearchTerm,
    selectedType, setSelectedType, setCurrentPage, handleSort,
    fetchAssets, // Ekspor fetchAssets agar bisa dipanggil setelah sukses
  };
}