// File: src/hooks/useManagementLogic.ts
import { useState, useEffect, useCallback } from 'react';
import apiClient from '../src/services/api';
import toast from 'react-hot-toast';

interface ManagedItem {
  id: number;
  name: string;
}

export function useManagementLogic(title: string, apiEndpoint: string) {
  const [data, setData] = useState<ManagedItem[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const [isConfirmOpen, setIsConfirmOpen] = useState(false);
  const [currentItem, setCurrentItem] = useState<ManagedItem | null>(null);
  const [itemName, setItemName] = useState("");
  const [itemToDelete, setItemToDelete] = useState<ManagedItem | null>(null);

  const fetchData = useCallback(() => {
    setIsLoading(true);
    apiClient.get(apiEndpoint)
      .then(res => setData(res.data))
      .catch(() => toast.error(`Gagal memuat data ${title}.`))
      .finally(() => setIsLoading(false));
  }, [apiEndpoint, title]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const handleOpenDialog = (item: ManagedItem | null) => {
    setCurrentItem(item);
    setItemName(item ? item.name : "");
    setIsDialogOpen(true);
  };

  const handleCloseDialog = () => {
    setIsDialogOpen(false);
    setCurrentItem(null);
    setItemName("");
  };

  const handleSave = () => {
    const promise = currentItem
      ? apiClient.put(`${apiEndpoint}/${currentItem.id}`, { name: itemName })
      : apiClient.post(apiEndpoint, { name: itemName });

    toast.promise(promise, {
      loading: `Menyimpan ${title}...`,
      success: () => {
        handleCloseDialog();
        fetchData();
        return `${title} berhasil disimpan!`;
      },
      error: (err) => err.response?.data?.error || `Gagal menyimpan ${title}.`,
    });
  };

  const handleDeleteClick = (item: ManagedItem) => {
    setItemToDelete(item);
    setIsConfirmOpen(true);
  };
  
  const handleConfirmDelete = () => {
    if (!itemToDelete) return;

    const promise = apiClient.delete(`${apiEndpoint}/${itemToDelete.id}`);
    toast.promise(promise, {
      loading: 'Menghapus...',
      success: () => {
        fetchData();
        return `${title} berhasil dihapus!`;
      },
      error: (err) => err.response?.data?.error || `Gagal menghapus ${title}.`,
    });
    
    setIsConfirmOpen(false);
    setItemToDelete(null);
  };

  return {
    data,
    isLoading,
    isDialogOpen,
    isConfirmOpen,
    setIsConfirmOpen,
    setIsDialogOpen,
    currentItem,
    itemName,
    setItemName,
    itemToDelete,
    handleOpenDialog,
    handleCloseDialog,
    handleSave,
    handleDeleteClick,
    handleConfirmDelete,
  };
}