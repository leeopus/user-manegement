import { redirect } from 'next/navigation'
import { headers } from 'next/headers'

export default async function UniversalVerifyEmailPage({
  searchParams,
}: {
  searchParams: Promise<{ token?: string }>
}) {
  const params = await searchParams
  const token = params.token

  let targetLang = 'zh'

  try {
    const headersList = await headers()
    const acceptLanguage = headersList.get('accept-language') || ''

    if (acceptLanguage.toLowerCase().startsWith('en')) {
      targetLang = 'en'
    }
  } catch {
    // fallback to default
  }

  const redirectUrl = token
    ? `/${targetLang}/verify-email?token=${token}`
    : `/${targetLang}/verify-email`

  redirect(redirectUrl)
}
