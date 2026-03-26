import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { Navbar } from '../Navbar';
import type { User } from '@/types/api';

const mockUser: User = {
  id: 'user-1',
  email: 'test@example.com',
  display_name: 'Test User',
  is_admin: false,
  preferred_language: 'ja',
  created_at: '2025-01-01T00:00:00Z',
};

const adminUser: User = { ...mockUser, is_admin: true, display_name: 'Admin' };

describe('Navbar', () => {
  it('未ログイン時にログイン・登録リンクを表示する', () => {
    render(<Navbar user={null} onLogout={jest.fn()} />);
    expect(screen.getByRole('link', { name: 'ログイン' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: '登録' })).toBeInTheDocument();
  });

  it('ログイン時にユーザー名の頭文字を表示する', () => {
    render(<Navbar user={mockUser} onLogout={jest.fn()} />);
    expect(screen.getByText('T')).toBeInTheDocument(); // 'Test User' → 'T'
  });

  it('ログイン時に For You リンクを表示する', () => {
    render(<Navbar user={mockUser} onLogout={jest.fn()} />);
    expect(screen.getByRole('link', { name: 'For You' })).toBeInTheDocument();
  });

  it('未ログイン時に For You リンクを表示しない', () => {
    render(<Navbar user={null} onLogout={jest.fn()} />);
    expect(screen.queryByRole('link', { name: 'For You' })).not.toBeInTheDocument();
  });

  it('ユーザーメニューをクリックで開く', () => {
    render(<Navbar user={mockUser} onLogout={jest.fn()} />);
    fireEvent.click(screen.getByLabelText('ユーザーメニュー'));
    expect(screen.getByText('設定')).toBeInTheDocument();
    expect(screen.getByText('ログアウト')).toBeInTheDocument();
  });

  it('管理者ユーザーには管理画面リンクを表示する', () => {
    render(<Navbar user={adminUser} onLogout={jest.fn()} />);
    fireEvent.click(screen.getByLabelText('ユーザーメニュー'));
    expect(screen.getByText('管理画面')).toBeInTheDocument();
  });

  it('一般ユーザーには管理画面リンクを表示しない', () => {
    render(<Navbar user={mockUser} onLogout={jest.fn()} />);
    fireEvent.click(screen.getByLabelText('ユーザーメニュー'));
    expect(screen.queryByText('管理画面')).not.toBeInTheDocument();
  });

  it('ログアウトボタンクリックで onLogout を呼ぶ', () => {
    const onLogout = jest.fn();
    render(<Navbar user={mockUser} onLogout={onLogout} />);
    fireEvent.click(screen.getByLabelText('ユーザーメニュー'));
    fireEvent.click(screen.getByText('ログアウト'));
    expect(onLogout).toHaveBeenCalledTimes(1);
  });

  it('isLoading=true のときメニューを表示しない', () => {
    render(<Navbar user={null} isLoading={true} onLogout={jest.fn()} />);
    expect(screen.queryByRole('link', { name: 'ログイン' })).not.toBeInTheDocument();
    expect(screen.queryByLabelText('ユーザーメニュー')).not.toBeInTheDocument();
  });
});
