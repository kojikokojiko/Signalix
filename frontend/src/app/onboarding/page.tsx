'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { useMutation } from '@tanstack/react-query';
import { apiClient } from '@/lib/api-client';

const TAGS_BY_CATEGORY: Record<string, string[]> = {
  '言語': ['go', 'rust', 'python', 'typescript', 'java', 'kotlin', 'swift', 'ruby'],
  'インフラ': ['kubernetes', 'docker', 'aws', 'gcp', 'azure', 'terraform', 'linux'],
  'AI・ML': ['llm', 'machine-learning', 'deep-learning', 'nlp', 'computer-vision'],
  'トピック': ['backend', 'frontend', 'security', 'database', 'architecture', 'devops', 'open-source'],
};

type Language = 'ja' | 'en';

export default function OnboardingPage() {
  const router = useRouter();
  const [step, setStep] = useState(1);
  const [selectedTags, setSelectedTags] = useState<Set<string>>(new Set());
  const [language, setLanguage] = useState<Language>('ja');

  const updateInterests = useMutation({
    mutationFn: () =>
      apiClient.users.updateInterests({
        interests: Array.from(selectedTags).map((tag) => ({ tag_name: tag, weight: 1.0 })),
      }),
  });

  const updateUser = useMutation({
    mutationFn: () => apiClient.users.update({ preferred_language: language }),
  });

  const toggleTag = (tag: string) => {
    setSelectedTags((prev) => {
      const next = new Set(prev);
      if (next.has(tag)) next.delete(tag);
      else next.add(tag);
      return next;
    });
  };

  const handleComplete = async () => {
    await Promise.all([
      selectedTags.size > 0 ? updateInterests.mutateAsync() : Promise.resolve(),
      updateUser.mutateAsync(),
    ]);
    router.push('/feed');
  };

  return (
    <div className="min-h-screen bg-gray-50 flex flex-col items-center justify-center px-4 py-12">
      <div className="w-full max-w-2xl">
        <div className="text-center mb-8">
          <h1 className="text-2xl font-bold text-gray-900 mb-2">ようこそ、Signalixへ!</h1>
          <p className="text-gray-600">あなた専用のフィードを作りましょう</p>
        </div>

        {/* Step indicator */}
        <div className="flex justify-center gap-2 mb-8">
          {[1, 2].map((s) => (
            <div
              key={s}
              className={`h-2 rounded-full transition-all ${
                s === step ? 'w-8 bg-primary-600' : 'w-2 bg-gray-300'
              }`}
            />
          ))}
        </div>

        {step === 1 && (
          <div className="bg-white rounded-2xl border border-gray-200 p-6">
            <h2 className="text-lg font-semibold text-gray-900 mb-4">
              どの分野に興味がありますか？
            </h2>
            {Object.entries(TAGS_BY_CATEGORY).map(([category, tags]) => (
              <div key={category} className="mb-4">
                <p className="text-xs font-medium text-gray-500 uppercase tracking-wide mb-2">
                  {category}
                </p>
                <div className="flex flex-wrap gap-2">
                  {tags.map((tag) => (
                    <button
                      key={tag}
                      onClick={() => toggleTag(tag)}
                      className={`px-3 py-1.5 rounded-full text-sm transition-colors ${
                        selectedTags.has(tag)
                          ? 'bg-primary-600 text-white'
                          : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
                      }`}
                    >
                      {tag}
                    </button>
                  ))}
                </div>
              </div>
            ))}

            <div className="flex justify-between mt-6">
              <button
                onClick={() => router.push('/feed')}
                className="text-sm text-gray-500 hover:text-gray-700"
              >
                スキップ
              </button>
              <button
                onClick={() => setStep(2)}
                disabled={selectedTags.size === 0}
                className="bg-primary-600 text-white px-6 py-2 rounded-lg text-sm font-medium hover:bg-primary-700 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                次へ ({selectedTags.size}個選択中)
              </button>
            </div>
          </div>
        )}

        {step === 2 && (
          <div className="bg-white rounded-2xl border border-gray-200 p-6">
            <h2 className="text-lg font-semibold text-gray-900 mb-4">
              優先言語を選んでください
            </h2>
            <div className="space-y-3">
              {(['ja', 'en'] as Language[]).map((lang) => (
                <label
                  key={lang}
                  className={`flex items-center gap-3 p-4 rounded-xl border-2 cursor-pointer transition-colors ${
                    language === lang
                      ? 'border-primary-500 bg-primary-50'
                      : 'border-gray-200 hover:border-gray-300'
                  }`}
                >
                  <input
                    type="radio"
                    name="language"
                    value={lang}
                    checked={language === lang}
                    onChange={() => setLanguage(lang)}
                    className="text-primary-600"
                  />
                  <span className="font-medium text-gray-900">
                    {lang === 'ja' ? '🇯🇵 日本語' : '🇺🇸 English'}
                  </span>
                </label>
              ))}
            </div>

            <div className="flex justify-between mt-6">
              <button
                onClick={() => setStep(1)}
                className="text-sm text-gray-500 hover:text-gray-700"
              >
                ← 戻る
              </button>
              <button
                onClick={handleComplete}
                disabled={updateInterests.isPending || updateUser.isPending}
                className="bg-primary-600 text-white px-6 py-2 rounded-lg text-sm font-medium hover:bg-primary-700 disabled:opacity-50 flex items-center gap-2"
              >
                {(updateInterests.isPending || updateUser.isPending) ? (
                  <>
                    <span className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin" />
                    保存中...
                  </>
                ) : (
                  '完了してフィードを見る'
                )}
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
