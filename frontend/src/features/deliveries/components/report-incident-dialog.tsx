'use client';

import { useState } from 'react';
import { toast } from 'sonner';
import { AlertTriangle } from 'lucide-react';
import { Button } from '@/shared/ui/button';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/shared/ui/dialog';
import { Input } from '@/shared/ui/input';
import { Label } from '@/shared/ui/label';
import { getApiErrorMessage } from '@/shared/lib/api-error';
import { useReportIncident } from '../hooks/use-deliveries';
import { INCIDENT_TYPES } from '../types/delivery';

interface ReportIncidentDialogProps {
  orderId: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function ReportIncidentDialog({ orderId, open, onOpenChange }: ReportIncidentDialogProps) {
  const [type, setType] = useState<string>(INCIDENT_TYPES[0].value);
  const [note, setNote] = useState('');
  const reportMutation = useReportIncident();

  const handleSubmit = async () => {
    if (!note.trim()) {
      toast.error('Vui lòng nhập ghi chú sự cố');
      return;
    }
    try {
      await reportMutation.mutateAsync({ orderId, body: { type, note: note.trim() } });
      toast.success('Đã báo sự cố');
      setNote('');
      setType(INCIDENT_TYPES[0].value);
      onOpenChange(false);
    } catch (err) {
      toast.error(getApiErrorMessage(err, 'Không báo được sự cố'));
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-sm">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <AlertTriangle className="h-4 w-4 text-warning" />
            Báo sự cố
          </DialogTitle>
        </DialogHeader>

        <div className="space-y-4 py-1">
          <div className="space-y-1.5">
            <Label htmlFor="incident-type">Loại sự cố</Label>
            <select
              id="incident-type"
              value={type}
              onChange={(e) => setType(e.target.value)}
              className="border-input bg-background focus-visible:ring-ring flex h-9 w-full rounded-md border px-3 py-1 text-sm shadow-sm focus-visible:ring-1 focus-visible:outline-none"
            >
              {INCIDENT_TYPES.map((t) => (
                <option key={t.value} value={t.value}>
                  {t.label}
                </option>
              ))}
            </select>
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="incident-note">Ghi chú</Label>
            <Input
              id="incident-note"
              value={note}
              onChange={(e) => setNote(e.target.value)}
              placeholder="Mô tả sự cố..."
              maxLength={500}
            />
          </div>
        </div>

        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={reportMutation.isPending}
          >
            Hủy
          </Button>
          <Button
            onClick={handleSubmit}
            disabled={reportMutation.isPending || !note.trim()}
          >
            {reportMutation.isPending ? 'Đang gửi…' : 'Gửi báo cáo'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
