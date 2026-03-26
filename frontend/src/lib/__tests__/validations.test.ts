import { loginSchema, registerSchema, sourceSchema } from '../validations';

// ─── loginSchema ──────────────────────────────────────────────────────────────

describe('loginSchema', () => {
  it('有効なデータを通過させる', () => {
    const result = loginSchema.safeParse({ email: 'test@example.com', password: 'pass123' });
    expect(result.success).toBe(true);
  });

  it('無効なメールアドレスを拒否する', () => {
    const result = loginSchema.safeParse({ email: 'not-an-email', password: 'pass123' });
    expect(result.success).toBe(false);
  });

  it('空のパスワードを拒否する', () => {
    const result = loginSchema.safeParse({ email: 'test@example.com', password: '' });
    expect(result.success).toBe(false);
  });
});

// ─── registerSchema ───────────────────────────────────────────────────────────

describe('registerSchema', () => {
  const valid = {
    email: 'user@example.com',
    password: 'Password1',
    display_name: '山田太郎',
  };

  it('有効なデータを通過させる', () => {
    expect(registerSchema.safeParse(valid).success).toBe(true);
  });

  it('8文字未満のパスワードを拒否する', () => {
    const result = registerSchema.safeParse({ ...valid, password: 'Pass1' });
    expect(result.success).toBe(false);
  });

  it('英字なしパスワードを拒否する', () => {
    const result = registerSchema.safeParse({ ...valid, password: '12345678' });
    expect(result.success).toBe(false);
  });

  it('数字なしパスワードを拒否する', () => {
    const result = registerSchema.safeParse({ ...valid, password: 'PasswordOnly' });
    expect(result.success).toBe(false);
  });

  it('51文字以上の display_name を拒否する', () => {
    const result = registerSchema.safeParse({ ...valid, display_name: 'a'.repeat(51) });
    expect(result.success).toBe(false);
  });

  it('空の display_name を拒否する', () => {
    const result = registerSchema.safeParse({ ...valid, display_name: '' });
    expect(result.success).toBe(false);
  });
});

// ─── sourceSchema ─────────────────────────────────────────────────────────────

describe('sourceSchema', () => {
  const valid = {
    name: 'Go Blog',
    feed_url: 'https://go.dev/feed.rss',
    site_url: 'https://go.dev/blog',
    category: 'language',
    language: 'en' as const,
    fetch_interval_minutes: 60,
    quality_score: 0.8,
  };

  it('有効なデータを通過させる', () => {
    expect(sourceSchema.safeParse(valid).success).toBe(true);
  });

  it('無効な feed_url を拒否する', () => {
    const result = sourceSchema.safeParse({ ...valid, feed_url: 'not-a-url' });
    expect(result.success).toBe(false);
  });

  it('5分未満の取得間隔を拒否する', () => {
    const result = sourceSchema.safeParse({ ...valid, fetch_interval_minutes: 4 });
    expect(result.success).toBe(false);
  });

  it('quality_score が 1 を超えるを拒否する', () => {
    const result = sourceSchema.safeParse({ ...valid, quality_score: 1.1 });
    expect(result.success).toBe(false);
  });

  it('quality_score が負数を拒否する', () => {
    const result = sourceSchema.safeParse({ ...valid, quality_score: -0.1 });
    expect(result.success).toBe(false);
  });

  it('language が ja/en 以外を拒否する', () => {
    const result = sourceSchema.safeParse({ ...valid, language: 'zh' });
    expect(result.success).toBe(false);
  });
});
