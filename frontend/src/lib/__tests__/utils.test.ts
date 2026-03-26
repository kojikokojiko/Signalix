import { formatRelativeTime, handleApiError } from '../utils';

// ─── formatRelativeTime ───────────────────────────────────────────────────────

describe('formatRelativeTime', () => {
  const now = new Date();

  it('null のとき "不明" を返す', () => {
    expect(formatRelativeTime(null)).toBe('不明');
  });

  it('0分未満のとき "たった今" を返す', () => {
    expect(formatRelativeTime(new Date(now.getTime() - 30000).toISOString())).toBe('たった今');
  });

  it('1時間前のとき "1時間前" を返す', () => {
    const date = new Date(now.getTime() - 60 * 60 * 1000).toISOString();
    expect(formatRelativeTime(date)).toBe('1時間前');
  });

  it('30分前のとき "30分前" を返す', () => {
    const date = new Date(now.getTime() - 30 * 60 * 1000).toISOString();
    expect(formatRelativeTime(date)).toBe('30分前');
  });

  it('3日前のとき "3日前" を返す', () => {
    const date = new Date(now.getTime() - 3 * 24 * 60 * 60 * 1000).toISOString();
    expect(formatRelativeTime(date)).toBe('3日前');
  });
});

// ─── handleApiError ───────────────────────────────────────────────────────────

describe('handleApiError', () => {
  it('非 axios エラーのとき汎用メッセージを返す', () => {
    expect(handleApiError(new Error('unknown'))).toBe('予期しないエラーが発生しました');
  });

  it('文字列エラーのとき汎用メッセージを返す', () => {
    expect(handleApiError('some error')).toBe('予期しないエラーが発生しました');
  });
});
