/**
 * 将缩略图统一为规范路径：thumb/{主干}.jpg（PNG 原图为 .png），无 -thumbnail 后缀。
 * 会尽量从重命名磁盘文件并写回 items.json。在 Backend 目录执行：node scripts/migrate-thumb-filenames.mjs
 */
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const resourceRoot = path.resolve(__dirname, '../data/resource')

const RASTER_EXTS = new Set(['.jpg', '.jpeg', '.png', '.webp', '.gif', '.bmp', '.avif'])

function canonicalThumbRel(filename) {
  const base = path.basename(String(filename || '').trim().replaceAll('\\', '/'))
  const ext = path.extname(base).toLowerCase()
  const stem = path.basename(base, ext)
  if (!stem || !RASTER_EXTS.has(ext)) return ''
  if (ext === '.png') return `thumb/${stem}.png`
  return `thumb/${stem}.jpg`
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
    const filename = String(item.filename || '').trim().replaceAll('\\', '/')
    if (!filename.startsWith('original/')) continue

    const thumbNew = canonicalThumbRel(filename)
    if (!thumbNew) continue

    const thumbOld = String(item.thumbnail || '').trim().replaceAll('\\', '/')
    const base = path.basename(String(filename || '').trim().replaceAll('\\', '/'))
    const stem = path.basename(base, path.extname(base))
    const candidates = new Set(
      [thumbOld, thumbNew, `thumb/${stem}-thumbnail.jpg`, `thumb/${path.basename(filename)}`].filter(
        Boolean,
      ),
    )

    const newAbs = path.join(folderDir, thumbNew)
    if (!fs.existsSync(newAbs)) {
      for (const c of candidates) {
        const oldAbs = path.join(folderDir, c)
        if (!c.startsWith('thumb/') || !fs.existsSync(oldAbs)) continue
        if (path.resolve(oldAbs) === path.resolve(newAbs)) break
        try {
          fs.renameSync(oldAbs, newAbs)
          break
        } catch (e) {
          console.warn(`[warn] ${folder.name} rename ${c} -> ${thumbNew}: ${e}`)
        }
      }
    }

    if (item.thumbnail !== thumbNew) {
      item.thumbnail = thumbNew
      changed = true
    }
  }

  if (changed) {
    fs.writeFileSync(itemsPath, JSON.stringify(doc, null, 2) + '\n', 'utf8')
    console.log('updated', folder.name)
  }
}

console.log('done: thumb filenames migrated')
