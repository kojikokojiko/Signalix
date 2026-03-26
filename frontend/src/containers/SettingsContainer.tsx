'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { useAuth } from '@/contexts/AuthContext';
import { useUserInterests, useUpdateUser, useUpdateInterests } from '@/hooks/useUserSettings';
import { useUserSources, useUnsubscribeSource } from '@/hooks/useUserSources';
import { updateUserSchema, UpdateUserFormValues } from '@/lib/validations';
import { Input } from '@/components/ui/Input';
import { Button } from '@/components/ui/Button';

const TAGS_BY_CATEGORY: Record<string, string[]> = {
  '言語': ['go', 'rust', 'python', 'typescript', 'javascript', 'java', 'kotlin', 'swift', 'ruby'],
  'インフラ': ['kubernetes', 'docker', 'aws', 'gcp', 'azure', 'terraform', 'linux'],
  'AI・ML': ['llm', 'machine-learning', 'deep-learning', 'nlp', 'computer-vision', 'generative-ai', 'ai-infrastructure'],
  'トピック': ['backend', 'frontend', 'security', 'database', 'architecture', 'devops', 'performance', 'open-source', 'startup'],
};

export function SettingsContainer() {
  const { user, status } = useAuth();
  const router = useRouter();
  const [saved, setSaved] = useState(false);
  const [showTagPicker, setShowTagPicker] = useState(false);

  useEffect(() => {
    if (status === 'unauthenticated') router.push('/login');
  }, [status, router]);

  const { data: interestsData } = useUserInterests(status === 'authenticated');
  const { data: sourcesData } = useUserSources(status === 'authenticated');
  const updateUser = useUpdateUser();
  const updateInterests = useUpdateInterests();
  const unsubscribeSource = useUnsubscribeSource();

  const {
    register,
    handleSubmit,
    formState: { isSubmitting },
  } = useForm<UpdateUserFormValues>({
    resolver: zodResolver(updateUserSchema),
    defaultValues: {
      display_name: user?.display_name ?? '',
      preferred_language: user?.preferred_language ?? 'ja',
    },
  });

  const handleSaveProfile = async (data: UpdateUserFormValues) => {
    await updateUser.mutateAsync(data);
    setSaved(true);
    setTimeout(() => setSaved(false), 2000);
  };

  const handleRemoveInterest = async (tagName: string) => {
    const current = interestsData?.data ?? [];
    const updated = current.filter((i) => i.tag_name !== tagName);
    await updateInterests.mutateAsync({
      interests: updated.map((i) => ({ tag_name: i.tag_name, weight: i.weight })),
    });
  };

  const handleAddTag = async (tagName: string) => {
    const current = interestsData?.data ?? [];
    if (current.some((i) => i.tag_name === tagName)) return;
    await updateInterests.mutateAsync({
      interests: [...current.map((i) => ({ tag_name: i.tag_name, weight: i.weight })), { tag_name: tagName, weight: 1.0 }],
    });
  };

  if (status !== 'authenticated') return null;

  const interests = interestsData?.data ?? [];
  const existingTagNames = new Set(interests.map((i) => i.tag_name));
  const subscribedSources = sourcesData?.data ?? [];

  return (
    <main className="max-w-2xl mx-auto px-4 py-6">
      <h1 className="text-xl font-semibold text-gray-900 mb-6">設定</h1>

      {/* Profile */}
      <section className="bg-white rounded-2xl border border-gray-200 p-6 mb-6">
        <h2 className="font-semibold text-gray-900 mb-4">基本情報</h2>
        <form onSubmit={handleSubmit(handleSaveProfile)} className="space-y-4">
          <Input label="表示名" {...register('display_name')} />
          <Input label="メールアドレス" value={user?.email ?? ''} disabled />

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">優先言語</label>
            <select
              {...register('preferred_language')}
              className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
            >
              <option value="ja">🇯🇵 日本語</option>
              <option value="en">🇺🇸 English</option>
            </select>
          </div>

          <div className="flex items-center gap-3">
            <Button type="submit" loading={isSubmitting}>保存</Button>
            {saved && <span className="text-sm text-green-600">✓ 保存しました</span>}
          </div>
        </form>
      </section>

      {/* Interests */}
      <section className="bg-white rounded-2xl border border-gray-200 p-6">
        <div className="flex items-center justify-between mb-4">
          <h2 className="font-semibold text-gray-900">興味・関心</h2>
          <button
            onClick={() => setShowTagPicker((v) => !v)}
            className="text-sm text-primary-600 hover:text-primary-700 font-medium"
          >
            {showTagPicker ? '閉じる' : '+ タグを追加'}
          </button>
        </div>

        {/* Tag picker */}
        {showTagPicker && (
          <div className="mb-5 p-4 bg-gray-50 rounded-xl border border-gray-200">
            {Object.entries(TAGS_BY_CATEGORY).map(([category, tags]) => (
              <div key={category} className="mb-3 last:mb-0">
                <p className="text-xs font-medium text-gray-500 uppercase tracking-wide mb-2">{category}</p>
                <div className="flex flex-wrap gap-2">
                  {tags.map((tag) => {
                    const added = existingTagNames.has(tag);
                    return (
                      <button
                        key={tag}
                        onClick={() => !added && handleAddTag(tag)}
                        disabled={added || updateInterests.isPending}
                        className={`px-3 py-1 rounded-full text-sm transition-colors ${
                          added
                            ? 'bg-primary-100 text-primary-700 cursor-default'
                            : 'bg-white border border-gray-300 text-gray-700 hover:border-primary-400 hover:text-primary-600 disabled:opacity-50'
                        }`}
                      >
                        {added ? `✓ ${tag}` : tag}
                      </button>
                    );
                  })}
                </div>
              </div>
            ))}
          </div>
        )}

        {/* Current interests */}
        {interests.length === 0 ? (
          <p className="text-sm text-gray-500">興味タグが設定されていません</p>
        ) : (
          <div className="space-y-3">
            {interests.map((interest) => (
              <div key={interest.tag_name} className="flex items-center gap-3">
                <span className="text-sm text-gray-700 w-28 shrink-0">{interest.tag_name}</span>
                <div className="flex-1 bg-gray-100 rounded-full h-2">
                  <div
                    className="bg-primary-500 h-2 rounded-full"
                    style={{ width: `${Math.min(interest.weight * 100, 100)}%` }}
                  />
                </div>
                <button
                  onClick={() => handleRemoveInterest(interest.tag_name)}
                  disabled={updateInterests.isPending}
                  className="text-xs text-red-400 hover:text-red-600 disabled:opacity-50"
                >
                  ✕
                </button>
              </div>
            ))}
          </div>
        )}
      </section>
      {/* Subscribed sources */}
      <section className="bg-white rounded-2xl border border-gray-200 p-6 mt-6">
        <div className="flex items-center justify-between mb-4">
          <h2 className="font-semibold text-gray-900">購読中ソース</h2>
          <a
            href="/sources"
            className="text-sm text-primary-600 hover:text-primary-700 font-medium"
          >
            ソースを探す
          </a>
        </div>

        {subscribedSources.length === 0 ? (
          <p className="text-sm text-gray-500">購読しているソースはありません</p>
        ) : (
          <div className="space-y-3">
            {subscribedSources.map((source) => (
              <div key={source.id} className="flex items-center justify-between gap-3">
                <div className="min-w-0">
                  <p className="text-sm text-gray-900 font-medium truncate">{source.name}</p>
                  <p className="text-xs text-gray-500">
                    {source.category} · {source.language.toUpperCase()}
                  </p>
                </div>
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={() => unsubscribeSource.mutate(source.id)}
                  loading={unsubscribeSource.isPending}
                >
                  解除
                </Button>
              </div>
            ))}
          </div>
        )}
      </section>
    </main>
  );
}
