import { redirect } from 'next/navigation'
import { headers } from 'next/headers'

export default async function UniversalResetPasswordPage({
  searchParams,
}: {
  searchParams: Promise<{ token?: string }>
}) {
  // 从URL参数获取token (searchParams现在是Promise)
  const params = await searchParams
  const token = params.token

  // 检测用户的语言偏好（按优先级）
  // 1. 浏览器语言设置
  // 2. localStorage 中的用户偏好
  // 3. 默认中文
  let targetLang = 'zh' // 默认中文

  try {
    // 简单的语言检测逻辑 (headers()现在返回Promise)
    const headersList = await headers()
    const acceptLanguage = headersList.get('accept-language') || ''

    if (acceptLanguage.toLowerCase().startsWith('en')) {
      targetLang = 'en'
    }
  } catch (error) {
    // 如果无法获取headers，使用默认语言
    console.log('无法获取浏览器语言设置，使用默认中文')
  }

  const redirectUrl = token
    ? `/${targetLang}/reset-password#token=${token}`
    : `/${targetLang}/reset-password`

  redirect(redirectUrl)
}
