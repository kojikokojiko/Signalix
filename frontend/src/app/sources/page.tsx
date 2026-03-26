import { NavbarContainer } from '@/containers/NavbarContainer';
import { SourcesContainer } from '@/containers/SourcesContainer';

export const metadata = { title: 'ソース一覧 | Signalix' };

export default function SourcesPage() {
  return (
    <>
      <NavbarContainer />
      <SourcesContainer />
    </>
  );
}
