/**
 * 翻译完整性检查脚本
 *
 * 自动扫描 messages/{locale}/ 目录，检查：
 * 1. 所有语言的翻译模块一致
 * 2. 所有翻译键完整且匹配
 * 3. 翻译值不为空
 */

import fs from 'fs'
import path from 'path'
import { fileURLToPath } from 'url'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)

const LOCALES = ['zh', 'en'] as const
const MESSAGES_DIR = path.join(__dirname, '../messages')

interface TranslationIssue {
  type: 'missing' | 'empty' | 'mismatch'
  locale: string
  module: string
  key?: string
  expected?: string
  actual?: string
}

const issues: TranslationIssue[] = []

// Auto-discover modules from the base locale (zh)
const zhDir = path.join(MESSAGES_DIR, 'zh')
const MODULES = fs.readdirSync(zhDir)
  .filter(f => f.endsWith('.json'))
  .map(f => f.replace('.json', ''))
  .sort()

function getAllKeys(obj: unknown, prefix = ''): string[] {
  if (!obj || typeof obj !== 'object') return []

  const keys: string[] = []
  const record = obj as Record<string, unknown>

  for (const key in record) {
    const fullKey = prefix ? `${prefix}.${key}` : key
    if (typeof record[key] === 'object' && record[key] !== null && !Array.isArray(record[key])) {
      keys.push(...getAllKeys(record[key], fullKey))
    } else {
      keys.push(fullKey)
    }
  }

  return keys.sort()
}

function checkModule(module: string) {
  const basePath = path.join(MESSAGES_DIR, 'zh', `${module}.json`)
  if (!fs.existsSync(basePath)) {
    issues.push({ type: 'missing', locale: 'zh', module, expected: '文件', actual: '文件不存在' })
    return
  }

  const baseTranslations = JSON.parse(fs.readFileSync(basePath, 'utf-8'))
  const baseKeys = getAllKeys(baseTranslations)

  for (const locale of LOCALES) {
    if (locale === 'zh') continue

    const localePath = path.join(MESSAGES_DIR, locale, `${module}.json`)

    if (!fs.existsSync(localePath)) {
      issues.push({ type: 'missing', locale, module, expected: '文件', actual: '文件不存在' })
      continue
    }

    const localeTranslations = JSON.parse(fs.readFileSync(localePath, 'utf-8'))
    const localeKeys = getAllKeys(localeTranslations)

    for (const key of baseKeys) {
      if (!localeKeys.includes(key)) {
        issues.push({ type: 'missing', locale, module, key, expected: `zh/${module}.json 中存在`, actual: '缺失' })
      }
    }

    for (const key of localeKeys) {
      if (!baseKeys.includes(key)) {
        issues.push({ type: 'mismatch', locale, module, key, expected: '不存在', actual: '多余的键' })
      }
    }

    for (const key of localeKeys) {
      const value = localeTranslations[key as keyof typeof localeTranslations]
      if (value === '' || value === null) {
        issues.push({ type: 'empty', locale, module, key, expected: '翻译内容', actual: '空值' })
      }
    }
  }
}

function main() {
  console.log('🔍 检查翻译完整性...\n')

  for (const module of MODULES) {
    checkModule(module)
  }

  if (issues.length === 0) {
    console.log('✅ 所有翻译检查通过！')
    console.log(`   - 语言: ${LOCALES.join(', ')}`)
    console.log(`   - 模块: ${MODULES.length} 个 (${MODULES.join(', ')})`)
    console.log(`   - 所有翻译键完整且一致`)
    process.exit(0)
  }

  console.log(`❌ 发现 ${issues.length} 个问题:\n`)

  const byType = issues.reduce((acc, issue) => {
    if (!acc[issue.type]) acc[issue.type] = []
    acc[issue.type].push(issue)
    return acc
  }, {} as Record<string, TranslationIssue[]>)

  for (const [type, typeIssues] of Object.entries(byType)) {
    const icon = type === 'missing' ? '❌' : type === 'empty' ? '⚠️' : '🔶'
    console.log(`${icon} ${type.toUpperCase()} (${typeIssues.length} 个)`)
    for (const issue of typeIssues) {
      if (issue.key) {
        console.log(`   ${issue.locale}/${issue.module}: ${issue.key}`)
      } else {
        console.log(`   ${issue.locale}/${issue.module}: ${issue.expected}`)
      }
    }
    console.log('')
  }

  process.exit(1)
}

main()
