import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const dataRoot = path.resolve(__dirname, '../data')
const categoriesPath = path.join(dataRoot, 'categories.json')
const resourceRoot = path.join(dataRoot, 'resource')

const ID_RE = /^[a-f0-9]{12}_[0-9]{8}$/i
const ALLOWED_FOLDER_ENTRIES = new Set(['items.json', 'original', 'thumb'])

const IMAGE_EXTS_THUMB = new Set(['.jpg', '.jpeg', '.png', '.webp', '.gif', '.bmp', '.avif'])

function expectedThumbnailRel(filename) {
  const base = path.basename(String(filename || '').trim().replaceAll('\\', '/'))
  const ext = path.extname(base).toLowerCase()
  const stem = path.basename(base, ext)
  if (!stem) return ''
  if (ext === '.png') return `thumb/${stem}.png`
  if (IMAGE_EXTS_THUMB.has(ext)) return `thumb/${stem}.jpg`
  return ''
}

function fail(msg) {
  console.error(`[FAIL] ${msg}`)
  hasError = true
}

let hasError = false

if (!fs.existsSync(categoriesPath)) {
  fail('missing Backend/data/categories.json')
  process.exit(1)
}
if (!fs.existsSync(resourceRoot)) {
  fail('missing Backend/data/resource')
  process.exit(1)
}

const categoriesDoc = JSON.parse(fs.readFileSync(categoriesPath, 'utf8'))
if ('homeFolderKey' in categoriesDoc) {
  fail('categories.json should not contain homeFolderKey (homepage is first by sort)')
}

const categories = Array.isArray(categoriesDoc.categories) ? categoriesDoc.categories : []
if (categories.length === 0) {
  fail('categories.json categories is empty')
}
const folderKeys = new Set()
const majorSortSeen = new Set()
for (const major of categories) {
  if (!major || typeof major !== 'object') continue
  if (!Number.isInteger(major.sort) || major.sort <= 0) {
    fail(`major category "${major.name || major.id}" missing valid positive integer sort`)
  } else if (majorSortSeen.has(major.sort)) {
    fail(`major sort duplicated: ${major.sort}`)
  } else {
    majorSortSeen.add(major.sort)
  }
  if (!String(major.key || '').trim()) {
    fail(`major category "${major.name || major.id}" missing key`)
  }
  const subs = Array.isArray(major.subcategories) ? major.subcategories : []
  if (subs.length === 0) {
    fail(`major category "${major.name || major.id}" has no subcategories`)
  }
  const subSortSeen = new Set()
  for (const sub of subs) {
    if (!Number.isInteger(sub?.sort) || sub.sort <= 0) {
      fail(`subcategory "${sub?.name || sub?.id}" missing valid positive integer sort`)
    } else if (subSortSeen.has(sub.sort)) {
      fail(`subcategory sort duplicated in major "${major.name || major.id}": ${sub.sort}`)
    } else {
      subSortSeen.add(sub.sort)
    }
    const fk = String(sub?.folderKey || '').trim()
    if (!fk) {
      fail(`subcategory "${sub?.name || sub?.id}" missing folderKey`)
      continue
    }
    if (folderKeys.has(fk)) {
      fail(`duplicate folderKey in categories.json: ${fk}`)
      continue
    }
    folderKeys.add(fk)
  }
}

if (!hasError) {
  // 明确首页来源：major.sort 最小 + 该 major 下 sub.sort 最小
  const firstMajor = [...categories]
    .filter((m) => Number.isInteger(m?.sort))
    .sort((a, b) => a.sort - b.sort)[0]
  const firstSub = [...(firstMajor?.subcategories || [])]
    .filter((s) => Number.isInteger(s?.sort))
    .sort((a, b) => a.sort - b.sort)[0]
  if (!firstMajor || !firstSub?.folderKey) {
    fail('cannot resolve homepage from sort order (major/sub)')
  } else {
    console.log(`INFO: homepage resolved by sort => ${firstSub.folderKey}`)
  }
}

for (const fk of folderKeys) {
  const folderDir = path.join(resourceRoot, fk)
  if (!fs.existsSync(folderDir)) {
    fail(`resource folder missing: ${fk}`)
    continue
  }
  const entries = fs.readdirSync(folderDir, { withFileTypes: true })
  for (const e of entries) {
    if (!ALLOWED_FOLDER_ENTRIES.has(e.name)) {
      fail(`folder ${fk} contains non-standard entry: ${e.name}`)
    }
  }
  const itemsPath = path.join(folderDir, 'items.json')
  if (!fs.existsSync(itemsPath)) {
    fail(`missing items.json in ${fk}`)
    continue
  }
  const itemsDoc = JSON.parse(fs.readFileSync(itemsPath, 'utf8'))
  const items = Array.isArray(itemsDoc.items) ? itemsDoc.items : []
  const idSeen = new Set()
  const linkSeen = new Set()
  for (const it of items) {
    const id = String(it?.id || '').trim()
    const filename = String(it?.filename || '').trim().replaceAll('\\', '/')
    const linkName = String(it?.linkName || '').trim()

    if (!ID_RE.test(id)) fail(`${fk}: invalid id format "${id}"`)
    if (idSeen.has(id)) fail(`${fk}: duplicate id "${id}"`)
    idSeen.add(id)

    if (!filename.startsWith('original/')) fail(`${fk}: filename should be under original/: ${filename}`)
    const ext = path.extname(filename)
    if (!ext) fail(`${fk}: filename missing extension: ${filename}`)
    if (filename !== `original/${id}${ext}`) {
      fail(`${fk}: filename should match id/ext, got "${filename}" expected "original/${id}${ext}"`)
    }
    const fullFile = path.join(folderDir, filename)
    if (!fs.existsSync(fullFile)) fail(`${fk}: missing file for item ${id}: ${filename}`)

    if (!linkName) fail(`${fk}: item ${id} missing linkName`)
    if (linkName !== `${id}${ext}`) fail(`${fk}: linkName should be "${id}${ext}", got "${linkName}"`)
    if (linkSeen.has(linkName)) fail(`${fk}: duplicate linkName "${linkName}"`)
    linkSeen.add(linkName)

    const thumbnail = String(it?.thumbnail || '').trim()
    if (thumbnail) {
      if (!thumbnail.startsWith('thumb/')) fail(`${fk}: thumbnail should be under thumb/: ${thumbnail}`)
      const expThumb = expectedThumbnailRel(filename)
      if (!expThumb) fail(`${fk}: thumbnail on item ${id} but filename is not a raster image: ${filename}`)
      if (thumbnail !== expThumb) {
        fail(`${fk}: thumbnail should be "${expThumb}", got ${thumbnail}`)
      }
      if (!fs.existsSync(path.join(folderDir, thumbnail))) {
        fail(`${fk}: thumbnail file missing: ${thumbnail}`)
      }
    }

    const rawFilename = String(it?.rawFilename || '').trim()
    if (rawFilename) {
      if (!rawFilename.startsWith('original/')) fail(`${fk}: rawFilename should be under original/: ${rawFilename}`)
      if (!fs.existsSync(path.join(folderDir, rawFilename))) {
        fail(`${fk}: raw file missing: ${rawFilename}`)
      }
      if (!String(it?.groupId || '').trim()) {
        fail(`${fk}: item ${id} has rawFilename but missing groupId`)
      }
    }
  }
}

if (hasError) {
  process.exit(1)
}
console.log('OK: data structure is compliant.')
