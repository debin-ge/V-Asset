import { render, screen } from '@testing-library/react'
import { vi } from 'vitest'
import { AccountOverview } from './AccountOverview'
import { useAuth } from '@/hooks/use-auth'

vi.mock('@/hooks/use-auth', () => ({
  useAuth: vi.fn(),
}))

describe('AccountOverview Component', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders loading state initially', () => {
    vi.mocked(useAuth).mockReturnValue({
      isLoading: true,
      billingAccount: null,
      refreshBillingAccount: vi.fn(),
      user: null,
      setUser: vi.fn(),
      isAuthenticated: false,
      login: vi.fn(),
      register: vi.fn(),
      logout: vi.fn(),
      refreshUser: vi.fn(),
      isAuthModalOpen: false,
      openAuthModal: vi.fn(),
      closeAuthModal: vi.fn(),
    })

    render(<AccountOverview />)
    expect(screen.getByText(/loading.../i)).toBeInTheDocument()
  })

  it('AccountOverviewEmptyState renders empty state when no account information is available', () => {
    vi.mocked(useAuth).mockReturnValue({
      isLoading: false,
      billingAccount: null,
      refreshBillingAccount: vi.fn(),
      user: null,
      setUser: vi.fn(),
      isAuthenticated: false,
      login: vi.fn(),
      register: vi.fn(),
      logout: vi.fn(),
      refreshUser: vi.fn(),
      isAuthModalOpen: false,
      openAuthModal: vi.fn(),
      closeAuthModal: vi.fn(),
    })

    render(<AccountOverview />)
    const emptyState = screen.getByTestId('account-overview-empty')
    expect(emptyState).toBeInTheDocument()
    expect(screen.getByText(/no account information available/i)).toBeInTheDocument()
  })

  it('renders populated state with account information', () => {
    const mockBillingAccount = {
      user_id: "u-123",
      currency_code: "CNY",
      available_balance_yuan: "10.00",
      reserved_balance_yuan: "2.00",
      total_recharged_yuan: "12.00",
      total_spent_yuan: "2.00",
      total_traffic_bytes: 1048576,  // 1 MB
      status: 1,                     // Active
      version: 1,
      created_at: "2023-01-01T00:00:00Z",
      updated_at: "2023-01-01T00:00:00Z",
    }

    vi.mocked(useAuth).mockReturnValue({
      isLoading: false,
      billingAccount: mockBillingAccount,
      refreshBillingAccount: vi.fn(),
      user: null,
      setUser: vi.fn(),
      isAuthenticated: false,
      login: vi.fn(),
      register: vi.fn(),
      logout: vi.fn(),
      refreshUser: vi.fn(),
      isAuthModalOpen: false,
      openAuthModal: vi.fn(),
      closeAuthModal: vi.fn(),
    })

    render(<AccountOverview />)
    
    // Check main card
    const card = screen.getByTestId('account-overview-card')
    expect(card).toBeInTheDocument()

    // Check balance value format
    const balanceValue = screen.getByTestId('account-balance-value')
    expect(balanceValue).toBeInTheDocument()
    expect(balanceValue).toHaveTextContent(/10\.00/)

    expect(screen.getByText('Total Spent')).toBeInTheDocument()
    
    expect(screen.getByText('Total Traffic')).toBeInTheDocument()
    expect(screen.getByText('1.0 MB')).toBeInTheDocument()
  })

  it('AccountOverview renders yuan balance directly', () => {
    const mockBillingAccount = {
      user_id: "u-123",
      currency_code: "CNY",
      available_balance_yuan: "1.00",
      reserved_balance_yuan: "1.50",
      total_recharged_yuan: "0",
      total_spent_yuan: "0",
      total_traffic_bytes: 0,
      status: 1,
      version: 1,
      created_at: "2023-01-01T00:00:00Z",
      updated_at: "2023-01-01T00:00:00Z",
    }

    vi.mocked(useAuth).mockReturnValue({
      isLoading: false,
      billingAccount: mockBillingAccount,
      refreshBillingAccount: vi.fn(),
      user: null,
      setUser: vi.fn(),
      isAuthenticated: false,
      login: vi.fn(),
      register: vi.fn(),
      logout: vi.fn(),
      refreshUser: vi.fn(),
      isAuthModalOpen: false,
      openAuthModal: vi.fn(),
      closeAuthModal: vi.fn(),
    })

    render(<AccountOverview />)
    
    // Check main card balance
    const balanceValue = screen.getByTestId('account-balance-value')
    expect(balanceValue).toHaveTextContent(/1\.00/)

    // Check reserved amount.
    expect(screen.getByText(/1\.50/)).toBeInTheDocument()
  })
})
