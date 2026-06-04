import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const resourceRoot = path.resolve(__dirname, '../data/resource')

function normalizeLinkName(filename) {
  const base = path.basename(String(filename || '').trim())
  if (!base) return ''
  return base.replace(/\s+/g, '-')
}

for (const folder of fs.readdirSync(resourceRoot, { withFileTypes: true })) {
  if (!folder.isDirectory()) continue
  const folderDir = path.join(resourceRoot, folder.name)
  const itemsPath = path.join(folderDir, 'items.json')
  if (!fs.existsSync(itemsPath)) continue

  const raw = fs.readFileSync(itemsPath, 'utf8')
  const doc = JSON.parse(raw || '{"items":[]}')
  if (!Array.isArray(doc.items)) continue

  let changed = false
  for (const item of doc.items) {
    if (!item || typeof item !== 'object') continue
    const filename = String(item.filename || '').trim()
    if (!filename) continue

    if (!item.linkName || !String(item.linkName).trim()) {
      item.linkName = normalizeLinkName(filename)
      changed = true
    }

    if (!item.thumbnail || !String(item.thumbnail).trim()) {
      const base = path.basename(String(filename).trim().replaceAll('\\', '/'))
      const ext = path.extname(base).toLowerCase()
      const stem = path.basename(base, ext)
      if (!stem) continue
      const canonicalJpg = path.join('thumb', `${stem}.jpg`).replaceAll('\\', '/')
      const canonicalPng = path.join('thumb', `${stem}.png`).replaceAll('\\', '/')
      const legacyRel = path.join('thumb', `${stem}-thumbnail.jpg`).replaceAll('\\', '/')
      if (ext === '.png' && fs.existsSync(path.join(folderDir, canonicalPng))) {
        item.thumbnail = canonicalPng
        changed = true
      } else if (fs.existsSync(path.join(folderDir, canonicalJpg))) {
        item.thumbnail = canonicalJpg
        changed = true
      } else if (fs.existsSync(path.join(folderDir, legacyRel))) {
        item.thumbnail = legacyRel
        changed = true
      }
    }
  }

  if (changed) {
    fs.writeFileSync(itemsPath, JSON.stringify(doc, null, 2) + '\n', 'utf8')
    console.log(`updated ${folder.name}`)
  }
}
