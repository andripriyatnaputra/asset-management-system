import { useEffect, useState } from 'react'
import apiClient from '@/services/api'
import { toast } from 'sonner'
import type { Budget } from '@/types'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Progress } from '@/components/ui/progress'
import { Input } from '@/components/ui/input'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import BudgetFormModal from '@/components/BudgetFormModal'
import BudgetReportModal from '@/components/BudgetReportModal'
import BudgetTransactionsModal from '@/components/BudgetTransactionModal'

// =============================
// Format mata uang
// =============================
const formatCurrency = (value: number) =>
  new Intl.NumberFormat('id-ID', {
    style: 'currency',
    currency: 'IDR',
    maximumFractionDigits: 0
  }).format(value)

// =============================
// Utility GLOBAL anti-null
// =============================
function toArray<T>(value: any): T[] {
  if (!value) return []
  if (Array.isArray(value)) return value

  if (value && typeof value === 'object') {
    if (Array.isArray(value.data)) return value.data
    if (value.data === null) return []
    if (Array.isArray(value.dashboard)) return value.dashboard
    if (value.dashboard === null) return []
  }

  return []
}

// =============================
// Main Component
// =============================
export default function BudgetsPage() {
  const [budgets, setBudgets] = useState<Budget[]>([])
  const [dashboard, setDashboard] = useState<any[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [activeTab, setActiveTab] = useState('list')

  // modal state
  const [isModalOpen, setIsModalOpen] = useState(false)
  const [editingBudget, setEditingBudget] = useState<Budget | null>(null)
  const [showReport, setShowReport] = useState(false)
  const [selectedBudgetId, setSelectedBudgetId] = useState<number | null>(null)
  const [showTransactions, setShowTransactions] = useState(false)

  // simple search
  const [search, setSearch] = useState('')

  // =============================
  // Fetch Budgets
  // =============================
  const fetchBudgets = () => {
    setIsLoading(true)
    apiClient
      .get('/budgets')
      .then(res => {
        const arr = toArray<Budget>(res.data)
        setBudgets(arr)
      })
      .catch(() => toast.error('Gagal memuat data anggaran.'))
      .finally(() => setIsLoading(false))
  }

  // =============================
  // Fetch Dashboard
  // =============================
  const fetchDashboard = () => {
    apiClient
      .get('/budgets/dashboard')
      .then(res => {
        const arr = toArray(res.data)
        setDashboard(arr)
      })
      .catch(() => toast.error('Gagal memuat dashboard.'))
  }

  useEffect(() => {
    fetchBudgets()
    fetchDashboard()
  }, [])

  const handleOpenModal = (budget: Budget | null) => {
    setEditingBudget(budget)
    setIsModalOpen(true)
  }

  const handleCloseModal = () => {
    setIsModalOpen(false)
    setEditingBudget(null)
  }

  const handleSuccess = () => {
    handleCloseModal()
    fetchBudgets()
    fetchDashboard()
  }

  // =============================
  // Filter Results
  // =============================
  const filtered = budgets.filter(
    b => !search.trim() || b.name.toLowerCase().includes(search.trim().toLowerCase())
  )

  return (
    <div className="container mx-auto py-8 space-y-6">
      {/* Header */}
      <div className="flex flex-wrap items-center gap-3 justify-between">
        <h1 className="text-3xl font-bold">Manajemen Anggaran</h1>

        <div className="flex items-center gap-2">
          <Input
            placeholder="Cari anggaran…"
            value={search}
            onChange={e => setSearch(e.target.value)}
            className="w-60"
          />
          <Button onClick={() => handleOpenModal(null)}>+ Tambah Anggaran</Button>

          <Button variant="secondary" onClick={() => setShowReport(true)}>
            Lihat Laporan
          </Button>
        </div>
      </div>

      {/* Tabs */}
      <Tabs defaultValue="list" value={activeTab} onValueChange={setActiveTab}>
        <TabsList>
          <TabsTrigger value="list">Daftar Anggaran</TabsTrigger>
          <TabsTrigger value="dashboard">Dashboard Overview</TabsTrigger>
        </TabsList>

        {/* =========================
            TAB LIST
        ========================== */}
        <TabsContent value="list" className="mt-6">
          {isLoading ? (
            <p className="text-muted-foreground">Memuat data…</p>
          ) : filtered.length === 0 ? (
            <p className="text-muted-foreground">Tidak ada anggaran.</p>
          ) : (
            <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
              {filtered.map(budget => {
                const spent = Number(budget.spent_amount || 0)
                const total = Number(budget.total_amount || 0)
                const pct = total > 0 ? (spent / total) * 100 : 0
                const remaining = total - spent

                return (
                  <Card key={budget.id} className="flex flex-col">
                    <CardHeader>
                      <CardTitle>{budget.name}</CardTitle>
                      <CardDescription>
                        {new Date(budget.start_date).toLocaleDateString('id-ID')} –{' '}
                        {new Date(budget.end_date).toLocaleDateString('id-ID')}
                      </CardDescription>
                    </CardHeader>

                    <CardContent className="space-y-4 flex-grow">
                      <div>
                        <div className="flex justify-between text-sm mb-1">
                          <span className="text-muted-foreground">Terpakai</span>
                          <span>{formatCurrency(spent)}</span>
                        </div>

                        <Progress value={pct} />

                        <div className="flex justify-between text-sm mt-1">
                          <span className="font-semibold">
                            Sisa: {formatCurrency(remaining)}
                          </span>
                          <span className="text-muted-foreground">
                            {formatCurrency(total)}
                          </span>
                        </div>
                      </div>

                      <div className="flex gap-2">
                        <Button
                          variant="outline"
                          size="sm"
                          className="w-full"
                          onClick={() => handleOpenModal(budget)}
                        >
                          Edit
                        </Button>

                        <Button
                          variant="secondary"
                          size="sm"
                          className="w-full"
                          onClick={() => {
                            setSelectedBudgetId(budget.id)
                            setShowTransactions(true)
                          }}
                        >
                          Lihat Transaksi
                        </Button>
                      </div>
                    </CardContent>
                  </Card>
                )
              })}
            </div>
          )}
        </TabsContent>

        {/* =========================
            TAB DASHBOARD
        ========================== */}
        <TabsContent value="dashboard" className="mt-6">
          {dashboard.length === 0 ? (
            <p className="text-muted-foreground">Tidak ada data dashboard.</p>
          ) : (
            <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
              {dashboard.map((item: any) => {
                const pct = item.realization_percent || 0
                const color =
                  item.status === 'overspend'
                    ? 'bg-red-500'
                    : item.status === 'warning'
                    ? 'bg-amber-500'
                    : 'bg-green-500'

                return (
                  <Card key={item.budget_id} className="flex flex-col">
                    <CardHeader>
                      <CardTitle className="flex justify-between">
                        <span>{item.budget_name}</span>
                        <span
                          className={`text-xs font-semibold uppercase ${color} text-white px-2 py-0.5 rounded`}
                        >
                          {item.status}
                        </span>
                      </CardTitle>

                      <CardDescription>Kategori: {item.category || '-'}</CardDescription>
                    </CardHeader>

                    <CardContent className="space-y-3 flex-grow">
                      <div className="flex justify-between text-sm">
                        <span>Total Anggaran:</span>
                        <span className="font-semibold">
                          {formatCurrency(item.total_amount)}
                        </span>
                      </div>

                      <div className="flex justify-between text-sm">
                        <span>Realisasi:</span>
                        <span className="font-semibold text-green-700">
                          {formatCurrency(item.realized_amount)}
                        </span>
                      </div>

                      <div className="flex justify-between text-sm">
                        <span>Sisa:</span>
                        <span className="font-semibold">
                          {formatCurrency(item.remaining_amount)}
                        </span>
                      </div>

                      <Progress value={pct} />
                      <p className="text-sm text-muted-foreground text-right">
                        {pct}% Terealisasi
                      </p>
                    </CardContent>
                  </Card>
                )
              })}
            </div>
          )}
        </TabsContent>
      </Tabs>

      {/* Modals */}
      <BudgetFormModal
        isOpen={isModalOpen}
        onClose={handleCloseModal}
        onSuccess={handleSuccess}
        budget={editingBudget}
      />

      <BudgetReportModal isOpen={showReport} onClose={() => setShowReport(false)} />

      <BudgetTransactionsModal
        isOpen={showTransactions}
        onClose={() => setShowTransactions(false)}
        budgetId={selectedBudgetId}
      />
    </div>
  )
}
