'use client';

interface ErrorPageProps {
  error: Error;
  reset: () => void;
}

export default function ErrorPage({ error, reset }: ErrorPageProps) {
  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50 px-4">
      <div className="text-center max-w-md">
        <p className="text-6xl font-bold text-red-500 mb-4">500</p>
        <h1 className="text-2xl font-semibold text-gray-900 mb-2">
          サーバーエラーが発生しました
        </h1>
        <p className="text-gray-600 mb-2">
          予期しないエラーが発生しました。しばらく待ってから再試行してください。
        </p>
        {error.message && (
          <p className="text-sm text-gray-400 mb-6 font-mono break-all">
            {error.message}
          </p>
        )}
        <button
          onClick={reset}
          className="inline-block bg-primary-600 text-white px-6 py-3 rounded-xl font-medium hover:bg-primary-700"
        >
          再試行する
        </button>
      </div>
    </div>
  );
}
