import type { Metadata } from 'next';
import { Karla, Playfair_Display_SC } from 'next/font/google';
import './globals.css';
import { Providers } from './providers';

const karla = Karla({
  variable: '--font-karla',
  subsets: ['latin', 'latin-ext'],
  weight: ['300', '400', '500', '600', '700'],
  display: 'swap',
});

const playfair = Playfair_Display_SC({
  variable: '--font-playfair',
  subsets: ['latin'],
  weight: ['400', '700'],
  display: 'swap',
});

export const metadata: Metadata = {
  title: {
    default: 'VeloxFood — Đặt món nhanh, chuẩn vị',
    template: '%s · VeloxFood',
  },
  description: 'VeloxFood — nền tảng đặt món ăn cho project Hệ thống phân tán.',
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html
      lang="vi"
      className={`${karla.variable} ${playfair.variable} h-full antialiased`}
      suppressHydrationWarning
    >
      <body className="bg-background text-foreground min-h-full font-sans" suppressHydrationWarning>
        <Providers>{children}</Providers>
      </body>
    </html>
  );
}
