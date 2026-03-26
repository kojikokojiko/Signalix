import { NavbarContainer } from '@/containers/NavbarContainer';
import { ArticleDetailContainer } from '@/containers/ArticleDetailContainer';

interface Props {
  params: { id: string };
}

export default function ArticleDetailPage({ params }: Props) {
  return (
    <>
      <NavbarContainer />
      <ArticleDetailContainer articleId={params.id} />
    </>
  );
}
