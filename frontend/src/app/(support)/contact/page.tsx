import type { Metadata } from 'next';
import { Mail, Phone, MapPin, Clock } from 'lucide-react';

export const metadata: Metadata = { title: 'Liên hệ · VeloxFood' };

const CHANNELS = [
  { icon: Mail, label: 'Email', value: 'support@veloxfood.test' },
  { icon: Phone, label: 'Hotline', value: '1900 1234 (8:00 – 22:00)' },
  { icon: MapPin, label: 'Văn phòng', value: 'Khu công nghệ, ĐHQG Hà Nội' },
  { icon: Clock, label: 'Giờ phục vụ', value: 'Cả tuần, 10:00 – 22:00' },
];

export default function ContactPage() {
  return (
    <div className="mx-auto w-full max-w-3xl px-4 py-12">
      <h1 className="font-serif text-2xl font-bold">Liên hệ</h1>
      <p className="text-muted-foreground mt-1.5 text-sm">
        Cần hỗ trợ về đơn hàng hay hợp tác mở gian hàng? Liên hệ với chúng tôi qua các kênh dưới đây.
      </p>
      <div className="mt-8 grid gap-4 sm:grid-cols-2">
        {CHANNELS.map((c) => (
          <div key={c.label} className="border-border flex items-start gap-3 rounded-xl border p-5">
            <div className="bg-primary/10 text-primary flex h-10 w-10 shrink-0 items-center justify-center rounded-lg">
              <c.icon className="h-5 w-5" />
            </div>
            <div>
              <p className="text-muted-foreground text-xs uppercase tracking-wide">{c.label}</p>
              <p className="mt-0.5 font-medium">{c.value}</p>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
