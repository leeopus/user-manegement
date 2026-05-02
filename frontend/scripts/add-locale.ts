/**
 * 添加新语言脚本
 *
 * 用法: npm run i18n:add-locale fr
 *
 * 自动：
 * 1. 创建新语言的目录结构
 * 2. 复制所有模块的翻译文件
 * 3. 使用占位符标记未翻译的项
 */

import fs from 'fs'
import path from 'path'
import { fileURLToPath } from 'url'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)

// 获取命令行参数
const args = process.argv.slice(2)
const newLocale = args[0]

if (!newLocale) {
  console.error('❌ 请指定要添加的语言代码')
  console.error('用法: npm run i18n:add-locale <locale>')
  console.error('示例: npm run i18n:add-locale fr')
  process.exit(1)
}

// 配置
const BASE_LOCALE = 'zh'
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
]

/**
 * 递归处理翻译对象，添加 TODO 标记
 */
function processTranslations(obj: unknown, locale: string, module: string): unknown {
  if (!obj || typeof obj !== 'object') {
    return obj
  }

  if (Array.isArray(obj)) {
    return obj.map(item => processTranslations(item, locale, module))
  }

  const result: Record<string, unknown> = {}

  for (const [key, value] of Object.entries(obj)) {
    if (typeof value === 'object' && value !== null && !Array.isArray(value)) {
      result[key] = processTranslations(value, locale, module)
    } else if (typeof value === 'string') {
      // 添加翻译 TODO 标记
      result[key] = `[TODO: ${locale}/${module}/${key}] ${value}`
    } else {
      result[key] = value
    }
  }

  return result
}

/**
 * 主函数
 */
function main() {
  console.log(`\n🌍 添加新语言: ${newLocale}\n`)

  // 1. 创建目录
  const localeDir = path.join(MESSAGES_DIR, newLocale)
  if (fs.existsSync(localeDir)) {
    console.log(`⚠️  目录 ${localeDir} 已存在`)
    console.log('如要重新生成，请先删除该目录\n')
    process.exit(1)
  }

  fs.mkdirSync(localeDir, { recursive: true })
  console.log(`✅ 创建目录: ${localeDir}`)

  // 2. 复制并处理翻译文件
  let fileCount = 0
  let keyCount = 0

  for (const module of MODULES) {
    const basePath = path.join(MESSAGES_DIR, BASE_LOCALE, `${module}.json`)
    const newPath = path.join(localeDir, `${module}.json`)

    if (!fs.existsSync(basePath)) {
      console.log(`⚠️  跳过 ${module}: 基础文件不存在`)
      continue
    }

    // 读取基础翻译
    const baseTranslations = JSON.parse(fs.readFileSync(basePath, 'utf-8'))

    // 处理翻译（添加 TODO 标记）
    const newTranslations = processTranslations(baseTranslations, newLocale, module)

    // 写入新文件
    fs.writeFileSync(
      newPath,
      JSON.stringify(newTranslations, null, 2) + '\n',
      'utf-8'
    )

    fileCount++
    const keys = Object.keys(baseTranslations).length
    keyCount += keys

    console.log(`✅ ${module}.json (${keys} 个键)`)
  }

  // 3. 输出总结
  console.log(`\n✨ 成功添加语言: ${newLocale}`)
  console.log(`   - 创建文件: ${fileCount} 个`)
  console.log(`   - 翻译键总数: ${keyCount} 个`)
  console.log(`   - 基础语言: ${BASE_LOCALE}`)
  console.log(`\n📝 下一步:`)
  console.log(`   1. 搜索并替换所有 "[TODO: ${newLocale}" 标记`)
  console.log(`   2. 运行检查: npm run check-i18n`)
  console.log(`   3. 更新 i18n 配置文件添加新语言\n`)
}

main()
