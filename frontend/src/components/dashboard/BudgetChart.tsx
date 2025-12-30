import { useEffect, useState } from "react";
import { getBudgetDashboard } from "@/services/budgetService";

export default function BudgetChart() {
  const [data, setData] = useState<any>(null);

  useEffect(() => {
    getBudgetDashboard().then(setData).catch(console.error);
  }, []);

  if (!data) return <div className="p-4 shadow rounded bg-white">Loading budgets...</div>;

  return (
    <div className="p-4 shadow rounded bg-white">
      <h3 className="text-lg font-semibold mb-2">Budget Utilization</h3>
      <p className="text-2xl font-bold text-blue-600">
        {data.utilization_percent ?? 0}%
      </p>
      <p className="text-sm text-gray-500">Used of total</p>
    </div>
  );
}
