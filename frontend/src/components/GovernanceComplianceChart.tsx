import { useState } from "react"
import { PieChart, Pie, Cell, Tooltip, Legend, ResponsiveContainer } from "recharts"
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { fetchComplianceDetails } from "@/services/complianceService"
import { toast } from "sonner"

export default function GovernanceComplianceChart({ data }: { data: any[] }) {
  const [selectedCategory, setSelectedCategory] = useState<string | null>(null)
  const [details, setDetails] = useState<any[]>([])
  const [open, setOpen] = useState(false)

  const COLORS: Record<string, string> = {
    Compliant: "#16a34a",
    "Partially Compliant": "#facc15",
    "Non-Compliant": "#ef4444",
    Pending: "#6b7280",
  }

  const total = data.reduce((sum, d) => sum + (d.value || 0), 0)
  const enriched = data.map((d) => ({
    ...d,
    percent: total > 0 ? ((d.value / total) * 100).toFixed(1) : "0.0",
  }))

  const handleClick = async (category: string) => {
    try {
      const list = await fetchComplianceDetails(category)
      if (!list.length) toast.info(`Tidak ada data untuk kategori ${category}.`)
      setDetails(list)
      setSelectedCategory(category)
      setOpen(true)
    } catch {
      toast.error("Gagal memuat detail kepatuhan.")
    }
  }

  return (
    <>
      <ResponsiveContainer width="100%" height={250}>
        <PieChart>
          <Pie
            data={enriched}
            cx="50%"
            cy="50%"
            outerRadius={100}
            dataKey="value"
            nameKey="name"
            labelLine={false}
            label={({ name, percent }) => `${name} (${percent}%)`}
            onClick={(entry) => handleClick(entry.name)}
          >
            {enriched.map((entry, index) => (
              <Cell
                key={`cell-${index}`}
                fill={COLORS[entry.name] || "#94a3b8"}
                className="cursor-pointer"
              />
            ))}
          </Pie>
          <Tooltip
            formatter={(v: number, n: string) => [`${v} departemen`, n]}
            contentStyle={{
              backgroundColor: "hsl(var(--popover))",
              border: "1px solid hsl(var(--border))",
              borderRadius: "0.5rem",
              color: "hsl(var(--popover-foreground))",
            }}
          />
          <Legend />
        </PieChart>
      </ResponsiveContainer>

      {/* 🔹 Modal detail */}
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="max-w-3xl">
          <DialogHeader>
            <DialogTitle>Detail {selectedCategory}</DialogTitle>
          </DialogHeader>
          <div className="max-h-[400px] overflow-auto mt-2">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Departemen</TableHead>
                  <TableHead>Indeks Kepatuhan</TableHead>
                  <TableHead>Audit Terakhir</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {details.map((r, i) => (
                  <TableRow key={i}>
                    <TableCell>{r.department_name}</TableCell>
                    <TableCell>{r.total_compliance_index?.toFixed(1) ?? "-"}</TableCell>
                    <TableCell>
                      {r.last_audit_date
                        ? new Date(r.last_audit_date).toLocaleDateString("id-ID")
                        : "-"}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        </DialogContent>
      </Dialog>
    </>
  )
}
