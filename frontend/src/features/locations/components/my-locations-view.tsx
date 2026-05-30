'use client';

import { useState } from 'react';
import { Loader2, MapPin, Star, Trash2 } from 'lucide-react';
import { toast } from 'sonner';
import { Button } from '@/shared/ui/button';
import { Input } from '@/shared/ui/input';
import { Badge } from '@/shared/ui/badge';
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card';
import { getApiErrorMessage } from '@/shared/lib/api-error';
import { useMyLocations, useMyLocationMutations } from '../hooks/use-locations';
import { formatRoomPath } from '../lib/format-room-path';
import { LocationPicker } from './location-picker';

export function MyLocationsView() {
  return (
    <div className="space-y-6">
      <AddLocationCard />
      <SavedLocationsCard />
    </div>
  );
}

function AddLocationCard() {
  const [roomId, setRoomId] = useState('');
  const [label, setLabel] = useState('');
  const [makeDefault, setMakeDefault] = useState(false);
  // Incrementing this key remounts the LocationPicker, resetting its internal
  // cascade state after a successful form submission (avoids useEffect setState).
  const [pickerKey, setPickerKey] = useState(0);
  const { add } = useMyLocationMutations();

  const submit = () => {
    if (!roomId) {
      toast.error('Vui lòng chọn phòng.');
      return;
    }
    add.mutate(
      { roomId, label: label.trim(), isDefault: makeDefault },
      {
        onSuccess: () => {
          toast.success('Đã lưu vị trí.');
          setRoomId('');
          setLabel('');
          setMakeDefault(false);
          setPickerKey((k) => k + 1);
        },
        onError: (e) => toast.error(getApiErrorMessage(e, 'Lưu vị trí thất bại')),
      },
    );
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle>Thêm vị trí giao</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        <LocationPicker key={pickerKey} level="room" value={roomId} onSelect={setRoomId} />
        <div>
          <label className="text-foreground mb-1.5 block text-sm font-medium">Nhãn (tuỳ chọn)</label>
          <Input value={label} onChange={(e) => setLabel(e.target.value)} placeholder="VD: Văn phòng, Phòng họp" />
        </div>
        <label className="flex cursor-pointer items-center gap-2 text-sm">
          <input type="checkbox" checked={makeDefault} onChange={(e) => setMakeDefault(e.target.checked)}
            className="border-input text-primary focus-visible:ring-ring h-4 w-4 rounded border" />
          <span className="text-muted-foreground">Đặt làm vị trí mặc định</span>
        </label>
        <Button onClick={submit} disabled={add.isPending} className="w-full">
          {add.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : 'Lưu vị trí'}
        </Button>
      </CardContent>
    </Card>
  );
}

function SavedLocationsCard() {
  const { data, isLoading } = useMyLocations();
  const { remove, setDefault } = useMyLocationMutations();

  return (
    <Card>
      <CardHeader>
        <CardTitle>Vị trí đã lưu</CardTitle>
      </CardHeader>
      <CardContent>
        {isLoading && <p className="text-muted-foreground text-sm">Đang tải…</p>}
        {!isLoading && (data?.length ?? 0) === 0 && (
          <p className="text-muted-foreground text-sm">Chưa có vị trí nào.</p>
        )}
        <ul className="space-y-2">
          {data?.map((loc) => (
            <li key={loc.ID} className="border-border flex items-center gap-3 rounded-lg border p-3">
              <MapPin className="text-primary h-5 w-5 shrink-0" />
              <div className="min-w-0 flex-1">
                <p className="truncate text-sm font-medium">{loc.Label || 'Vị trí'}</p>
                <p className="text-muted-foreground truncate text-xs">{formatRoomPath(loc)}</p>
              </div>
              {loc.IsDefault ? (
                <Badge variant="success">Mặc định</Badge>
              ) : (
                <Button size="sm" variant="ghost" onClick={() => setDefault.mutate(loc.ID)}>
                  <Star className="h-4 w-4" /> Đặt mặc định
                </Button>
              )}
              <button type="button" aria-label="Xoá" onClick={() => remove.mutate(loc.ID)}
                className="text-muted-foreground hover:text-destructive p-1">
                <Trash2 className="h-4 w-4" />
              </button>
            </li>
          ))}
        </ul>
      </CardContent>
    </Card>
  );
}

