"use client"

import { useEffect } from "react"
import { useRouter } from "next/navigation"
import { useLocale } from "next-intl"

export default function Home() {
  const router = useRouter()
  const locale = useLocale()

  useEffect(() => {
    router.push(`/${locale}/login`)
  }, [router, locale])

  return null
}
