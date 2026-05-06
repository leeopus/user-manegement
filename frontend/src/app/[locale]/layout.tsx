import type { Metadata } from "next";
import { NextIntlClientProvider } from 'next-intl';
import { getMessages, getTranslations } from 'next-intl/server';
import { notFound } from 'next/navigation';
import { routing } from '@/i18n/routing';
import { LocalePreference } from '@/components/locale-preference';
import { AuthProvider } from '@/lib/auth-provider';
import { ErrorBoundary } from '@/components/error-boundary';

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

  // 添加多语言SEO支持
  const siteUrl = process.env.NEXT_PUBLIC_SITE_URL || 'https://localhost:3000'
  return {
    title: t('title'),
    description: t('description'),
    alternates: {
      languages: {
        'zh-CN': `${siteUrl}/zh`,
        'en': `${siteUrl}/en`,
        'x-default': `${siteUrl}/zh`
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
        <ErrorBoundary>
          <AuthProvider>
            <LocalePreference />
            {children}
          </AuthProvider>
        </ErrorBoundary>
      </NextIntlClientProvider>
    </div>
  );
}
