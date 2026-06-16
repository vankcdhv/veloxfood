import type { Metadata } from 'next';
import { StoreListView } from '@/features/stores/components/store-list-view';

export const metadata: Metadata = {
  title: 'Cửa hàng · VeloxFood',
};

export default async function StoresPage({
  searchParams,
}: {
  searchParams: Promise<{ cuisine?: string }>;
}) {
  const { cuisine } = await searchParams;
  return (
    <main className="bg-background min-h-dvh px-4 py-10">
      <div className="mx-auto w-full max-w-5xl space-y-6">
        <div>
          <h1 className="font-serif text-3xl font-bold">Cửa hàng</h1>
          <p className="text-muted-foreground mt-1 text-sm">
            Khám phá các quầy ăn trong khuôn viên trường.
          </p>
        </div>
        <StoreListView initialCuisine={cuisine} />
      </div>
    </main>
  );
}
