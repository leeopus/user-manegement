import createMiddleware from 'next-intl/middleware';
import { NextRequest, NextResponse } from 'next/server';
import { routing } from './i18n/routing';

const PUBLIC_PATHS = ['/login', '/register', '/forgot-password', '/reset-password'];
const LOCALE_PATTERN = new RegExp(`^/(${routing.locales.join('|')})`);

function isPublicPath(pathname: string): boolean {
  for (const p of PUBLIC_PATHS) {
    if (pathname === p || pathname.startsWith(p + '/')) {
      return true;
    }
  }
  return false;
}

function addSecurityHeaders(response: NextResponse): void {
  response.headers.set('X-Content-Type-Options', 'nosniff');
  response.headers.set('X-Frame-Options', 'DENY');
  response.headers.set('X-XSS-Protection', '1; mode=block');
  response.headers.set('Referrer-Policy', 'strict-origin-when-cross-origin');
  response.headers.set('Permissions-Policy', 'camera=(), microphone=(), geolocation=()');
  response.headers.set('Cross-Origin-Opener-Policy', 'same-origin');
  response.headers.set('Cross-Origin-Resource-Policy', 'same-origin');

  const apiURL = process.env.NEXT_PUBLIC_API_URL || '';
  const connectSrc = apiURL ? `'self' ${apiURL}` : "'self'";

  // unsafe-inline required by Tailwind CSS runtime style injection
  response.headers.set(
    'Content-Security-Policy',
    [
      "default-src 'self'",
      "script-src 'self'",
      "style-src 'self' 'unsafe-inline'",
      "img-src 'self' data: blob:",
      "font-src 'self'",
      `connect-src ${connectSrc}`,
      "frame-ancestors 'none'",
      "base-uri 'self'",
      "form-action 'self'",
      "object-src 'none'",
    ].join('; ')
  );
  if (process.env.NODE_ENV === 'production') {
    response.headers.set('Strict-Transport-Security', 'max-age=63072000; includeSubDomains; preload');
  }
}

function authMiddleware(request: NextRequest) {
  const { pathname } = request.nextUrl;

  // 跳过静态资源、API 路由、_next 路径
  if (
    pathname.startsWith('/_next') ||
    pathname.startsWith('/api') ||
    pathname.includes('.') // static files
  ) {
    return null;
  }

  // 提取无 locale 前缀的路径
  const pathWithoutLocale = pathname.replace(LOCALE_PATTERN, '') || '/';

  // 检查是否有有效的 access_token 或 refresh_token cookie
  // 空字符串表示 cookie 已被清除（过期），不应视为已认证
  const accessToken = request.cookies.get('access_token')?.value;
  const refreshToken = request.cookies.get('refresh_token')?.value;
  const hasAuth = !!(accessToken || refreshToken);

  // 已登录用户访问公共页面，不做重定向（让客户端处理）
  // 未登录用户访问受保护页面，重定向到登录页
  if (!hasAuth && !isPublicPath(pathWithoutLocale)) {
    const locale = pathname.match(LOCALE_PATTERN)?.[1] || routing.defaultLocale;
    const loginUrl = new URL(`/${locale}/login`, request.url);
    const response = NextResponse.redirect(loginUrl);
    addSecurityHeaders(response);
    return response;
  }

  return null;
}

export default function middleware(request: NextRequest) {
  // 先执行认证检查
  const authResponse = authMiddleware(request);
  if (authResponse) {
    return authResponse;
  }

  // 再执行 i18n 路由
  const response = createMiddleware(routing)(request);
  addSecurityHeaders(response);
  return response;
}

const localeMatcher = routing.locales.join('|');

export const config = {
  matcher: ['/', `/(${localeMatcher})/:path*`]
};
