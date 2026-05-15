import type { Metadata } from "next";
import { routing } from '@/i18n/routing';
import { notFound } from 'next/navigation';
import './globals.css';

export function generateStaticParams() {
  return routing.locales.map((locale) => ({ locale }));
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html suppressHydrationWarning >
      <body className="min-h-screen" suppressHydrationWarning>
        {children}
      </body>
    </html>
  );
}
