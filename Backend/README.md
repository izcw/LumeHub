# LumeHub 后端（Go）

单二进制 HTTP 服务：JSON API、静态资源（`data/resource`）、前端 SPA（`www/`）。同域部署，无跨域，无数据库。

**相关文档**：[项目总览](../README.md) · [资源规范](images_README.md) · [前端](../Frontend/README.md) · [宝塔部署](../README.md#宝塔部署)

## 目录

- [本地开发](#本地开发)
- [生产构建](#生产构建)
- [上传到服务器](#上传到服务器)
- [环境变量](#环境变量)
- [数据目录](#数据目录)
- [数据维护脚本](#数据维护脚本)
- [API 概览](#api-概览)
- [测试](#测试)

## 本地开发

需要 **Go 1.24+**。在 `Backend` 目录：

```bash
go run ./cmd/lumehub
```

默认 **http://127.0.0.1:5353**。

日常开发可在 `Frontend` 另开 `pnpm dev`（Vite 代理 `/api`、`/resource`），无需每次构建 `www/`。  
若要接近生产环境联调：先在 `Frontend` 执行 `pnpm build`，再启动后端。

## 生产构建

发版前先在 `Frontend` 构建前端（产物写入 `Backend/www/`）：

```bash
cd Frontend && pnpm install && pnpm build
```

后端按开发机系统选 **一种** 方式编译 Linux 可执行文件，再按 [上传到服务器](#上传到服务器) 部署。  
Supervisor 与端口配置见 [项目 README · 宝塔部署](../README.md#宝塔部署)。

### Linux 上编译

在 Linux 开发机或服务器上，于 `Backend` 目录：

```bash
go build -trimpath -ldflags "-s -w" -o lumehub ./cmd/lumehub
```

产物：`Backend/lumehub`。

### Windows 上编译

在 `Backend` 目录交叉编译 Linux 可执行文件：

```powershell
cd Frontend; pnpm build
cd ..\Backend
.\scripts\build-linux.ps1
```

产物：`Backend/dist/linux-amd64/lumehub`。  
上传用宝塔文件管理或 WinSCP 手动操作，见 [上传到服务器](#上传到服务器)。

## 上传到服务器

服务在服务器上以 **一个目录** 为根（示例 `/www/wwwroot/lumehub/`），进程的工作目录即该路径。  
以下路径均相对于 `Backend/`，上传后保持相同层级：

```text
/www/wwwroot/lumehub/          ← 服务器根目录（LUMEHUB_ROOT）
├── lumehub                    ← Go 可执行文件（Linux 上 chmod +x）
├── www/                       ← 前端构建产物（pnpm build 输出）
│   ├── index.html
│   └── assets/
└── data/                      ← 运行时数据（JSON + 图片/视频）
    ├── categories.json
    ├── accounts.json
    ├── storage.json
    ├── trash.json
    ├── resource/              ← 各分类原图、缩略图（体积最大）
    └── system/                ← 头像、系统占位图等
```

### 各路径说明

| 路径 | 来源 | 是否必传 | 说明 |
|------|------|----------|------|
| `lumehub` | `go build` 或 `dist/linux-amd64/lumehub` | **每次发版** | 后端主程序；Windows 编译后从此处取 |
| `www/` | `Frontend` 的 `pnpm build` | **每次发版** | 整目录覆盖；含 `index.html` 与 `assets/` |
| `data/` | 本地 `Backend/data/` | **仅首次** | 分类、账号、资源库；日常发版**不要覆盖** |
| `data/resource/` | 同上 | **仅首次或单独同步** | 用户图片/视频，体积大，与程序发版分开管理 |
| `.env` | 可选 | 否 | 变量通常写在 Supervisor 环境配置中，见 [.env.example](.env.example) |

**不要上传：** `internal/`、`cmd/`、`scripts/`、`go.mod`、源码及 Windows 上的 `lumehub.exe`、`dist/` 等构建中间目录。

### 首次部署

1. 在服务器创建目录，例如 `/www/wwwroot/lumehub/`
2. 上传 **`lumehub`**、**`www/`** 整个目录、**`data/`** 整个目录（含已有资源时一并上传 `data/resource/`）
3. 执行 `chmod +x lumehub`；按需 `chown`（见 [宝塔部署 · 权限](../README.md#2-权限)）
4. 修改 `data/accounts.json` 中的默认密码，配置 Supervisor 并启动

**Linux 手动上传**（在 `Backend` 目录，按实际修改）：

```bash
scp lumehub user@服务器IP:/www/wwwroot/lumehub/
scp -r www user@服务器IP:/www/wwwroot/lumehub/
scp -r data user@服务器IP:/www/wwwroot/lumehub/    # 仅首次
```

**Windows 手动上传**（编译完成后，用宝塔 / WinSCP 上传到 `REMOTE_PATH`，例如 `/www/wwwroot/lumehub/`）：

| 本地路径 | 上传到服务器 |
|----------|----------------|
| `dist/linux-amd64/lumehub` | `lumehub`（覆盖） |
| `www/` 整个文件夹 | `www/`（整目录覆盖） |
| `data/` 整个文件夹 | `data/`（**仅首次**） |

上传后在服务器执行 `chmod +x lumehub`，并在 Supervisor 中重启进程。

<details>
<summary>可选：scp 自动上传（scripts/deploy.ps1）</summary>

复制 [deploy.example.env](deploy.example.env) 为 `deploy.env`，填写 `DEPLOY_TARGET`、`REMOTE_PATH` 后：

```powershell
.\scripts\deploy.ps1              # 上传已有产物
.\scripts\deploy.ps1 -Build        # 先编译再上传
.\scripts\deploy.ps1 -BuildFrontend  # 前端 + 编译 + 上传
```

脚本只上传 `lumehub` 与 `www/`，**不会覆盖** `data/`。
</details>

### 版本更新

只替换会随代码变更的部分，**保留服务器上的 `data/`**（含用户上传的资源与账号）：

| 操作 | 上传 |
|------|------|
| 后端或前端有更新 | 覆盖 `lumehub` + `www/` |
| 仅前端 UI 更新 | 仅覆盖 `www/` |
| 仅后端逻辑更新 | 仅覆盖 `lumehub` |

上传后在 Supervisor 中 **重启进程**。

### 仅同步资源库

若在本地整理过 `data/resource/` 或 JSON，可单独同步整个 `data/`，或只传变更的文件；**避免用开发环境的 `accounts.json` 覆盖生产密码**。大体积资源建议用 rsync / 宝塔同步，不必每次随程序发版。

## 环境变量

| 变量 | 含义 | 默认 |
|------|------|------|
| `LUMEHUB_ADDR` | 监听地址 | `:5353` |
| `LUMEHUB_DATA` | 数据根目录 | `./data` |
| `LUMEHUB_WWW` | 前端静态目录 | `./www` |
| `LUMEHUB_ROOT` | 工作目录（影响相对路径） | 当前工作目录 |
| `LUMEHUB_PASSWORD` | 单口令（多账号时优先 `accounts.json`） | 空 |
| `LUMEHUB_SESSION_HOURS` | 会话有效小时数 | 内置默认 |
| `LUMEHUB_COOKIE_SECURE` | HTTPS 下 Secure Cookie（`1` / `true`） | 关闭 |
| `LUMEHUB_NO_COLOR` | 关闭启动横幅颜色 | 关闭 |

生产示例：[.env.example](.env.example) · Supervisor：[supervisor.example.ini](supervisor.example.ini)

## 数据目录

```text
data/
  categories.json      # 大类 / 子分类、布局、公开与加密
  accounts.json        # 多账号、角色与权限
  storage.json         # 存储配额与用量
  trash.json           # 回收站
  resource/
    {folderKey}/
      items.json       # 条目索引
      original/        # 原文件
      thumb/           # 缩略图
  system/
    avatar/            # 用户头像
    resource/          # 系统占位图、默认头像等
```

目录约定：[data/resource/README.md](data/resource/README.md)  
完整字段规范：[images_README.md](images_README.md)

要点：

- **首页**：`sort` 最小的首个可见子分类
- **条目 ID**：`{hex12}_{YYYYMMDD}`
- **业务 URL**：`/resource/{folderKey}/{linkName}`
- **加密公开**：`/resource/.../{linkName}?k={view-key}`

## 数据维护脚本

在 `Backend` 目录执行（需 **Node.js**）：

| 命令 | 作用 |
|------|------|
| `node scripts/sync-resource-items.mjs` | 扫描磁盘，重写各目录 `items.json` |
| `node scripts/migrate-items-metadata.mjs` | 补充 `linkName`、`thumbnail` 等 |
| `node scripts/migrate-thumb-filenames.mjs` | 缩略图文件名迁移 |
| `node scripts/normalize-link-names.mjs` | 规范化 `linkName` |
| `node scripts/reorganize-data.mjs` | 批量整理目录结构 |
| `node scripts/validate-data.mjs` | 校验数据结构 |

Go 工具：`go run ./cmd/resourceindex` · `go run ./cmd/vidthumb`

批量整理后执行 `validate-data.mjs` 与 `go test ./...`（见 [images_README.md §7](images_README.md#7-执行与校验必须)）。

## API 概览

写操作需已登录；管理接口需相应角色权限。

### 分类与条目

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/categories` | 全部分类元数据 |
| PATCH | `/api/categories/visibility` | 修改分类可见性 |
| PATCH | `/api/categories/name` | 修改分类名称 |
| PATCH | `/api/categories/folder-key` | 修改 `folderKey` |
| PATCH | `/api/categories/sub-major` | 调整子分类所属大类 |
| PATCH | `/api/categories/nav-order` | 导航排序 |
| POST | `/api/categories/major` | 创建大类 |
| POST | `/api/categories/sub` | 创建子分类 |
| DELETE | `/api/categories/major` | 删除大类 |
| DELETE | `/api/categories/sub` | 删除子分类 |
| GET | `/api/category/{folderKey}` | 布局 + 条目列表 |
| PATCH | `/api/category/{folderKey}/layout` | 更新布局 |
| POST | `/api/category/{folderKey}/upload` | 上传文件 |
| POST | `/api/category/{folderKey}/upload/session` | 分片上传 |
| PATCH | `/api/category/{folderKey}/items/{itemId}` | 更新条目 |
| DELETE | `/api/category/{folderKey}/items/{itemId}` | 删除条目 |

### 认证与账号

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/auth/login` | 登录 |
| POST | `/api/auth/logout` | 登出 |
| GET | `/api/auth/status` | 登录状态 |
| GET | `/api/auth/me` | 当前用户 |
| PATCH | `/api/auth/me` | 更新资料 |
| POST / DELETE | `/api/auth/me/avatar` | 头像 |
| GET / POST / PATCH / DELETE | `/api/auth/accounts` | 账号管理 |
| POST | `/api/auth/passkey/register/*` | Passkey 注册 |
| POST | `/api/auth/qr/session` | 扫码登录 |

### 存储与回收站

| 方法 | 路径 | 说明 |
|------|------|------|
| GET / PATCH | `/api/storage` | 存储配额与用量 |
| POST | `/api/storage/recalculate` | 重新计算用量 |
| GET / DELETE | `/api/trash` | 回收站列表 / 清空 |
| DELETE | `/api/trash/{folderKey}` | 永久删除目录下全部 |
| DELETE | `/api/trash/{folderKey}/items/{itemId}` | 永久删除单条 |
| POST | `/api/trash/{folderKey}/items/{itemId}/restore` | 恢复单条 |
| POST | `/api/trash/{folderKey}/restore` | 恢复目录全部 |

### 静态与 SPA

| 路径 | 说明 |
|------|------|
| `/resource/*` | 业务与加密资源 |
| `/resource/system/*` | 系统资源 |
| `/api/avatar/{id}` | 用户头像 |
| `/` | SPA → `www/index.html` |

## 测试

```bash
go test ./...
```
