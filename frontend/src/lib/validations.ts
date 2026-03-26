import { z } from 'zod';

export const loginSchema = z.object({
  email: z.email('有効なメールアドレスを入力してください'),
  password: z.string().min(1, 'パスワードを入力してください'),
});

export const registerSchema = z.object({
  email: z.email('有効なメールアドレスを入力してください'),
  password: z
    .string()
    .min(8, 'パスワードは8文字以上必要です')
    .check((v) => {
      if (!/[a-zA-Z]/.test(v.value))
        v.issues.push({ code: 'custom', message: '英字を含めてください', input: v.value });
      if (!/[0-9]/.test(v.value))
        v.issues.push({ code: 'custom', message: '数字を含めてください', input: v.value });
    }),
  display_name: z
    .string()
    .min(1, '表示名を入力してください')
    .max(50, '表示名は50文字以内にしてください'),
});

export const updateUserSchema = z.object({
  display_name: z.string().min(1).max(50).optional(),
  preferred_language: z.enum(['ja', 'en']).optional(),
});

export const sourceSchema = z.object({
  name: z.string().min(1, '名前を入力してください'),
  feed_url: z.url('有効なURLを入力してください'),
  site_url: z.url('有効なURLを入力してください'),
  category: z.string().min(1, 'カテゴリを入力してください'),
  language: z.enum(['ja', 'en']),
  description: z.string().nullable().optional(),
  fetch_interval_minutes: z.number().int().min(5).max(1440),
  quality_score: z.number().min(0).max(1),
});

export type LoginFormValues = z.infer<typeof loginSchema>;
export type RegisterFormValues = z.infer<typeof registerSchema>;
export type UpdateUserFormValues = z.infer<typeof updateUserSchema>;
export type SourceFormValues = z.infer<typeof sourceSchema>;
