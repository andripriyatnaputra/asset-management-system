import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { RoleGuard } from './RoleGuard'

// Mock AuthContext so tests don't need a real provider
vi.mock('@/context/AuthContext', () => ({
  useAuthContext: vi.fn(),
}))

import { useAuthContext } from '@/context/AuthContext'

function setRole(role: string | null, effectiveRole: string | null = null) {
  vi.mocked(useAuthContext).mockReturnValue({
    role,
    effectiveRole,
    token: null,
    departmentId: null,
    userId: null,
    delegatedRole: null,
    setEffectiveRole: vi.fn(),
    login: vi.fn(),
    logout: vi.fn(),
    isAdmin: role === 'super_admin',
    isManager: role === 'asset_manager',
    isEmployee: role === 'employee',
  })
}

describe('RoleGuard', () => {
  it('merender children jika role sesuai allow', () => {
    setRole('super_admin')
    render(
      <RoleGuard allow={['super_admin', 'asset_manager']}>
        <span>konten admin</span>
      </RoleGuard>
    )
    expect(screen.getByText('konten admin')).toBeInTheDocument()
  })

  it('tidak merender children jika role tidak ada di allow', () => {
    setRole('employee')
    const { container } = render(
      <RoleGuard allow={['super_admin', 'asset_manager']}>
        <span>konten admin</span>
      </RoleGuard>
    )
    expect(container).toBeEmptyDOMElement()
  })

  it('menggunakan effectiveRole saat ada delegasi', () => {
    setRole('employee', 'asset_manager')
    render(
      <RoleGuard allow={['asset_manager']}>
        <span>akses delegasi</span>
      </RoleGuard>
    )
    expect(screen.getByText('akses delegasi')).toBeInTheDocument()
  })

  it('selfOnly=true merender semua role tanpa filter', () => {
    setRole('employee')
    render(
      <RoleGuard selfOnly>
        <span>data milik sendiri</span>
      </RoleGuard>
    )
    expect(screen.getByText('data milik sendiri')).toBeInTheDocument()
  })

  it('merender null jika role kosong dan ada allow list', () => {
    setRole(null)
    const { container } = render(
      <RoleGuard allow={['super_admin']}>
        <span>tidak kelihatan</span>
      </RoleGuard>
    )
    expect(container).toBeEmptyDOMElement()
  })

  it('merender children jika tidak ada allow list (tidak terbatas)', () => {
    setRole('employee')
    render(
      <RoleGuard>
        <span>terbuka semua</span>
      </RoleGuard>
    )
    expect(screen.getByText('terbuka semua')).toBeInTheDocument()
  })
})
