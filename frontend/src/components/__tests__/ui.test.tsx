import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { Button } from '../ui/Button';
import { Input } from '../ui/Input';
import { Pagination } from '../ui/Pagination';

// ─── Button ──────────────────────────────────────────────────────────────────

describe('Button', () => {
  it('children を表示する', () => {
    render(<Button>保存</Button>);
    expect(screen.getByRole('button', { name: '保存' })).toBeInTheDocument();
  });

  it('loading=true のときスピナーを表示しクリック不可にする', () => {
    render(<Button loading>送信中</Button>);
    const btn = screen.getByRole('button');
    expect(btn).toBeDisabled();
  });

  it('disabled=true のとき disabled 属性を持つ', () => {
    render(<Button disabled>無効</Button>);
    expect(screen.getByRole('button')).toBeDisabled();
  });

  it('onClick ハンドラーが呼ばれる', () => {
    const onClick = jest.fn();
    render(<Button onClick={onClick}>クリック</Button>);
    fireEvent.click(screen.getByRole('button'));
    expect(onClick).toHaveBeenCalledTimes(1);
  });

  it('disabled 時に onClick が呼ばれない', () => {
    const onClick = jest.fn();
    render(<Button disabled onClick={onClick}>無効</Button>);
    fireEvent.click(screen.getByRole('button'));
    expect(onClick).not.toHaveBeenCalled();
  });
});

// ─── Input ───────────────────────────────────────────────────────────────────

describe('Input', () => {
  it('label を表示する', () => {
    render(<Input label="メールアドレス" />);
    expect(screen.getByLabelText('メールアドレス')).toBeInTheDocument();
  });

  it('error メッセージを表示する', () => {
    render(<Input error="必須項目です" />);
    expect(screen.getByText('必須項目です')).toBeInTheDocument();
  });

  it('error があるとき input にエラー用ボーダーを適用する', () => {
    render(<Input error="エラー" />);
    expect(screen.getByRole('textbox')).toHaveClass('border-red-400');
  });

  it('error がないとき通常ボーダーを適用する', () => {
    render(<Input />);
    expect(screen.getByRole('textbox')).toHaveClass('border-gray-300');
  });
});

// ─── Pagination ───────────────────────────────────────────────────────────────

describe('Pagination', () => {
  const defaultProps = {
    page: 2,
    totalPages: 5,
    hasNext: true,
    hasPrev: true,
    onNext: jest.fn(),
    onPrev: jest.fn(),
  };

  it('現在ページ / 総ページ数を表示する', () => {
    render(<Pagination {...defaultProps} />);
    expect(screen.getByText('2 / 5')).toBeInTheDocument();
  });

  it('次へボタンクリックで onNext を呼ぶ', () => {
    const onNext = jest.fn();
    render(<Pagination {...defaultProps} onNext={onNext} />);
    fireEvent.click(screen.getByRole('button', { name: '→' }));
    expect(onNext).toHaveBeenCalledTimes(1);
  });

  it('前へボタンクリックで onPrev を呼ぶ', () => {
    const onPrev = jest.fn();
    render(<Pagination {...defaultProps} onPrev={onPrev} />);
    fireEvent.click(screen.getByRole('button', { name: '←' }));
    expect(onPrev).toHaveBeenCalledTimes(1);
  });

  it('hasNext=false のとき次へボタンが disabled', () => {
    render(<Pagination {...defaultProps} hasNext={false} />);
    expect(screen.getByRole('button', { name: '→' })).toBeDisabled();
  });

  it('hasPrev=false のとき前へボタンが disabled', () => {
    render(<Pagination {...defaultProps} hasPrev={false} />);
    expect(screen.getByRole('button', { name: '←' })).toBeDisabled();
  });
});
