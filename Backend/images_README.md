# 图片与资源规范

本文件为 **`Backend/data` 资源组织、命名与访问策略的定稿说明**，面向上线与批量迁移：结构须可验证、可脚本化，避免上线后高成本重构。

| 文档 | 用途 |
|------|------|
| [../README.md](../README.md) | 项目总览、部署与发版检查 |
| [README.md](README.md) | Go 服务、API、维护脚本 |
| [data/resource/README.md](data/resource/README.md) | `resource/{folderKey}/` 目录速查 |

## 1. 目录结构（强制）

```text
Backend/data/
  categories.json
  accounts.json
  storage.json
  trash.json
  resource/
    {folderKey}/
      items.json
      original/
      thumb/
  system/
    avatar/
    resource/
```

规则：

- 业务资源**仅**允许在 `resource/{folderKey}/` 下。
- 系统资源**仅**允许在 `system/` 下（头像、占位图、默认图标等）。
- 每个 `resource/{folderKey}/` 下**仅**允许：`items.json`、`original/`、`thumb/`。

## 2. URL 与访问策略

| 类型 | 路径模式 |
|------|----------|
| 业务资源 | `/resource/{folderKey}/{linkName}` |
| 语义路径 | `/resource/{majorKey}/{folderKey}/{linkName}` |
| 系统资源 | `/resource/system/{fileName}` |
| 加密公开 | 上述路径 + `?k={view-key}` |

访问失败时返回英文提示（便于区分原因）：

- `Access denied: authentication required.`
- `Access denied: missing view key. Use ?k=YOUR_KEY.`
- `Access denied: invalid view key.`

## 3. 命名规范（ID + 日期）

| 字段 | 规则 | 示例 |
|------|------|------|
| `id` | `{hex12}_{YYYYMMDD}`，全局稳定、同条目不重复生成 | `ff9562f8dd83_20260511` |
| `filename` | `original/{id}.{ext}` | `original/ff9562f8dd83_20260511.jpeg` |
| `linkName` | `{id}.{ext}`，同目录唯一 | `ff9562f8dd83_20260511.jpeg` |
| `thumbnail` | `thumb/{与 original 同主干}.jpg`；PNG 原图为 `.png` | `thumb/ff9562f8dd83_20260511.jpg` |

示例：原图 `original/x_20260101.jpeg` → 缩略图 `thumb/x_20260101.jpg`。  
历史数据可用 `scripts/migrate-thumb-filenames.mjs` 迁移（无 `-thumbnail` 后缀）。

## 4. 图片组（JPG + RAW）

同一拍摄内容可并存「网页预览」与「原始下载」：

| 字段 | 说明 |
|------|------|
| `groupId` | 组标识，同组条目共用 |
| `rawFilename` | 原始文件路径（如 ARW），建议位于 `original/` |

展示策略：

1. 列表优先 `thumbnail`，无则回退 `url`。
2. 点击查看使用 `url`（JPG/PNG 等可预览格式）。
3. 存在 `rawUrl` 时提供「下载原始文件」入口。

## 5. items.json 字段

**必填：** `id`、`sort`、`uploadedAt`、`updatedAt`、`filename`、`linkName`

**可选：** `thumbnail`、`groupId`、`rawFilename`、`title`、`tags`、`masonryCol`、`masonryRow`

## 6. categories.json 字段

- `version`：结构版本号，供将来迁移；不参与业务排序。
- **首页**：不再使用 `homeFolderKey`；`sort` 最小的首个**可见**子分类即为首页。
- 大类建议提供语义 `key`（如 `liulan`、`tuku`），用于 `/resource/{majorKey}/...` 路径。

## 7. 执行与校验（必须）

每次批量整理 `data/` 后：

```bash
node Backend/scripts/reorganize-data.mjs
node Backend/scripts/validate-data.mjs
```

通过标准：

- 校验脚本输出：`OK: data structure is compliant.`
- `cd Backend && go test ./...`
- `cd Frontend && pnpm type-check`

## 8. 规划中的增强（非阻塞上线）

- 查看器：显式 `rawUrl` 下载按钮（ARW/RAW）
- 上传：预览图 + 原图（`rawFile`）双文件成组
- 管理端：冲突检测（重复 `linkName`、缺失文件、缺缩略图）
- 后台任务：历史图片批量补全 `thumb/`
