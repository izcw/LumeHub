import crypto from 'node:crypto'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const dataRoot = path.resolve(__dirname, '../data')
const resourceRoot = path.join(dataRoot, 'resource')
const categoriesPath = path.join(dataRoot, 'categories.json')

const FOLDER_RENAME_MAP = new Map([
  ['Kasumi', 'kasumi'],
  ['jingxuan_3', 'jingxuan'],
  ['linshi_2', 'linshi'],
  ['jilu_1', 'jilu'],
  ['lvyou_4', 'lvyou'],
  ['sheying_5', 'sheying'],
  ['meishi_6', 'meishi'],
  ['shenghuo_7', 'shenghuo'],
  ['qita_8', 'qita'],
])

const IMAGE_EXTS = new Set(['.jpg', '.jpeg', '.png', '.webp', '.gif', '.bmp', '.avif'])
const CANONICAL_ID_RE = /^[a-f0-9]{10,16}_[0-9]{8}$/i

function ensureDir(p) {
  fs.mkdirSync(p, { recursive: true })
}

function normalizeExt(file) {
  const ext = path.extname(file || '').toLowerCase()
  return ext || '.bin'
}

function toDateCompact(raw) {
  const d = new Date(raw || '')
  if (!Number.isNaN(d.getTime())) {
    const y = d.getUTCFullYear()
    const m = String(d.getUTCMonth() + 1).padStart(2, '0')
    const day = String(d.getUTCDate()).padStart(2, '0')
    return `${y}${m}${day}`
  }
  return '19700101'
}

function safeMove(fromAbs, toAbs) {
  if (fromAbs === toAbs) return
  ensureDir(path.dirname(toAbs))
  if (!fs.existsSync(fromAbs)) return
  if (fs.existsSync(toAbs)) {
    fs.rmSync(toAbs, { force: true })
  }
  fs.renameSync(fromAbs, toAbs)
}

function stableToken(input) {
  return crypto.createHash('sha1').update(input).digest('hex').slice(0, 12)
}

function nextCanonicalID(folderKey, oldItem, oldFilename, seq, used) {
  const ext = normalizeExt(oldFilename)
  const date = toDateCompact(oldItem.uploadedAt || oldItem.updatedAt)
  const oldID = String(oldItem.id || '').trim()
  if (CANONICAL_ID_RE.test(oldID)) {
    const expect = `original/${oldID}${ext}`
    if (oldFilename === expect && !used.has(oldID)) {
      used.add(oldID)
      return oldID
    }
  }
  const seedBase = `${folderKey}|${oldFilename}|${oldItem.uploadedAt || ''}|${oldItem.updatedAt || ''}|${seq}`
  let token = stableToken(seedBase)
  let id = `${token}_${date}`
  let n = 1
  while (used.has(id)) {
    token = stableToken(`${seedBase}|${n}`).slice(0, 12)
    id = `${token}_${date}`
    n += 1
  }
  used.add(id)
  return id
}

function removeEmptyDevDir(folderDir) {
  const devDir = path.join(folderDir, '_dev')
  if (!fs.existsSync(devDir)) return
  const children = fs.readdirSync(devDir)
  if (children.length === 0) {
    fs.rmSync(devDir, { recursive: true, force: true })
  }
}

function collectFilesRecursive(baseDir, rel = '') {
  const abs = rel ? path.join(baseDir, rel) : baseDir
  const out = []
  for (const e of fs.readdirSync(abs, { withFileTypes: true })) {
    const relPath = rel ? path.join(rel, e.name) : e.name
    const norm = relPath.replaceAll('\\', '/')
    if (e.isDirectory()) {
      if (norm === 'thumb') continue
      out.push(...collectFilesRecursive(baseDir, norm))
      continue
    }
    if (norm.toLowerCase() === 'items.json') continue
    out.push(norm)
  }
  return out
}

function normalizeItemsForFolder(folderKey) {
  const folderDir = path.join(resourceRoot, folderKey)
  const itemsPath = path.join(folderDir, 'items.json')
  if (!fs.existsSync(itemsPath)) return

  const doc = JSON.parse(fs.readFileSync(itemsPath, 'utf8') || '{"items":[]}')
  if (!Array.isArray(doc.items)) doc.items = []

  ensureDir(path.join(folderDir, 'original'))
  ensureDir(path.join(folderDir, 'thumb'))

  const sorted = [...doc.items].sort((a, b) => {
    const sa = Number(a?.sort || 0)
    const sb = Number(b?.sort || 0)
    if (sa !== sb) return sa - sb
    return String(a?.filename || '').localeCompare(String(b?.filename || ''), undefined, {
      sensitivity: 'base',
    })
  })

  const usedIDs = new Set()
  const processedPaths = new Set()
  const nextItems = []
  let seq = 0
  for (const oldItem of sorted) {
    if (!oldItem || typeof oldItem !== 'object') continue
    const oldFilenameRaw = String(oldItem.filename || '').trim().replaceAll('\\', '/')
    if (!oldFilenameRaw) continue
    seq += 1

    const oldFilename = oldFilenameRaw.startsWith('./') ? oldFilenameRaw.slice(2) : oldFilenameRaw
    processedPaths.add(oldFilename)
    const ext = normalizeExt(oldFilename)
    const id = nextCanonicalID(folderKey, oldItem, oldFilename, seq, usedIDs)
    const newFilename = `original/${id}${ext}`
    const linkName = `${id}${ext}`

    const fromAbs = path.join(folderDir, oldFilename)
    const toAbs = path.join(folderDir, newFilename)
    safeMove(fromAbs, toAbs)
    processedPaths.add(newFilename)

    let thumbnail = ''
    if (IMAGE_EXTS.has(ext)) {
      const oldThumb = String(oldItem.thumbnail || '').trim().replaceAll('\\', '/')
      const thumbExt = ext === '.png' ? '.png' : '.jpg'
      const newThumb = `thumb/${id}${thumbExt}`
      if (oldThumb) {
        const oldThumbAbs = path.join(folderDir, oldThumb)
        const newThumbAbs = path.join(folderDir, newThumb)
        safeMove(oldThumbAbs, newThumbAbs)
      }
      if (fs.existsSync(path.join(folderDir, newThumb))) {
        thumbnail = newThumb
      }
    }

    nextItems.push({
      id,
      sort: seq * 10,
      ...(oldItem.masonryCol ? { masonryCol: oldItem.masonryCol } : {}),
      ...(oldItem.masonryRow ? { masonryRow: oldItem.masonryRow } : {}),
      uploadedAt: oldItem.uploadedAt,
      updatedAt: oldItem.updatedAt,
      filename: newFilename,
      linkName,
      ...(thumbnail ? { thumbnail } : {}),
      ...(oldItem.title ? { title: oldItem.title } : {}),
      ...(Array.isArray(oldItem.tags) && oldItem.tags.length > 0 ? { tags: oldItem.tags } : {}),
    })
  }

  const allFiles = collectFilesRecursive(folderDir)
  for (const file of allFiles) {
    const norm = file.replaceAll('\\', '/')
    if (processedPaths.has(norm)) continue
    if (norm.startsWith('original/')) continue
    seq += 1
    const fakeOld = {
      id: '',
      uploadedAt: '',
      updatedAt: '',
      tags: [],
    }
    const id = nextCanonicalID(folderKey, fakeOld, norm, seq, usedIDs)
    const ext = normalizeExt(norm)
    const newFilename = `original/${id}${ext}`
    safeMove(path.join(folderDir, norm), path.join(folderDir, newFilename))
    nextItems.push({
      id,
      sort: seq * 10,
      filename: newFilename,
      linkName: `${id}${ext}`,
      tags: [ext.startsWith('.') ? ext.slice(1) : ext],
    })
  }

  fs.writeFileSync(itemsPath, `${JSON.stringify({ items: nextItems }, null, 2)}\n`, 'utf8')
  removeEmptyDevDir(folderDir)
}

function renameResourceFolders() {
  for (const [from, to] of FOLDER_RENAME_MAP.entries()) {
    const fromDir = path.join(resourceRoot, from)
    const toDir = path.join(resourceRoot, to)
    if (!fs.existsSync(fromDir)) continue
    if (from === to) continue
    if (from.toLowerCase() === to.toLowerCase()) {
      const tempDir = path.join(resourceRoot, `${to}__tmp_rename__`)
      if (fs.existsSync(tempDir)) fs.rmSync(tempDir, { recursive: true, force: true })
      fs.renameSync(fromDir, tempDir)
      fs.renameSync(tempDir, toDir)
      continue
    }
    if (fs.existsSync(toDir)) {
      throw new Error(`target folder already exists: ${to}`)
    }
    fs.renameSync(fromDir, toDir)
  }
}

function patchCategories() {
  if (!fs.existsSync(categoriesPath)) return
  const doc = JSON.parse(fs.readFileSync(categoriesPath, 'utf8'))
  if (typeof doc.homeFolderKey === 'string' && FOLDER_RENAME_MAP.has(doc.homeFolderKey)) {
    doc.homeFolderKey = FOLDER_RENAME_MAP.get(doc.homeFolderKey)
  }
  const categories = Array.isArray(doc.categories) ? doc.categories : []
  for (const major of categories) {
    const subs = Array.isArray(major?.subcategories) ? major.subcategories : []
    for (const sub of subs) {
      if (typeof sub?.folderKey === 'string' && FOLDER_RENAME_MAP.has(sub.folderKey)) {
        sub.folderKey = FOLDER_RENAME_MAP.get(sub.folderKey)
      }
    }
  }
  fs.writeFileSync(categoriesPath, `${JSON.stringify(doc, null, 2)}\n`, 'utf8')
}

renameResourceFolders()
for (const entry of fs.readdirSync(resourceRoot, { withFileTypes: true })) {
  if (!entry.isDirectory()) continue
  normalizeItemsForFolder(entry.name)
}
patchCategories()
console.log('done: data reorganized')
