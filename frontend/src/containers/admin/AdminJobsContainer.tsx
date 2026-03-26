'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { useAuth } from '@/contexts/AuthContext';
import { useAdminJobs } from '@/hooks/useAdminStats';
import { formatRelativeTime } from '@/lib/utils';
import { Pagination } from '@/components/ui/Pagination';
import type { IngestionJob } from '@/types/api';

const STATUS_COLORS: Record<string, string> = {
  completed: 'bg-green-100 text-green-700',
  running: 'bg-blue-100 text-blue-700',
  failed: 'bg-red-100 text-red-700',
  pending: 'bg-gray-100 text-gray-600',
};

// ─── Presentation ─────────────────────────────────────────────────────────────

export interface JobsTableProps {
  jobs: IngestionJob[];
  isLoading: boolean;
}

export function JobsTable({ jobs, isLoading }: JobsTableProps) {
  if (isLoading) {
    return (
      <div className="animate-pulse space-y-3">
        {[1, 2, 3].map((i) => <div key={i} className="h-16 bg-gray-100 rounded-xl" />)}
      </div>
    );
  }

  return (
    <div className="bg-white rounded-xl border border-gray-200 overflow-hidden">
      <table className="w-full text-sm">
        <thead className="bg-gray-50 border-b border-gray-200">
          <tr>
            {['ソース', 'ステータス', '新規', 'スキップ', '開始時刻', 'エラー'].map((h) => (
              <th key={h} className="text-left px-4 py-3 font-medium text-gray-600">{h}</th>
            ))}
          </tr>
        </thead>
        <tbody>
          {jobs.map((job) => (
            <tr key={job.id} className="border-b border-gray-100 hover:bg-gray-50">
              <td className="px-4 py-3 font-medium text-gray-900">{job.source_name}</td>
              <td className="px-4 py-3">
                <span className={`text-xs px-2 py-1 rounded-full ${STATUS_COLORS[job.status] ?? 'bg-gray-100 text-gray-600'}`}>
                  {job.status}
                </span>
              </td>
              <td className="px-4 py-3 text-gray-600">{job.articles_new}</td>
              <td className="px-4 py-3 text-gray-600">{job.articles_skipped}</td>
              <td className="px-4 py-3 text-gray-500">{formatRelativeTime(job.started_at)}</td>
              <td className="px-4 py-3 text-red-500 text-xs max-w-xs truncate">
                {job.error_message ?? '—'}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

// ─── Container ────────────────────────────────────────────────────────────────

export function AdminJobsContainer() {
  const { user, status } = useAuth();
  const router = useRouter();
  const [page, setPage] = useState(1);

  useEffect(() => {
    if (status === 'authenticated' && !user?.is_admin) router.push('/feed');
    if (status === 'unauthenticated') router.push('/login');
  }, [status, user, router]);

  const { data, isLoading } = useAdminJobs(
    page,
    status === 'authenticated' && (user?.is_admin ?? false)
  );

  if (status !== 'authenticated' || !user?.is_admin) return null;

  const jobs = data?.data ?? [];
  const pagination = data?.pagination;

  return (
    <main className="max-w-5xl mx-auto px-4 py-6">
      <h1 className="text-xl font-semibold text-gray-900 mb-6">ジョブ監視</h1>
      <JobsTable jobs={jobs} isLoading={isLoading} />
      {pagination && pagination.total_pages > 1 && (
        <Pagination
          page={page}
          totalPages={pagination.total_pages}
          hasNext={pagination.has_next}
          hasPrev={pagination.has_prev}
          onNext={() => setPage((p) => p + 1)}
          onPrev={() => setPage((p) => p - 1)}
        />
      )}
    </main>
  );
}
