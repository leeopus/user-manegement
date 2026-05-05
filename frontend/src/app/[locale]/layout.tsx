import type { Metadata } from "next";
import { NextIntlClientProvider } from 'next-intl';
import { getMessages, getTranslations } from 'next-intl/server';
import { notFound } from 'next/navigation';
import { routing } from '@/i18n/routing';
import { LocalePreference } from '@/components/locale-preference';
import { AuthProvider } from '@/lib/auth-provider';

export function generateStaticParams() {
  return routing.locales.map((locale) => ({ locale }));
}

export async function generateMetadata({
  params,
}: {
  params: Promise<{ locale: string }>;
}): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: 'metadata' });

  return {
    title: t('title'),
    description: t('description'),
    // 添加多语言SEO支持
    alternates: {
      languages: {
        'zh-CN': `http://106.15.3.98:3000/zh`,
        'en': `http://106.15.3.98:3000/en`,
        'x-default': `http://106.15.3.98:3000/zh`
      }
    }
  };
}

export default async function LocaleLayout({
  children,
  params,
}: {
  children: React.ReactNode;
  params: Promise<{ locale: string }>;
}) {
  const { locale } = await params;

  // Ensure that the incoming `locale` is valid
  if (!routing.locales.includes(locale as any)) {
    notFound();
  }

  // Providing all messages to the client
  // side is the easiest way to get started
  const messages = await getMessages();

  return (
    <div className="h-full antialiased min-h-full flex flex-col">
      <NextIntlClientProvider messages={messages}>
        <AuthProvider>
          <LocalePreference />
          {children}
        </AuthProvider>
      </NextIntlClientProvider>
    </div>
  );
}
