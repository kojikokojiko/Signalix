import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { NavbarContainer } from '../NavbarContainer';

// ─── Mocks ────────────────────────────────────────────────────────────────────

const mockPush = jest.fn();

jest.mock('next/navigation', () => ({
  useRouter: () => ({ push: mockPush }),
}));

const mockLogout = jest.fn();
let mockAuthState = {
  user: null as { id: string; email: string; display_name: string; is_admin: boolean; preferred_language: 'ja' | 'en'; created_at: string } | null,
  status: 'loading' as 'loading' | 'authenticated' | 'unauthenticated',
  logout: mockLogout,
};

jest.mock('@/contexts/AuthContext', () => ({
  useAuth: () => mockAuthState,
}));

jest.mock('@/components/Navbar', () => ({
  Navbar: ({ user, isLoading, onLogout }: {
    user: { display_name: string } | null;
    isLoading: boolean;
    onLogout: () => void;
  }) => (
    <nav data-testid="navbar">
      <span data-testid="user">{user?.display_name ?? 'guest'}</span>
      <span data-testid="loading">{isLoading ? 'loading' : 'ready'}</span>
      <button data-testid="logout-btn" onClick={onLogout}>logout</button>
    </nav>
  ),
}));

// ─── Tests ────────────────────────────────────────────────────────────────────

describe('NavbarContainer', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockAuthState = { user: null, status: 'loading', logout: mockLogout };
  });

  it('loading 中は isLoading=true を Navbar に渡す', () => {
    mockAuthState.status = 'loading';

    render(<NavbarContainer />);

    expect(screen.getByTestId('loading').textContent).toBe('loading');
  });

  it('認証済みはユーザー情報を Navbar に渡す', () => {
    mockAuthState.user = {
      id: 'u1', email: 'alice@example.com', display_name: 'Alice',
      is_admin: false, preferred_language: 'ja', created_at: '2025-01-01T00:00:00Z',
    };
    mockAuthState.status = 'authenticated';

    render(<NavbarContainer />);

    expect(screen.getByTestId('user').textContent).toBe('Alice');
    expect(screen.getByTestId('loading').textContent).toBe('ready');
  });

  it('未認証は user=null を Navbar に渡す', () => {
    mockAuthState.status = 'unauthenticated';

    render(<NavbarContainer />);

    expect(screen.getByTestId('user').textContent).toBe('guest');
  });

  it('ログアウトボタン押下で logout() を呼びトップページへリダイレクト', async () => {
    mockLogout.mockResolvedValueOnce(undefined);
    mockAuthState.status = 'authenticated';
    mockAuthState.user = {
      id: 'u1', email: 'alice@example.com', display_name: 'Alice',
      is_admin: false, preferred_language: 'ja', created_at: '2025-01-01T00:00:00Z',
    };

    render(<NavbarContainer />);

    fireEvent.click(screen.getByTestId('logout-btn'));

    await waitFor(() => expect(mockLogout).toHaveBeenCalledTimes(1));
    await waitFor(() => expect(mockPush).toHaveBeenCalledWith('/'));
  });
});
