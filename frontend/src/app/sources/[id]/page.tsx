import { NavbarContainer } from '@/containers/NavbarContainer';
import { SourceDetailContainer } from '@/containers/SourceDetailContainer';

export default function SourceDetailPage({ params }: { params: { id: string } }) {
  return (
    <>
      <NavbarContainer />
      <SourceDetailContainer sourceId={params.id} />
    </>
  );
}
