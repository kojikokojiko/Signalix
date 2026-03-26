import Link from 'next/link';
import { useState } from 'react';
import type { User } from '@/types/api';

// ─── Props ────────────────────────────────────────────────────────────────────

export interface NavbarProps {
  user: User | null;
  isLoading?: boolean;
  onLogout: () => void;
}

// ─── Component ───────────────────────────────────────────────────────────────

export function Navbar({ user, isLoading = false, onLogout }: NavbarProps) {
  const [menuOpen, setMenuOpen] = useState(false);

  return (
    <nav className="bg-white border-b border-gray-200 sticky top-0 z-50">
      <div className="max-w-6xl mx-auto px-4 h-14 flex items-center justify-between">
        <div className="flex items-center gap-6">
          <Link href="/" className="font-bold text-lg text-primary-600">
            Signalix
          </Link>
          <Link href="/trending" className="hidden sm:block text-sm text-gray-600 hover:text-gray-900">
            Trending
          </Link>
          {user && (
            <Link href="/feed" className="hidden sm:block text-sm text-gray-600 hover:text-gray-900">
              For You
            </Link>
          )}
        </div>

        <div className="flex items-center gap-3">
          {isLoading ? null : user ? (
            <>
              <Link href="/bookmarks" className="hidden sm:block text-sm text-gray-600 hover:text-gray-900">
                🔖
              </Link>
              <div className="relative">
                <button
                  onClick={() => setMenuOpen(!menuOpen)}
                  aria-label="ユーザーメニュー"
                  className="flex items-center gap-1 text-sm text-gray-700 hover:text-gray-900"
                >
                  <span className="w-8 h-8 rounded-full bg-primary-100 flex items-center justify-center font-medium text-primary-700">
                    {user.display_name.charAt(0).toUpperCase()}
                  </span>
                  <span className="hidden sm:block">{user.display_name}</span>
                </button>

                {menuOpen && (
                  <div className="absolute right-0 mt-1 w-44 bg-white rounded-lg shadow-lg border border-gray-200 py-1 z-50">
                    <Link
                      href="/settings"
                      className="block px-4 py-2 text-sm text-gray-700 hover:bg-gray-50"
                      onClick={() => setMenuOpen(false)}
                    >
                      設定
                    </Link>
                    {user.is_admin && (
                      <Link
                        href="/admin"
                        className="block px-4 py-2 text-sm text-gray-700 hover:bg-gray-50"
                        onClick={() => setMenuOpen(false)}
                      >
                        管理画面
                      </Link>
                    )}
                    <hr className="my-1" />
                    <button
                      onClick={() => { setMenuOpen(false); onLogout(); }}
                      className="block w-full text-left px-4 py-2 text-sm text-red-600 hover:bg-gray-50"
                    >
                      ログアウト
                    </button>
                  </div>
                )}
              </div>
            </>
          ) : (
            <>
              <Link href="/login" className="text-sm text-gray-600 hover:text-gray-900">
                ログイン
              </Link>
              <Link
                href="/signup"
                className="text-sm bg-primary-600 text-white px-3 py-1.5 rounded-lg hover:bg-primary-700"
              >
                登録
              </Link>
            </>
          )}
        </div>
      </div>
    </nav>
  );
}
