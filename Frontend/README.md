# LumeHub 前端

基于 **Vue 3**、**Vite 8**、**Pinia**、**TypeScript** 的单页应用，与 Go 后端同域部署；开发期通过 Vite 代理访问 API。

**相关文档**：[项目总览](../README.md) · [后端 API 与数据](../Backend/README.md) · [资源规范](../Backend/images_README.md)

## 环境要求

| 依赖 | 版本 |
|------|------|
| Node.js | `^20.19.0` 或 `>=22.12.0` |
| 包管理器 | pnpm（推荐） |

## 快速开始

```bash
pnpm install
```

先启动后端（`Backend` 目录：`go run ./cmd/lumehub`，默认 `:5353`），再启动前端：

```bash
pnpm dev
```

- 开发服务器默认 `--host`，便于局域网调试
- `/api`、`/resource` 代理到 `http://127.0.0.1:5353`

自定义后端地址：

```bash
# Windows PowerShell
$env:VITE_DEV_API_PROXY="http://127.0.0.1:5353"; pnpm dev

# Linux / macOS
VITE_DEV_API_PROXY=http://127.0.0.1:5353 pnpm dev
```

## 构建与预览

```bash
pnpm build      # vue-tsc 类型检查 + 输出到 ../Backend/www
pnpm preview    # 本地预览构建产物（仍需后端提供 API）
pnpm type-check # 仅类型检查
pnpm format     # Prettier 格式化 src/
```

生产环境只需部署 `Backend/www` 并由 Go 服务托管，见 [项目 README · 宝塔部署](../README.md#宝塔部署)。

## 目录结构

```text
src/
  api/                    # HTTP 封装（gallery、auth、admin、trash）
  assets/                 # 静态资源与 SVG 图标
  components/
    gallery/              # 卡片、网格、瀑布流、编辑与转移
    modals/               # 登录、设置、导航添加等
    shared-ui/            # 对话框、分页、表单控件
    viewers/              # 图片 / 视频 / 文件详情查看器
    settings/             # 设置子面板（如回收站）
  composables/            # 可复用逻辑（如画廊图片编辑）
  layout/                 # 顶栏导航、页脚
  router/                 # 路由定义
  stores/                 # Pinia（auth、分类导航、画廊同步等）
  utils/                  # URL、权限、导出质量等工具
  views/
    index.vue             # 首页（首个可见子分类）
    Category.vue          # 路由壳：权限与 folderKey 校验
    GalleryView.vue       # 画廊主视图
    gallery/              # 上传面板、工具栏、组合式函数
```

## 路由

| 路径 | 组件 | 说明 |
|------|------|------|
| `/` | `index.vue` | 首页，展示排序最前的可见子分类 |
| `/gallery/:folderKey` | `Category.vue` → `GalleryView.vue` | 指定分类画廊 |
| `/c/:folderKey` | — | 301 式重定向到 `/gallery/:folderKey` |

## 推荐工具

- 编辑器：[VS Code](https://code.visualstudio.com/) + [Vue - Official](https://marketplace.visualstudio.com/items?itemName=Vue.volar)（请禁用 Vetur）
- 调试：[Vue.js devtools](https://chromewebstore.google.com/detail/vuejs-devtools/nhdogjmejiglipccpnnnanhbledajbpd)
