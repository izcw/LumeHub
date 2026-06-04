import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const resourceRoot = path.resolve(__dirname, '../data/resource')

const TARGET_FOLDERS = new Set(['jilu_1', 'jingxuan_3', 'linshi_2'])

function datePart(iso) {
  const d = new Date(iso || '')
  if (Number.isNaN(d.getTime())) return 'unknown-date'
  const y = d.getUTCFullYear()
  const m = String(d.getUTCMonth() + 1).padStart(2, '0')
  const day = String(d.getUTCDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}

function extOf(filename) {
  return path.extname(String(filename || '').trim()) || ''
}

function uniqueName(seed, used) {
  let n = 1
  let cand = seed
  while (used.has(cand.toLowerCase())) {
    n += 1
    cand = `${seed}-${String(n).padStart(2, '0')}`
  }
  used.add(cand.toLowerCase())
  return cand
}

for (const folder of fs.readdirSync(resourceRoot, { withFileTypes: true })) {
  if (!folder.isDirectory()) continue
  if (!TARGET_FOLDERS.has(folder.name)) continue

  const dir = path.join(resourceRoot, folder.name)
  const itemsPath = path.join(dir, 'items.json')
  if (!fs.existsSync(itemsPath)) continue

  const doc = JSON.parse(fs.readFileSync(itemsPath, 'utf8') || '{"items":[]}')
  if (!Array.isArray(doc.items)) continue

  const used = new Set()
  for (const item of doc.items) {
    const existing = String(item?.linkName || '').trim()
    if (existing) used.add(existing.toLowerCase())
  }

  let changed = false
  let mediaSeq = 0
  for (const item of doc.items) {
    if (!item || typeof item !== 'object') continue
    const filename = String(item.filename || '').trim()
    if (!filename) continue
    if (filename.startsWith('_dev/')) continue

    mediaSeq += 1
    const ext = extOf(filename)
    const date = datePart(item.uploadedAt || item.updatedAt)
    const seq = String(mediaSeq).padStart(4, '0')
    const seed = `${folder.name}-${seq}-${date}${ext}`
    const linkName = uniqueName(seed, used)
    if (item.linkName !== linkName) {
      item.linkName = linkName
      changed = true
    }
  }

  if (changed) {
    fs.writeFileSync(itemsPath, `${JSON.stringify(doc, null, 2)}\n`, 'utf8')
    console.log(`normalized linkName: ${folder.name}`)
  } else {
    console.log(`no change: ${folder.name}`)
  }
}
