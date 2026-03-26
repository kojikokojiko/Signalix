import Link from 'next/link';
import { NavbarContainer } from '@/containers/NavbarContainer';

export default function Home() {
  return (
    <>
      <NavbarContainer />
      <main>
        {/* Hero */}
        <section className="bg-gradient-to-b from-primary-50 to-white py-24 px-4 text-center">
          <h1 className="text-4xl sm:text-5xl font-bold text-gray-900 mb-4 leading-tight">
            AIが選ぶ、
            <br className="sm:hidden" />
            あなたのための技術ニュース
          </h1>
          <p className="text-lg text-gray-600 max-w-xl mx-auto mb-8">
            RSS・ブログ・ニュースをAIが要約・ランク付け。
            読む価値のある記事だけを、理由とともにお届けします。
          </p>
          <Link
            href="/signup"
            className="inline-block bg-primary-600 text-white text-lg px-8 py-3 rounded-xl hover:bg-primary-700 font-medium"
          >
            無料で始める
          </Link>
        </section>

        {/* Features */}
        <section className="max-w-4xl mx-auto py-20 px-4">
          <div className="grid sm:grid-cols-3 gap-8">
            <div className="text-center">
              <div className="text-4xl mb-3">✨</div>
              <h3 className="font-semibold text-gray-900 mb-2">AIによる記事要約</h3>
              <p className="text-sm text-gray-600">
                GPT-4o-miniが記事を2〜4文で要約。スキャンするだけで価値を把握できます。
              </p>
            </div>
            <div className="text-center">
              <div className="text-4xl mb-3">🎯</div>
              <h3 className="font-semibold text-gray-900 mb-2">パーソナライズフィード</h3>
              <p className="text-sm text-gray-600">
                あなたの興味・行動から学習し、最適な記事を優先表示します。
              </p>
            </div>
            <div className="text-center">
              <div className="text-4xl mb-3">💡</div>
              <h3 className="font-semibold text-gray-900 mb-2">推薦理由の透明性</h3>
              <p className="text-sm text-gray-600">
                「なぜこの記事?」が分かるスコア内訳を表示。フィードを自分でコントロール。
              </p>
            </div>
          </div>
        </section>

        {/* CTA */}
        <section className="bg-primary-600 py-16 px-4 text-center text-white">
          <h2 className="text-3xl font-bold mb-4">今すぐ始める</h2>
          <p className="text-primary-100 mb-8">無料で登録、すぐに使えます</p>
          <div className="flex justify-center gap-4">
            <Link
              href="/signup"
              className="bg-white text-primary-600 font-medium px-6 py-3 rounded-xl hover:bg-primary-50"
            >
              アカウント作成
            </Link>
            <Link
              href="/trending"
              className="border border-white text-white font-medium px-6 py-3 rounded-xl hover:bg-primary-700"
            >
              Trendingを見る
            </Link>
          </div>
        </section>

        {/* Footer */}
        <footer className="bg-gray-50 border-t border-gray-200 py-8 px-4">
          <div className="max-w-4xl mx-auto flex flex-col sm:flex-row justify-between items-center gap-4 text-sm text-gray-500">
            <span>© 2025 Signalix</span>
            <div className="flex gap-4">
              <a href="#" className="hover:text-gray-700">About</a>
              <a href="#" className="hover:text-gray-700">Privacy</a>
              <a href="#" className="hover:text-gray-700">Terms</a>
            </div>
          </div>
        </footer>
      </main>
    </>
  );
}
