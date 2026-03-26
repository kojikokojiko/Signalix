'use client';

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';
import Link from 'next/link';
import { useAuth } from '@/contexts/AuthContext';
import { useAdminStats } from '@/hooks/useAdminStats';
import type { AdminStats } from '@/types/api';

function StatCard({ label, value, sub }: { label: string; value: number; sub?: string }) {
  return (
    <div className="bg-white rounded-xl border border-gray-200 p-4">
      <p className="text-sm text-gray-500 mb-1">{label}</p>
      <p className="text-3xl font-bold text-gray-900">{value.toLocaleString()}</p>
      {sub && <p className="text-xs text-gray-400 mt-1">{sub}</p>}
    </div>
  );
}

interface AdminDashboardViewProps {
  stats: AdminStats | undefined;
  isLoading: boolean;
}

export function AdminDashboardView({ stats, isLoading }: AdminDashboardViewProps) {
  return (
    <>
      {isLoading ? (
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-4 animate-pulse">
          {[1, 2, 3, 4].map((i) => (
            <div key={i} className="h-24 bg-gray-200 rounded-xl" />
          ))}
        </div>
      ) : stats ? (
        <>
          {(stats.sources.degraded > 0 || stats.sources.disabled > 0) && (
            <div className="mb-4 p-3 bg-yellow-50 border border-yellow-200 rounded-lg text-sm text-yellow-800">
              ⚠ {stats.sources.degraded} ソースが degraded、{stats.sources.disabled} ソースが disabled です
            </div>
          )}
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-4 mb-8">
            <StatCard label="アクティブソース" value={stats.sources.active} sub={`全 ${stats.sources.total} ソース`} />
            <StatCard label="処理済み記事" value={stats.articles.processed} sub={`未処理 ${stats.articles.pending}`} />
            <StatCard label="完了ジョブ (24h)" value={stats.ingestion_jobs.last_24h_completed} />
            <StatCard label="失敗ジョブ (24h)" value={stats.ingestion_jobs.last_24h_failed} />
          </div>
          <div className="grid sm:grid-cols-2 gap-4">
            <Link href="/admin/sources" className="bg-white rounded-xl border border-gray-200 p-4 hover:shadow-sm transition-shadow">
              <h3 className="font-semibold text-gray-900 mb-1">ソース管理 →</h3>
              <p className="text-sm text-gray-500">RSSソースの追加・編集・削除</p>
            </Link>
            <Link href="/admin/jobs" className="bg-white rounded-xl border border-gray-200 p-4 hover:shadow-sm transition-shadow">
              <h3 className="font-semibold text-gray-900 mb-1">ジョブ監視 →</h3>
              <p className="text-sm text-gray-500">インジェスション・処理ジョブの監視</p>
            </Link>
          </div>
        </>
      ) : null}
    </>
  );
}

export function AdminDashboardContainer() {
  const { user, status } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (status === 'authenticated' && !user?.is_admin) router.push('/feed');
    if (status === 'unauthenticated') router.push('/login');
  }, [status, user, router]);

  const { data, isLoading } = useAdminStats(
    status === 'authenticated' && (user?.is_admin ?? false)
  );

  if (status !== 'authenticated' || !user?.is_admin) return null;

  return (
    <main className="max-w-4xl mx-auto px-4 py-6">
      <h1 className="text-xl font-semibold text-gray-900 mb-6">管理ダッシュボード</h1>
      <AdminDashboardView stats={data?.data} isLoading={isLoading} />
    </main>
  );
}
