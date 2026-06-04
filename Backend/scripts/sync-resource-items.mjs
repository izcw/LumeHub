import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import crypto from 'node:crypto'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const resourceRoot = path.resolve(__dirname, '../data/resource')
const IMAGE_EXTS = new Set(['.jpg', '.jpeg', '.png', '.webp', '.gif', '.bmp', '.avif'])
const RAW_EXTS = new Set(['.arw', '.dng', '.nef', '.cr2', '.rw2'])

function fileTimestamps(fullPath) {
  const st = fs.statSync(fullPath)
  const mtimeIso = st.mtime.toISOString()
  const birth = st.birthtime
  const uploadedAt =
    birth && !Number.isNaN(birth.getTime()) && birth.getTime() > 0
      ? birth.toISOString()
      : mtimeIso
  return { uploadedAt, updatedAt: mtimeIso }
}

function datePart(iso) {
  const d = new Date(iso)
  const y = d.getUTCFullYear()
  const m = String(d.getUTCMonth() + 1).padStart(2, '0')
  const day = String(d.getUTCDate()).padStart(2, '0')
  return `${y}${m}${day}`
}

function collectFilesRecursive(baseDir, rel = '') {
  const abs = rel ? path.join(baseDir, rel) : baseDir
  const out = []
  for (const e of fs.readdirSync(abs, { withFileTypes: true })) {
    const relPath = rel ? path.join(rel, e.name) : e.name
    if (e.isDirectory()) {
      if (e.name.toLowerCase() === 'thumb') continue
      out.push(...collectFilesRecursive(baseDir, relPath))
      continue
    }
    const norm = relPath.replaceAll('\\', '/')
    if (norm.toLowerCase() === 'items.json') continue
    out.push(norm)
  }
  return out
}

for (const folder of fs.readdirSync(resourceRoot, { withFileTypes: true })) {
  if (!folder.isDirectory()) continue
  const dir = path.join(resourceRoot, folder.name)
  const names = collectFilesRecursive(dir).sort((a, b) =>
    a.localeCompare(b, undefined, { sensitivity: 'base', numeric: true }),
  )

  const used = new Set()
  const byStem = new Map()
  for (const name of names) {
    const ext = path.extname(name).toLowerCase()
    const stem = path.parse(name).name
    if (!byStem.has(stem)) byStem.set(stem, [])
    byStem.get(stem).push({ name, ext })
  }

  const items = names.map((filename) => {
    const full = path.join(dir, filename)
    const { uploadedAt, updatedAt } = fileTimestamps(full)
    const extName = path.extname(filename).toLowerCase()
    const ext = extName.slice(1)
    const d = datePart(uploadedAt || updatedAt)
    const seed = `${folder.name}|${filename}|${uploadedAt}|${updatedAt}`
    let token = crypto.createHash('sha1').update(seed).digest('hex').slice(0, 12)
    let id = `${token}_${d}`
    let n = 1
    while (used.has(id)) {
      token = crypto
        .createHash('sha1')
        .update(`${seed}|${n}`)
        .digest('hex')
        .slice(0, 12)
      id = `${token}_${d}`
      n += 1
    }
    used.add(id)
    const sort = used.size * 10
    const linkName = `${id}${extName || '.bin'}`
    let thumbnail = ''
    if (IMAGE_EXTS.has(extName)) {
      const stem = path.parse(path.basename(filename)).name
      thumbnail = extName === '.png' ? `thumb/${stem}.png` : `thumb/${stem}.jpg`
    }
    const thumbnailPath = thumbnail ? path.join(dir, thumbnail) : ''
    let rawFilename = ''
    let groupId = ''
    if (IMAGE_EXTS.has(extName)) {
      const stem = path.parse(filename).name
      const peers = byStem.get(stem) || []
      const rawPeer = peers.find((p) => RAW_EXTS.has(p.ext))
      if (rawPeer) {
        rawFilename = rawPeer.name.replaceAll('\\', '/')
        groupId = `grp_${id}`
      }
    }
    return {
      id,
      sort,
      uploadedAt,
      updatedAt,
      filename,
      linkName,
      ...(thumbnail && fs.existsSync(thumbnailPath) ? { thumbnail: thumbnail.replaceAll('\\', '/') } : {}),
      ...(groupId ? { groupId } : {}),
      ...(rawFilename ? { rawFilename } : {}),
      ...(ext ? { tags: [ext] } : {}),
    }
  })

  const out = path.join(dir, 'items.json')
  fs.writeFileSync(out, JSON.stringify({ items }, null, 2) + '\n', 'utf8')
  console.log(folder.name, items.length)
}
