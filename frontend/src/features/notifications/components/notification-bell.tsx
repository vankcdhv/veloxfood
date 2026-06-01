'use client';

import { useState } from 'react';
import { Bell, CheckCheck } from 'lucide-react';
import { toast } from 'sonner';
import { Button } from '@/shared/ui/button';
import { Badge } from '@/shared/ui/badge';
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetTrigger } from '@/shared/ui/sheet';
import { Skeleton } from '@/shared/ui/skeleton';
import { getApiErrorMessage } from '@/shared/lib/api-error';
import {
  useUnreadCount,
  useNotifications,
  useMarkNotificationRead,
  useMarkAllNotificationsRead,
} from '../hooks/use-notifications';
import type { Notification } from '../types/notification';

export function NotificationBell() {
  const [open, setOpen] = useState(false);
  const { data: countData } = useUnreadCount();
  const unread = countData?.Count ?? 0;

  return (
    <Sheet open={open} onOpenChange={setOpen}>
      <SheetTrigger asChild>
        <Button variant="ghost" size="icon" aria-label="Thông báo" className="relative">
          <Bell className="h-4 w-4" />
          {unread > 0 && (
            <Badge
              variant="destructive"
              className="absolute -top-1 -right-1 h-5 min-w-5 justify-center px-1 text-[10px]"
            >
              {unread > 99 ? '99+' : unread}
            </Badge>
          )}
        </Button>
      </SheetTrigger>
      <SheetContent side="right" className="w-80 sm:w-96 flex flex-col p-0">
        <NotificationSheetBody onClose={() => setOpen(false)} />
      </SheetContent>
    </Sheet>
  );
}

// onClose is reserved for future sheet dismissal on notification click.
// eslint-disable-next-line @typescript-eslint/no-unused-vars
function NotificationSheetBody({ onClose }: { onClose: () => void }) {
  const { data, isLoading, isError } = useNotifications(0, 30);
  const markRead = useMarkNotificationRead();
  const markAll = useMarkAllNotificationsRead();

  const notifications = data?.Items ?? [];
  const hasUnread = notifications.some((n) => !n.ReadAt);

  const handleMarkAll = async () => {
    try {
      await markAll.mutateAsync();
      toast.success('Đã đánh dấu tất cả là đã đọc');
    } catch (e) {
      toast.error(getApiErrorMessage(e, 'Không cập nhật được'));
    }
  };

  const handleMarkOne = async (id: string) => {
    try {
      await markRead.mutateAsync(id);
    } catch {
      // Silent – non-critical action.
    }
  };

  return (
    <>
      <SheetHeader className="px-4 py-4 border-b border-border shrink-0">
        <div className="flex items-center justify-between">
          <SheetTitle>Thông báo</SheetTitle>
          {hasUnread && (
            <Button
              variant="ghost"
              size="sm"
              className="h-7 text-xs gap-1"
              onClick={handleMarkAll}
              disabled={markAll.isPending}
            >
              <CheckCheck className="h-3.5 w-3.5" />
              Đọc tất cả
            </Button>
          )}
        </div>
      </SheetHeader>

      <div className="flex-1 overflow-y-auto">
        {isLoading && (
          <div className="space-y-2 p-4">
            {Array.from({ length: 4 }).map((_, i) => (
              <Skeleton key={i} className="h-16 rounded-lg" />
            ))}
          </div>
        )}

        {isError && (
          <p className="text-destructive text-sm text-center py-12 px-4">
            Không tải được thông báo.
          </p>
        )}

        {!isLoading && !isError && notifications.length === 0 && (
          <div className="flex flex-col items-center gap-2 py-16 text-muted-foreground px-4">
            <Bell className="h-8 w-8 opacity-30" />
            <p className="text-sm">Không có thông báo nào.</p>
          </div>
        )}

        {!isLoading && !isError && notifications.map((n) => (
          <NotificationItem
            key={n.ID}
            notification={n}
            onRead={handleMarkOne}
          />
        ))}
      </div>
    </>
  );
}

function NotificationItem({
  notification,
  onRead,
}: {
  notification: Notification;
  onRead: (id: string) => void;
}) {
  const isUnread = !notification.ReadAt;

  return (
    <button
      type="button"
      onClick={() => isUnread && onRead(notification.ID)}
      className={`w-full text-left px-4 py-3 border-b border-border last:border-0 hover:bg-muted/50 transition-colors ${
        isUnread ? 'bg-primary/5' : ''
      }`}
    >
      <div className="flex items-start gap-2">
        {isUnread && (
          <span className="mt-1.5 h-2 w-2 shrink-0 rounded-full bg-primary" />
        )}
        {!isUnread && <span className="mt-1.5 h-2 w-2 shrink-0" />}
        <div className="min-w-0 flex-1">
          <p className={`text-sm leading-snug ${isUnread ? 'font-medium' : 'text-muted-foreground'}`}>
            {notification.Title}
          </p>
          {notification.Body && (
            <p className="text-xs text-muted-foreground mt-0.5 line-clamp-2">
              {notification.Body}
            </p>
          )}
          <p className="text-xs text-muted-foreground/60 mt-1">
            {new Date(notification.CreatedAt).toLocaleString('vi-VN')}
          </p>
        </div>
      </div>
    </button>
  );
}
