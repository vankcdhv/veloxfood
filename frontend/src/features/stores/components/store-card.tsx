import Link from 'next/link';
import { Package, Phone, Truck } from 'lucide-react';
import { Card, CardContent } from '@/shared/ui/card';
import { Badge } from '@/shared/ui/badge';
import { SaleStatusBadge } from './sale-status-badge';
import type { Store } from '../types/store';

interface StoreCardProps {
  store: Store;
  href?: string;
}

export function StoreCard({ store, href }: StoreCardProps) {
  const content = (
    <Card className="group cursor-pointer transition-shadow hover:shadow-md">
      <CardContent className="p-5 space-y-3">
        <div className="flex items-start justify-between gap-2">
          <h3 className="font-serif text-base font-semibold leading-snug group-hover:text-primary transition-colors">
            {store.Name}
          </h3>
          <SaleStatusBadge status={store.SaleStatus} />
        </div>

        {store.BusinessType && (
          <p className="text-muted-foreground text-xs">{store.BusinessType}</p>
        )}

        {store.Address && (
          <p className="text-sm text-muted-foreground truncate">{store.Address}</p>
        )}

        <div className="flex items-center gap-3 flex-wrap">
          {store.Phone && (
            <span className="flex items-center gap-1 text-xs text-muted-foreground">
              <Phone className="h-3.5 w-3.5" />
              {store.Phone}
            </span>
          )}
          {store.PickupEnabled && (
            <Badge variant="secondary" className="text-xs gap-1">
              <Package className="h-3 w-3" />
              Tự lấy
            </Badge>
          )}
          <Badge variant="outline" className="text-xs gap-1">
            <Truck className="h-3 w-3" />
            Giao hàng
          </Badge>
        </div>
      </CardContent>
    </Card>
  );

  if (href) {
    return <Link href={href}>{content}</Link>;
  }
  return content;
}
