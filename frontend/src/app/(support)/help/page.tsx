import type { Metadata } from 'next';
import { HelpCircle } from 'lucide-react';

export const metadata: Metadata = { title: 'Trung tâm trợ giúp · VeloxFood' };

const FAQS = [
  {
    q: 'Làm sao để đặt món?',
    a: 'Chọn cửa hàng → thêm món vào giỏ → vào Thanh toán, chọn ca giao và phương thức thanh toán, rồi bấm Đặt hàng.',
  },
  {
    q: 'Ca giao (suất) hoạt động thế nào?',
    a: 'Mỗi cửa hàng mở vài ca giao trong ngày (ví dụ 11:00, 17:30, 22:00). Mỗi ca có số suất giới hạn cho từng món. Đơn phải đặt trước giờ cắt đơn của ca.',
  },
  {
    q: 'Tôi thanh toán bằng những hình thức nào?',
    a: 'Tiền mặt khi nhận (COD), Ví VeloxFood, hoặc MoMo. Với ví, số dư được trừ ngay khi đặt; nếu chưa đủ bạn có thể nạp thêm trong mục Ví.',
  },
  {
    q: 'Tôi theo dõi đơn ở đâu?',
    a: 'Vào Tài khoản → Đơn hàng để xem trạng thái realtime: đã xác nhận, đang chuẩn bị, đang giao, đã giao.',
  },
  {
    q: 'Tôi có thể tự đến lấy không?',
    a: 'Với cửa hàng hỗ trợ "Tự đến lấy", chọn hình thức này ở bước thanh toán. Bạn sẽ nhận mã PIN để nhận món tại quầy.',
  },
];

export default function HelpPage() {
  return (
    <div className="mx-auto w-full max-w-3xl px-4 py-12">
      <div className="mb-8 flex items-center gap-3">
        <div className="bg-primary/10 text-primary flex h-11 w-11 items-center justify-center rounded-xl">
          <HelpCircle className="h-6 w-6" />
        </div>
        <div>
          <h1 className="font-serif text-2xl font-bold">Trung tâm trợ giúp</h1>
          <p className="text-muted-foreground text-sm">Câu hỏi thường gặp về đặt món trên VeloxFood.</p>
        </div>
      </div>
      <div className="space-y-4">
        {FAQS.map((f) => (
          <div key={f.q} className="border-border rounded-xl border p-5">
            <h2 className="font-semibold">{f.q}</h2>
            <p className="text-muted-foreground mt-1.5 text-sm leading-relaxed">{f.a}</p>
          </div>
        ))}
      </div>
    </div>
  );
}
