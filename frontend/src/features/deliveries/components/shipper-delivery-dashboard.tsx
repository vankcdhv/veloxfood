'use client';

import { useState } from 'react';
import { Truck, ListOrdered } from 'lucide-react';
import { RoleGuard } from '@/features/auth/components/role-guard';
import { AvailableDeliveriesList } from './available-deliveries-list';
import { MyDeliveriesPanel } from './my-deliveries-panel';

type Tab = 'available' | 'mine';

export function ShipperDeliveryDashboard() {
  const [tab, setTab] = useState<Tab>('available');

  return (
    <RoleGuard allow={(auth) => auth.isShipper} loginNext="/account/deliveries">
      <div className="mx-auto max-w-2xl px-4 py-6 space-y-4">
        <div className="flex items-center gap-2">
          <Truck className="h-5 w-5 text-primary" />
          <h1 className="text-xl font-bold">Giao hàng</h1>
        </div>

        {/* Tab switcher */}
        <div className="flex rounded-lg border border-border overflow-hidden text-sm font-medium">
          <TabButton
            active={tab === 'available'}
            onClick={() => setTab('available')}
            icon={<Truck className="h-3.5 w-3.5" />}
            label="Đơn chờ nhận"
          />
          <TabButton
            active={tab === 'mine'}
            onClick={() => setTab('mine')}
            icon={<ListOrdered className="h-3.5 w-3.5" />}
            label="Đơn của tôi"
          />
        </div>

        {tab === 'available' && <AvailableDeliveriesList />}
        {tab === 'mine' && <MyDeliveriesPanel />}
      </div>
    </RoleGuard>
  );
}

function TabButton({
  active,
  onClick,
  icon,
  label,
}: {
  active: boolean;
  onClick: () => void;
  icon: React.ReactNode;
  label: string;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`flex flex-1 items-center justify-center gap-1.5 px-4 py-2 transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring ${
        active
          ? 'bg-primary text-primary-foreground'
          : 'bg-background text-muted-foreground hover:bg-muted/50'
      }`}
    >
      {icon}
      {label}
    </button>
  );
}
