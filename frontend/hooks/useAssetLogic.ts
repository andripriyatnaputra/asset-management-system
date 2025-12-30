import { useMemo } from 'react'
import type { Asset, AssetStatus } from '@/types'

type BadgeVariant = 'default' | 'secondary' | 'destructive' | 'outline'

export function useAssetStatusChip(asset?: Pick<Asset, 'status'> | null) {
  const { variant, label } = useMemo(() => {
    const s = (asset?.status as AssetStatus) || 'in_stock'
    let v: BadgeVariant = 'secondary'
    switch (s) {
      case 'assigned':
        v = 'default'
        break
      case 'maintenance':
        v = 'secondary'
        break
      case 'retired':
        v = 'outline'
        break
      case 'disposed':
        v = 'destructive'
        break
      default:
        v = 'secondary'
    }
    return { variant: v, label: s.replace('_', ' ') }
  }, [asset?.status])

  return { variant, label }
}
