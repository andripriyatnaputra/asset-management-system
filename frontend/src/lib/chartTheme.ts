// Baca warna dari CSS variables Tailwind (sinkron dengan index.css)
export function getChartColors() {
  const styles = getComputedStyle(document.documentElement)
  // urutkan sesuai token yang kamu set di :root/.dark
  const vars = ['--chart-1', '--chart-2', '--chart-3', '--chart-4', '--chart-5']
  return vars
    .map((v) => styles.getPropertyValue(v).trim())
    .filter(Boolean)
    .map((hsl) => `hsl(${hsl})`) // ubah "H S L" → "hsl(H S L)"
}
