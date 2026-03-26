'use client';

import { useState, useRef, useEffect } from 'react';
import { useMutation } from '@tanstack/react-query';
import { apiClient } from '@/lib/api-client';
import type { ChatMessage } from '@/types/api';

interface Props {
  articleId: string;
}

export function ArticleChat({ articleId }: Props) {
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [input, setInput] = useState('');
  const [isOpen, setIsOpen] = useState(false);
  const bottomRef = useRef<HTMLDivElement>(null);

  const mutation = useMutation({
    mutationFn: (message: string) =>
      apiClient.articles.chat(articleId, message, messages),
    onSuccess: (res, message) => {
      setMessages((prev) => [
        ...prev,
        { role: 'user', content: message },
        { role: 'assistant', content: res.data.reply },
      ]);
    },
  });

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const msg = input.trim();
    if (!msg || mutation.isPending) return;
    setInput('');
    mutation.mutate(msg);
  };

  return (
    <div className="border border-gray-200 rounded-xl overflow-hidden">
      <button
        onClick={() => setIsOpen((v) => !v)}
        className="w-full flex items-center justify-between px-4 py-3 bg-gray-50 hover:bg-gray-100 text-sm font-medium text-gray-700"
      >
        <span>💬 この記事についてAIに質問する</span>
        <span className="text-gray-400">{isOpen ? '▲' : '▼'}</span>
      </button>

      {isOpen && (
        <div className="flex flex-col bg-white">
          {/* メッセージ一覧 */}
          <div className="max-h-80 overflow-y-auto p-4 space-y-3">
            {messages.length === 0 && (
              <p className="text-xs text-gray-400 text-center py-4">
                記事の内容について何でも聞いてください
              </p>
            )}
            {messages.map((m, i) => (
              <div
                key={i}
                className={`flex ${m.role === 'user' ? 'justify-end' : 'justify-start'}`}
              >
                <div
                  className={`max-w-[85%] rounded-xl px-3 py-2 text-sm whitespace-pre-wrap ${
                    m.role === 'user'
                      ? 'bg-primary-600 text-white'
                      : 'bg-gray-100 text-gray-800'
                  }`}
                >
                  {m.content}
                </div>
              </div>
            ))}
            {mutation.isPending && (
              <div className="flex justify-start">
                <div className="bg-gray-100 rounded-xl px-3 py-2 text-sm text-gray-500">
                  <span className="animate-pulse">考え中...</span>
                </div>
              </div>
            )}
            {mutation.isError && (
              <p className="text-xs text-red-500 text-center">
                エラーが発生しました。もう一度お試しください。
              </p>
            )}
            <div ref={bottomRef} />
          </div>

          {/* 入力欄 */}
          <form
            onSubmit={handleSubmit}
            className="border-t border-gray-200 flex items-center gap-2 p-3"
          >
            <input
              type="text"
              value={input}
              onChange={(e) => setInput(e.target.value)}
              placeholder="質問を入力..."
              disabled={mutation.isPending}
              className="flex-1 text-sm border border-gray-200 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-primary-300 disabled:bg-gray-50"
            />
            <button
              type="submit"
              disabled={!input.trim() || mutation.isPending}
              className="text-sm bg-primary-600 text-white px-3 py-2 rounded-lg hover:bg-primary-700 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              送信
            </button>
          </form>
        </div>
      )}
    </div>
  );
}
