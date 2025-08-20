// File: src/pages/BudgetsPage.tsx
import { useEffect, useState } from 'react';
import apiClient from '../src/services/api';
import toast from 'react-hot-toast';
import type { Budget } from '../src/types';

import { Button } from "../src/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "../src/components/ui/card";
import { Progress } from "../src/components/ui/progress";
import BudgetFormModal from '../src/components/BudgetFormModal';
// We will create BudgetFormModal later

const formatCurrency = (value: number) => new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR' }).format(value);

export default function BudgetsPage() {
  const [budgets, setBudgets] = useState<Budget[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  // State untuk modal
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingBudget, setEditingBudget] = useState<Budget | null>(null);

  const fetchBudgets = () => {
    setIsLoading(true);
    apiClient.get('/budgets')
      .then(res => setBudgets(res.data))
      .catch(() => toast.error('Gagal memuat data anggaran.'))
      .finally(() => setIsLoading(false));
  };

  useEffect(() => {
    fetchBudgets();
  }, []);

  const handleOpenModal = (budget: Budget | null) => {
    setEditingBudget(budget);
    setIsModalOpen(true);
  };

  const handleCloseModal = () => {
    setIsModalOpen(false);
    setEditingBudget(null);
  };

  const handleSuccess = () => {
    handleCloseModal();
    fetchBudgets();
  };

  return (
    <div className="container mx-auto py-8">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-3xl font-bold">Manajemen Anggaran</h1>
        <Button onClick={() => handleOpenModal(null)}>+ Tambah Anggaran</Button>
      </div>

      {isLoading ? <p>Loading...</p> : (
        <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
          {budgets && budgets.map(budget => {
            const spentPercentage = budget.total_amount > 0 ? (budget.spent_amount / budget.total_amount) * 100 : 0;
            const remaining = budget.total_amount - budget.spent_amount;
            
            return (
              <Card key={budget.id} className="flex flex-col">
                <CardHeader>
                  <CardTitle>{budget.name}</CardTitle>
                  <CardDescription>
                    {new Date(budget.start_date).toLocaleDateString('id-ID')} - {new Date(budget.end_date).toLocaleDateString('id-ID')}
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-4 flex-grow flex flex-col justify-end">
                  <div>
                    <div className="flex justify-between text-sm mb-1">
                      <span className="text-muted-foreground">Terpakai</span>
                      <span>{formatCurrency(budget.spent_amount)}</span>
                    </div>
                    <Progress value={spentPercentage} />
                    <div className="flex justify-between text-sm mt-1">
                      <span className="font-semibold">Sisa: {formatCurrency(remaining)}</span>
                      <span className="text-muted-foreground">{formatCurrency(budget.total_amount)}</span>
                    </div>
                  </div>
                  <Button variant="outline" size="sm" className="w-full mt-4" onClick={() => handleOpenModal(budget)}>
                    Edit
                  </Button>
                </CardContent>
              </Card>
            )
          })}
        </div>
      )}

      <BudgetFormModal
        isOpen={isModalOpen}
        onClose={handleCloseModal}
        onSuccess={handleSuccess}
        budget={editingBudget}
      />
    </div>
  );
}