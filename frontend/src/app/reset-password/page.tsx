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

  // 重定向到带语言的页面，保留token参数
  const redirectUrl = token
    ? `/${targetLang}/reset-password?token=${token}`
    : `/${targetLang}/reset-password`

  redirect(redirectUrl)

  // 显示加载状态（以防重定向失败）
  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-gray-50 to-gray-100 px-4">
      <div className="text-center">
        <div className="animate-spin rounded-full h-12 w-12 border-b-4 border-blue-600 mx-auto mb-4"></div>
        <p className="text-gray-600">正在跳转到密码重置页面...</p>
      </div>
    </div>
  )
}
