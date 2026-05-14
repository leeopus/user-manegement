/**
 * 翻译完整性检查脚本
 *
 * 检查所有语言的翻译是否完整，确保：
 * 1. 所有语言都有相同的翻译键
 * 2. 所有翻译键都有对应的值
 * 3. 翻译值不为空
 */

import fs from 'fs'
import path from 'path'
import { fileURLToPath } from 'url'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)

// 配置
const LOCALES = ['zh', 'en'] as const
const MESSAGES_DIR = path.join(__dirname, '../messages')
const MODULES = [
  'common',
  'errors',
  'auth',
  'validation',
  'profile',
  'dashboard',
  'users',
  'passwordStrength',
  'clearData',
] as const

interface TranslationIssue {
  type: 'missing' | 'empty' | 'mismatch'
  locale: string
  module: string
  key?: string
  expected?: string
  actual?: string
}

const issues: TranslationIssue[] = []

/**
 * 深度获取对象的所有键
 */
function getAllKeys(obj: unknown, prefix = ''): string[] {
  if (!obj || typeof obj !== 'object') {
    return []
  }

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

/**
 * 检查单个模块的翻译
 */
function checkModule(module: string) {
  // 加载基础语言（中文）的翻译
  const basePath = path.join(MESSAGES_DIR, 'zh', `${module}.json`)
  if (!fs.existsSync(basePath)) {
    issues.push({
      type: 'missing',
      locale: 'zh',
      module,
      key: undefined,
      expected: '文件',
      actual: '文件不存在',
    })
    return
  }

  const baseTranslations = JSON.parse(fs.readFileSync(basePath, 'utf-8'))
  const baseKeys = getAllKeys(baseTranslations)

  // 检查其他语言
  for (const locale of LOCALES) {
    if (locale === 'zh') continue

    const localePath = path.join(MESSAGES_DIR, locale, `${module}.json`)

    if (!fs.existsSync(localePath)) {
      issues.push({
        type: 'missing',
        locale,
        module,
        key: undefined,
        expected: '文件',
        actual: '文件不存在',
      })
      continue
    }

    const localeTranslations = JSON.parse(fs.readFileSync(localePath, 'utf-8'))
    const localeKeys = getAllKeys(localeTranslations)

    // 检查缺失的键
    for (const key of baseKeys) {
      if (!localeKeys.includes(key)) {
        issues.push({
          type: 'missing',
          locale,
          module,
          key,
          expected: `zh/${module}.json 中存在`,
          actual: '缺失',
        })
      }
    }

    // 检查多余的键
    for (const key of localeKeys) {
      if (!baseKeys.includes(key)) {
        issues.push({
          type: 'mismatch',
          locale,
          module,
          key,
          expected: '不存在',
          actual: '多余的键',
        })
      }
    }

    // 检查空值
    for (const key of localeKeys) {
      const value = localeTranslations[key as keyof typeof localeTranslations]
      if (value === '' || value === null) {
        issues.push({
          type: 'empty',
          locale,
          module,
          key,
          expected: '翻译内容',
          actual: '空值',
        })
      }
    }
  }
}

/**
 * 主函数
 */
function main() {
  console.log('🔍 检查翻译完整性...\n')

  // 检查所有模块
  for (const module of MODULES) {
    checkModule(module)
  }

  // 输出结果
  if (issues.length === 0) {
    console.log('✅ 所有翻译检查通过！')
    console.log(`   - 语言: ${LOCALES.join(', ')}`)
    console.log(`   - 模块: ${MODULES.length} 个`)
    console.log(`   - 所有翻译键完整且一致`)
    process.exit(0)
  }

  console.log(`❌ 发现 ${issues.length} 个问题:\n`)

  // 按类型分组显示
  const byType = issues.reduce((acc, issue) => {
    if (!acc[issue.type]) {
      acc[issue.type] = []
    }
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

  console.log('💡 建议:')
  console.log('   1. 补充缺失的翻译')
  console.log('   2. 移除多余的翻译键')
  console.log('   3. 填充空值翻译')
  console.log('')

  process.exit(1)
}

main()
