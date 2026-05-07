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
  response.headers.set('X-Frame-Options', 'SAMEORIGIN');
  response.headers.set('X-XSS-Protection', '0');
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
      process.env.NODE_ENV === 'production'
        ? "script-src 'self'"
        : "script-src 'self' 'unsafe-inline' 'unsafe-eval'",
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

// UX-only optimization: checks JWT format (three dot-separated base64url segments).
// This is NOT a security boundary — a fake cookie like "a.b.c" will bypass this check.
// Real authentication is enforced by the backend via HttpOnly cookies + /auth/me validation.
function isValidJWTFormat(token: string): boolean {
  const parts = token.split('.');
  if (parts.length !== 3) return false;
  // 每段必须是非空的 base64url 字符串
  const base64urlPattern = /^[A-Za-z0-9_-]+$/;
  return parts.every(part => part.length > 0 && base64urlPattern.test(part));
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
  // 验证 cookie 不仅存在，而且包含格式有效的 JWT（三段 base64 结构）
  const accessToken = request.cookies.get('access_token')?.value;
  const refreshToken = request.cookies.get('refresh_token')?.value;
  const hasAuth = !!(
    (accessToken && isValidJWTFormat(accessToken)) ||
    (refreshToken && isValidJWTFormat(refreshToken))
  );

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

export const config = {
  matcher: ['/', '/(zh|en)/:path*']
};
