'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { useAuth } from '@/contexts/AuthContext';
import {
  useAdminSources,
  useCreateSource,
  useUpdateSource,
  useDeleteSource,
  useTriggerFetch,
} from '@/hooks/useAdminStats';
import { sourceSchema, SourceFormValues } from '@/lib/validations';
import { formatRelativeTime, handleApiError } from '@/lib/utils';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import type { Source } from '@/types/api';

const STATUS_COLORS: Record<string, string> = {
  active: 'bg-green-100 text-green-700',
  paused: 'bg-yellow-100 text-yellow-700',
  degraded: 'bg-orange-100 text-orange-700',
  disabled: 'bg-red-100 text-red-700',
};

// ─── Source Form Modal (Presentation) ────────────────────────────────────────

interface SourceFormProps {
  source?: Source;
  isSubmitting: boolean;
  error: string;
  onSubmit: (data: SourceFormValues) => void;
  onCancel: () => void;
}

export function SourceForm({ source, isSubmitting, error, onSubmit, onCancel }: SourceFormProps) {
  const isEdit = !!source;
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<SourceFormValues>({
    resolver: zodResolver(sourceSchema),
    defaultValues: source
      ? {
          name: source.name,
          feed_url: source.feed_url,
          site_url: source.site_url,
          category: source.category,
          language: source.language as 'ja' | 'en',
          description: source.description ?? undefined,
          fetch_interval_minutes: source.fetch_interval_minutes,
          quality_score: source.quality_score,
        }
      : {
          language: 'en' as const,
          fetch_interval_minutes: 60,
          quality_score: 0.7,
          name: '',
          feed_url: '',
          site_url: '',
          category: '',
        },
  });

  return (
    <div className="fixed inset-0 bg-black/40 z-50 flex items-center justify-center p-4">
      <div className="bg-white rounded-2xl w-full max-w-md p-6 max-h-[90vh] overflow-y-auto">
        <div className="flex justify-between mb-4">
          <h2 className="font-semibold text-gray-900">{isEdit ? 'ソースを編集' : 'ソースを追加'}</h2>
          <button onClick={onCancel} className="text-gray-400 hover:text-gray-600">✕</button>
        </div>

        {error && (
          <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700">
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit((d: SourceFormValues) => onSubmit(d))} className="space-y-3">
          <Input label="名前" error={errors.name?.message} {...register('name')} />
          <Input label="Feed URL" type="url" error={errors.feed_url?.message} {...register('feed_url')} />
          <Input label="サイト URL" type="url" error={errors.site_url?.message} {...register('site_url')} />
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">カテゴリ</label>
            <input
              list="category-suggestions"
              className={`w-full border rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent ${errors.category ? 'border-red-400' : 'border-gray-300'}`}
              placeholder="例: backend, ai, tech ..."
              {...register('category')}
            />
            <datalist id="category-suggestions">
              {['tech', 'ai', 'startup', 'infrastructure', 'backend', 'frontend', 'security', 'data', 'other'].map((c) => (
                <option key={c} value={c} />
              ))}
            </datalist>
            {errors.category && <p className="mt-1 text-xs text-red-600">{errors.category.message}</p>}
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">言語</label>
            <select {...register('language')} className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm">
              <option value="en">English</option>
              <option value="ja">日本語</option>
            </select>
          </div>

          <div className="flex gap-3">
            <div className="flex-1">
              <Input
                label="取得間隔 (分)"
                type="number"
                error={errors.fetch_interval_minutes?.message}
                {...register('fetch_interval_minutes', { valueAsNumber: true })}
              />
            </div>
            <div className="flex-1">
              <Input
                label="品質スコア"
                type="number"
                step="0.1"
                error={errors.quality_score?.message}
                {...register('quality_score', { valueAsNumber: true })}
              />
            </div>
          </div>

          <div className="flex gap-3 pt-2">
            <Button type="button" variant="secondary" className="flex-1" onClick={onCancel}>
              キャンセル
            </Button>
            <Button type="submit" className="flex-1" loading={isSubmitting}>
              {isEdit ? '更新' : '追加'}
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
}

// ─── Container ────────────────────────────────────────────────────────────────

export function AdminSourcesContainer() {
  const { user, status } = useAuth();
  const router = useRouter();
  const [modal, setModal] = useState<{ open: boolean; source?: Source }>({ open: false });
  const [formError, setFormError] = useState('');

  useEffect(() => {
    if (status === 'authenticated' && !user?.is_admin) router.push('/feed');
    if (status === 'unauthenticated') router.push('/login');
  }, [status, user, router]);

  const { data, isLoading } = useAdminSources(
    1,
    status === 'authenticated' && (user?.is_admin ?? false)
  );
  const createSource = useCreateSource();
  const updateSource = useUpdateSource();
  const deleteSource = useDeleteSource();
  const triggerFetch = useTriggerFetch();

  const handleSubmit = async (formData: SourceFormValues) => {
    setFormError('');
    try {
      if (modal.source) {
        await updateSource.mutateAsync({ id: modal.source.id, data: formData });
      } else {
        await createSource.mutateAsync(formData);
      }
      setModal({ open: false });
    } catch (err) {
      setFormError(handleApiError(err));
    }
  };

  if (status !== 'authenticated' || !user?.is_admin) return null;

  const sources = data?.data ?? [];

  return (
    <>
      {modal.open && (
        <SourceForm
          source={modal.source}
          isSubmitting={createSource.isPending || updateSource.isPending}
          error={formError}
          onSubmit={handleSubmit}
          onCancel={() => { setModal({ open: false }); setFormError(''); }}
        />
      )}

      <main className="max-w-5xl mx-auto px-4 py-6">
        <div className="flex justify-between items-center mb-6">
          <h1 className="text-xl font-semibold text-gray-900">ソース管理</h1>
          <Button onClick={() => setModal({ open: true })}>+ 新規追加</Button>
        </div>

        {isLoading ? (
          <div className="animate-pulse space-y-3">
            {[1, 2, 3].map((i) => <div key={i} className="h-16 bg-gray-100 rounded-xl" />)}
          </div>
        ) : (
          <div className="bg-white rounded-xl border border-gray-200 overflow-hidden">
            <table className="w-full text-sm">
              <thead className="bg-gray-50 border-b border-gray-200">
                <tr>
                  {['名前', 'カテゴリ', '言語', 'ステータス', '最終取得', '操作'].map((h) => (
                    <th key={h} className="text-left px-4 py-3 font-medium text-gray-600">{h}</th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {sources.map((src) => (
                  <tr key={src.id} className="border-b border-gray-100 hover:bg-gray-50">
                    <td className="px-4 py-3 font-medium text-gray-900">{src.name}</td>
                    <td className="px-4 py-3 text-gray-600">{src.category}</td>
                    <td className="px-4 py-3 text-gray-600">{src.language}</td>
                    <td className="px-4 py-3">
                      <span className={`text-xs px-2 py-1 rounded-full ${STATUS_COLORS[src.status] ?? 'bg-gray-100 text-gray-600'}`}>
                        {src.status}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-gray-500">{formatRelativeTime(src.last_fetched_at)}</td>
                    <td className="px-4 py-3">
                      <div className="flex gap-2">
                        <button onClick={() => triggerFetch.mutate(src.id)} className="text-xs text-blue-600 hover:underline">取得</button>
                        <button onClick={() => setModal({ open: true, source: src })} className="text-xs text-gray-600 hover:underline">編集</button>
                        <button
                          onClick={() => { if (confirm('削除しますか？')) deleteSource.mutate(src.id); }}
                          className="text-xs text-red-500 hover:underline"
                        >
                          削除
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </main>
    </>
  );
}
