'use client';

import { useRouter } from 'next/navigation';
import { useAuth } from '@/contexts/AuthContext';
import { Navbar } from '@/components/Navbar';

export function NavbarContainer() {
  const { user, status, logout } = useAuth();
  const router = useRouter();

  const handleLogout = async () => {
    await logout();
    router.push('/');
  };

  return (
    <Navbar
      user={user}
      isLoading={status === 'loading'}
      onLogout={handleLogout}
    />
  );
}
