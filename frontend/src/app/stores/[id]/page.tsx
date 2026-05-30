import type { Metadata } from 'next';
import { StoreDetailView } from '@/features/stores/components/store-detail-view';

interface Props {
  params: Promise<{ id: string }>;
}

export const metadata: Metadata = {
  title: 'Chi tiết cửa hàng · VeloxFood',
};

export default async function StoreDetailPage({ params }: Props) {
  const { id } = await params;
  return (
    <main className="bg-background min-h-dvh px-4 py-10">
      <div className="mx-auto w-full max-w-3xl">
        <StoreDetailView storeId={id} />
      </div>
    </main>
  );
}
