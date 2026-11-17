// Translations for the v3 frontend come from the same TOML files as the
// panel's server-side messages (web/translation). The frontend takes only its
// own namespaces from them; the other tables belong to the old pages.
//
// Run directly (node build/i18n.ts) it checks that every language has the
// same keys, that every message compiles as vue-i18n reads it, that a key
// takes the same named values in every language and that every key is used
// somewhere in src/, and refreshes the generated key type. Vite uses the same
// functions to serve the messages.

import { readdirSync, readFileSync, writeFileSync } from 'node:fs'
import { dirname, join, relative } from 'node:path'
import { fileURLToPath } from 'node:url'
import { baseCompile, type CompileError } from '@intlify/message-compiler'
import { parse } from 'smol-toml'

const root = join(dirname(fileURLToPath(import.meta.url)), '..')
export const translationDir = join(root, '..', 'web', 'translation')
const srcDir = join(root, 'src')
const schemaFile = join(srcDir, 'i18n', 'schema.gen.ts')

/** Locale id (also the value of the panel's lang cookie) → TOML file. */
export const localeFiles = {
  'en-US': 'translate.en_US.toml',
  'ru-RU': 'translate.ru_RU.toml',
} as const
export type Locale = keyof typeof localeFiles

/** Top-level tables the v3 frontend owns. */
export const namespaces = ['common', 'nav', 'auth', 'dashboard', 'settings', 'xray', 'inbounds', 'sub', 'units', 'kit'] as const

type Tree = { [key: string]: string | Tree }

export function loadMessages(locale: Locale): Tree {
  const all = parse(readFileSync(join(translationDir, localeFiles[locale]), 'utf8')) as Tree
  const out: Tree = {}
  for (const ns of namespaces) {
    const table = all[ns]
    if (table && typeof table === 'object') out[ns] = table
  }
  return out
}

function flatten(tree: Tree, prefix = ''): string[] {
  return Object.entries(tree).flatMap(([k, v]) => (typeof v === 'string' ? [prefix + k] : flatten(v, `${prefix}${k}.`)))
}

/** Differences in keys between the languages, as readable lines. */
export function parityErrors(): string[] {
  const [first, ...rest] = Object.keys(localeFiles) as Locale[]
  const base = new Set(flatten(loadMessages(first!)))
  const errors: string[] = []
  for (const locale of rest) {
    const keys = new Set(flatten(loadMessages(locale)))
    for (const k of base) if (!keys.has(k)) errors.push(`${localeFiles[locale]}: missing ${k}`)
    for (const k of keys) if (!base.has(k)) errors.push(`${localeFiles[locale]}: ${k} is not in ${localeFiles[first!]}`)
  }
  return errors
}

function entries(tree: Tree, prefix = ''): [string, string][] {
  return Object.entries(tree).flatMap(([k, v]) => (typeof v === 'string' ? [[prefix + k, v] as [string, string]] : entries(v, `${prefix}${k}.`)))
}

/**
 * Messages vue-i18n cannot read, as readable lines: in a message "{…}" is a
 * value put in and "@" starts a link to another message, so a literal brace
 * or at sign is written {'{'} or {'@'}. Such a message throws when shown and
 * stops the page it is on from updating.
 */
export function syntaxErrors(): string[] {
  const errors: string[] = []
  for (const locale of Object.keys(localeFiles) as Locale[]) {
    for (const [key, text] of entries(loadMessages(locale))) {
      baseCompile(text, {
        location: false,
        onError: (e: CompileError) => errors.push(`${localeFiles[locale]}: ${key}: ${e.message}`),
      })
    }
  }
  return errors
}

/** The named values a message takes, e.g. n in "{n} entries"; literals such as {'@'} are not. */
export function placeholders(text: string): string[] {
  return [...new Set([...text.matchAll(/\{\s*([A-Za-z_]\w*)\s*\}/g)].map((m) => m[1]!))].sort()
}

/** Keys whose languages take different named values: one of them would show an empty gap. */
export function placeholderErrors(): string[] {
  const [first, ...rest] = Object.keys(localeFiles) as Locale[]
  const base = new Map(entries(loadMessages(first!)))
  const errors: string[] = []
  for (const locale of rest) {
    for (const [key, text] of entries(loadMessages(locale))) {
      const want = base.get(key)
      if (want === undefined) continue
      const a = placeholders(want).join(', ')
      const b = placeholders(text).join(', ')
      if (a !== b) errors.push(`${key}: ${localeFiles[first!]} takes {${a}}, ${localeFiles[locale]} takes {${b}}`)
    }
  }
  return errors
}

function sourceFiles(dir: string): string[] {
  return readdirSync(dir, { withFileTypes: true }).flatMap((e) => {
    const p = join(dir, e.name)
    if (e.isDirectory()) return sourceFiles(p)
    return /\.(vue|ts)$/.test(e.name) && p !== schemaFile ? [p] : []
  })
}

/** Keys that no source file mentions as a quoted string. */
export function unusedKeys(): string[] {
  const code = sourceFiles(srcDir).map((f) => readFileSync(f, 'utf8')).join('\n')
  const en = flatten(loadMessages('en-US'))
  return en.filter((k) => !code.includes(`'${k}'`) && !code.includes(`"${k}"`) && !code.includes(`\`${k}\``))
}

function schemaType(tree: Tree, indent = '  '): string {
  const lines = Object.entries(tree).map(([k, v]) =>
    typeof v === 'string' ? `${indent}${JSON.stringify(k)}: string` : `${indent}${JSON.stringify(k)}: {\n${schemaType(v, indent + '  ')}\n${indent}}`,
  )
  return lines.join('\n')
}

/** Writes the message type from the English file; returns true when it changed. */
export function writeSchema(): boolean {
  const text = `// Generated by build/i18n.ts from web/translation/${localeFiles['en-US']}. Do not edit.\n\nexport interface MessageSchema {\n${schemaType(loadMessages('en-US'))}\n}\n`
  let old = ''
  try {
    old = readFileSync(schemaFile, 'utf8')
  } catch {
    // first run
  }
  if (old === text) return false
  writeFileSync(schemaFile, text)
  return true
}

if (process.argv[1] && fileURLToPath(import.meta.url) === process.argv[1]) {
  const errors = [...parityErrors(), ...syntaxErrors(), ...placeholderErrors(), ...unusedKeys().map((k) => `unused key ${k}`)]
  if (writeSchema()) console.log(`updated ${relative(root, schemaFile)}`)
  if (errors.length) {
    console.error(errors.join('\n'))
    process.exit(1)
  }
  console.log('translations ok')
}
