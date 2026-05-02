"use client"

import { LanguageSwitcher } from "@/components/language-switcher"

export default function AuthLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <div className="relative">
      {/* Language Switcher - Top Right */}
      <div className="absolute top-4 right-4 z-10">
        <LanguageSwitcher />
      </div>

      {/* Main Content */}
      {children}
    </div>
  )
}
