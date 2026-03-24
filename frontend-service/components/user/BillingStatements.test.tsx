import { render, screen, waitFor } from '@testing-library/react'
import { vi } from 'vitest'
import { BillingStatements } from './BillingStatements'
import { useAuth } from '@/hooks/use-auth'
import { billingApi } from '@/lib/api/billing'

vi.mock('@/hooks/use-auth', () => ({
  useAuth: vi.fn(),
}))

vi.mock('@/lib/api/billing', () => ({
  billingApi: {
    listStatements: vi.fn(),
  },
}))

describe('BillingStatements Component', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders loading state initially', () => {
    vi.mocked(useAuth).mockReturnValue({
      billingAccount: null,
      isLoading: false,
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

    // Promise that never resolves so it stays loading
    vi.mocked(billingApi.listStatements).mockImplementation(() => new Promise(() => {}))

    render(<BillingStatements />)
    expect(screen.getByText(/loading.../i)).toBeInTheDocument()
  })

  it('BillingStatements renders empty state when no statements are available', async () => {
    vi.mocked(useAuth).mockReturnValue({
      billingAccount: null,
      isLoading: false,
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

    vi.mocked(billingApi.listStatements).mockResolvedValue({
      total: 0,
      page: 1,
      page_size: 20,
      items: [],
    })

    render(<BillingStatements />)
    
    await waitFor(() => {
      expect(screen.queryByText(/loading.../i)).not.toBeInTheDocument()
    })

    expect(screen.getByTestId('billing-statements-panel')).toBeInTheDocument()
    expect(screen.getByTestId('billing-statements-empty')).toBeInTheDocument()
    expect(screen.getByText(/no billing records match the current filters/i)).toBeInTheDocument()
  })

  it('BillingStatements renders table when statements are returned', async () => {
    vi.mocked(useAuth).mockReturnValue({
      billingAccount: null,
      isLoading: false,
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

    vi.mocked(billingApi.listStatements).mockResolvedValue({
      total: 1,
      page: 1,
      page_size: 20,
      items: [
        {
          statement_id: "stmt-123",
          type: 2, // Download
          history_id: 101,
          traffic_bytes: 5242880, // 5MB
          amount_fen: "100", // 1.00
          status: 3, // Completed
          remark: "Youtube Video",
          created_at: "2023-01-01T00:00:00Z"
        }
      ],
    })

    render(<BillingStatements />)
    
    await waitFor(() => {
      expect(screen.queryByText(/loading.../i)).not.toBeInTheDocument()
    })

    expect(screen.getByTestId('billing-statements-panel')).toBeInTheDocument()
    expect(screen.getByTestId('billing-statements-table')).toBeInTheDocument()
    
    expect(screen.getByText('stmt-123')).toBeInTheDocument()
    expect(screen.getByText('History #101')).toBeInTheDocument()
    expect(screen.getByText('Youtube Video')).toBeInTheDocument()
  })

  it('BillingStatementsErrorState renders error state when API fails', async () => {
    vi.mocked(useAuth).mockReturnValue({
      billingAccount: null,
      isLoading: false,
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

    vi.mocked(billingApi.listStatements).mockRejectedValue(new Error('API Error'))

    render(<BillingStatements />)
    
    await waitFor(() => {
      expect(screen.queryByText(/loading.../i)).not.toBeInTheDocument()
    })

    expect(screen.getByTestId('billing-statements-error')).toBeInTheDocument()
    expect(screen.getByText(/failed to load billing statements/i)).toBeInTheDocument()
  })
})
