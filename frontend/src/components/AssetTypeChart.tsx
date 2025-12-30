import { useEffect, useState } from 'react'
import { PieChart, Pie, Cell, ResponsiveContainer, Legend, Tooltip } from 'recharts'

interface ChartData {
  name: string
  value: number
}

function useChartPalette() {
  const [colors, setColors] = useState<string[]>([])
  useEffect(() => {
    const styles = getComputedStyle(document.documentElement)
    const vars = ['--chart-1', '--chart-2', '--chart-3', '--chart-4', '--chart-5']
    const palette = vars
      .map(v => styles.getPropertyValue(v).trim())
      .filter(Boolean)
      .map(hsl => `hsl(${hsl})`)
    setColors(palette)
  }, [])
  return colors.length ? colors : ['#3b82f6', '#22c55e', '#f59e0b', '#a855f7', '#ef4444']
}

const CustomTooltip = ({ active, payload }: any) => {
  if (active && payload && payload.length) {
    return (
      <div className="rounded border bg-background p-2 text-sm shadow">
        <p className="font-medium">{payload[0].name}</p>
        <p className="text-muted-foreground">{payload[0].value}</p>
      </div>
    )
  }
  return null
}

export default function AssetTypeChart({ data }: { data: ChartData[] }) {
  const palette = useChartPalette()

  if (!data || data.length === 0) {
    return <div className="flex h-full items-center justify-center text-muted-foreground">Tidak ada data</div>
  }

  return (
    <ResponsiveContainer width="100%" height="100%">
      <PieChart>
        <Pie data={data} dataKey="value" nameKey="name" cx="50%" cy="50%" outerRadius="80%">
          {data.map((_, i) => (
            <Cell key={i} fill={palette[i % palette.length]} />
          ))}
        </Pie>
        <Tooltip content={<CustomTooltip />} />
        <Legend />
      </PieChart>
    </ResponsiveContainer>
  )
}
