import axios from 'axios';

export function formatRelativeTime(dateStr: string | null): string {
  if (!dateStr) return '不明';
  const date = new Date(dateStr);
  const now = new Date();
  const diff = now.getTime() - date.getTime();
  const minutes = Math.floor(diff / 60000);
  const hours = Math.floor(minutes / 60);
  const days = Math.floor(hours / 24);

  if (minutes < 1) return 'たった今';
  if (minutes < 60) return `${minutes}分前`;
  if (hours < 24) return `${hours}時間前`;
  if (days < 7) return `${days}日前`;
  return date.toLocaleDateString('ja-JP');
}

export function handleApiError(error: unknown): string {
  if (axios.isAxiosError(error)) {
    const code = error.response?.data?.code as string | undefined;
    const message = error.response?.data?.message as string | undefined;

    if (message) return message;

    switch (code) {
      case 'email_already_exists':
        return 'このメールアドレスは既に使用されています';
      case 'invalid_credentials':
        return 'メールアドレスまたはパスワードが間違っています';
      case 'account_locked':
        return 'アカウントが一時的にロックされています。15分後に再試行してください';
      case 'account_disabled':
        return 'アカウントが無効化されています';
      case 'rate_limit_exceeded':
        return 'リクエストが多すぎます。しばらく待ってから再試行してください';
    }

    if (error.response?.status === 429) {
      return 'リクエストが多すぎます。しばらく待ってから再試行してください';
    }
    if ((error.response?.status ?? 0) >= 500) {
      return 'サーバーエラーが発生しました。しばらく待ってから再試行してください';
    }
  }
  return '予期しないエラーが発生しました';
}
