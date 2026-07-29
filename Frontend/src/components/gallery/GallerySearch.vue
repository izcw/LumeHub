<template>
  <button
    type="button"
    class="gallery-search__trigger"
    title="搜索与筛选"
    aria-label="打开搜索筛选"
    :aria-expanded="open"
    @click="open = true"
  >
    <img class="gallery-search__trigger-icon" src="@/assets/icon/search.svg" alt="" />
  </button>
  <Teleport to="body">
    <Transition name="gallery-search-mount">
      <div v-if="open" class="gallery-search__mount">
        <div class="gallery-search__backdrop" aria-hidden="true" @click="close" />
        <div
          class="gallery-search__dialog"
          role="dialog"
          aria-modal="true"
          aria-label="搜索与筛选"
          @click.stop
        >
          <span class="gallery-search__sr-only">搜索与筛选</span>
          <div class="gallery-search__sheet">
            <div class="container gallery-search__inner">
              <div class="gallery-search__top">
                <div class="gallery-search__row">
                  <input
                    ref="searchInputRef"
                    v-model="searchDraft.query"
                    type="search"
                    class="gallery-search__input"
                    placeholder="请输入内容.."
                    autocomplete="off"
                    @keydown.enter.prevent="apply"
                  />
                  <button type="button" class="gallery-search__submit" @click="apply">搜索</button>
                </div>
                <button
                  type="button"
                  class="gallery-search__close"
                  aria-label="关闭"
                  @click="close"
                />
              </div>

              <div class="gallery-search__body">
                <section class="gallery-search__section" aria-labelledby="gallery-search-scope-title">
                  <h3 id="gallery-search-scope-title" class="gallery-search__section-title">搜索范围</h3>
                  <div
                    class="gallery-search__chips gallery-search__chips--scope"
                    role="group"
                    aria-label="搜索范围"
                  >
                    <button
                      type="button"
                      class="gallery-search__chip gallery-search__chip--scope"
                      :class="{ 'is-on': searchDraft.searchScope === 'current' }"
                      @click="searchDraft.searchScope = 'current'"
                    >
                      当前分类
                    </button>
                    <button
                      type="button"
                      class="gallery-search__chip gallery-search__chip--scope"
                      :class="{ 'is-on': searchDraft.searchScope === 'all' }"
                      @click="searchDraft.searchScope = 'all'"
                    >
                      全部
                    </button>
                  </div>
                </section>

                <section
                  v-if="categoryOptions.length"
                  class="gallery-search__section"
                  aria-labelledby="gallery-search-category-title"
                >
                  <h3 id="gallery-search-category-title" class="gallery-search__section-title">分类</h3>
                  <div
                    class="gallery-search__chips gallery-search__chips--category"
                    role="group"
                    aria-label="文件大类"
                  >
                    <button
                      v-for="opt in categoryOptions"
                      :key="opt.id"
                      type="button"
                      class="gallery-search__chip gallery-search__chip--category"
                      :class="{ 'is-on': searchDraft.categories.includes(opt.id) }"
                      @click="toggleDraftCategory(opt.id)"
                    >
                      <img
                        v-if="categoryIconSrc(opt.id)"
                        class="gallery-search__chip-category-icon"
                        :src="categoryIconSrc(opt.id)"
                        alt=""
                        width="16"
                        height="16"
                      />
                      <span class="gallery-search__chip-category-label">{{ opt.label }}</span>
                    </button>
                  </div>
                </section>

                <section
                  v-if="extensionOptions.length"
                  class="gallery-search__section"
                  aria-labelledby="gallery-search-suffix-title"
                >
                  <h3 id="gallery-search-suffix-title" class="gallery-search__section-title">后缀</h3>
                  <div
                    class="gallery-search__chips gallery-search__chips--suffix"
                    role="group"
                    aria-label="文件扩展名"
                  >
                    <button
                      v-for="opt in extensionOptions"
                      :key="opt.id"
                      type="button"
                      class="gallery-search__chip gallery-search__chip--suffix"
                      :class="{ 'is-on': searchDraft.extensions.includes(opt.id) }"
                      @click="toggleDraftExtension(opt.id)"
                    >
                      {{ opt.label }}
                    </button>
                  </div>
                </section>

                <section
                  v-if="tagOptions.length"
                  class="gallery-search__section"
                  aria-labelledby="gallery-search-tags-title"
                >
                  <h3 id="gallery-search-tags-title" class="gallery-search__section-title">标签</h3>
                  <div
                    class="gallery-search__chips gallery-search__chips--tags"
                    role="group"
                    aria-label="标签"
                  >
                    <button
                      v-for="opt in tagOptions"
                      :key="opt.id"
                      type="button"
                      class="gallery-search__chip gallery-search__chip--tag"
                      :class="{ 'is-on': searchDraft.tags.includes(opt.id) }"
                      @click="toggleDraftTag(opt.id)"
                    >
                      {{ opt.label }}
                    </button>
                  </div>
                </section>
              </div>

              <div class="gallery-search__footer">
                <button type="button" class="gallery-search__footer-reset" @click="reset">
                  重置搜索
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { reactive, ref, watch, nextTick, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { storeToRefs } from 'pinia'
import { useCategoryNavStore } from '@/stores/categoryNav'
import {
  useGallerySearchStore,
  GALLERY_SEARCH_FOLDER_KEY,
  GALLERY_CATEGORY_OPTIONS,
  type GallerySearchScope,
} from '@/stores/gallerySearch'
import { isOnSearchGalleryRoute } from '@/utils/gallerySearchFolder'
import { buildGallerySearchQuery } from '@/utils/gallerySearchQuery'
import iconCategoryImage from '@/assets/icon/Picture.svg'
import iconCategoryVideo from '@/assets/icon/video.svg'
import iconCategoryAudio from '@/assets/icon/Audio.svg'
import iconCategoryArchive from '@/assets/icon/Compress.svg'
import iconCategoryDocument from '@/assets/icon/File.svg'

const GALLERY_CATEGORY_ICON_SRC: Record<string, string> = {
  image: iconCategoryImage,
  video: iconCategoryVideo,
  audio: iconCategoryAudio,
  archive: iconCategoryArchive,
  document: iconCategoryDocument,
  other: iconCategoryDocument,
}

function categoryIconSrc(id: string): string | undefined {
  return GALLERY_CATEGORY_ICON_SRC[id]
}

/** 与 header 同步：打开时参与 body 滚动锁与 Esc */
const open = defineModel<boolean>('open', { default: false })

const route = useRoute()
const router = useRouter()
const gallerySearchStore = useGallerySearchStore()
const categoryNavStore = useCategoryNavStore()
const { filterOptionsAll } = storeToRefs(gallerySearchStore)

const STATIC_SEARCH_CATEGORY_IDS = new Set([
  ...GALLERY_CATEGORY_OPTIONS.map((c) => c.id),
  'other',
])

function allScopeFilterOptionsReady(): boolean {
  const all = filterOptionsAll.value
  return (
    all.categories.length > 0 || all.extensions.length > 0 || all.tags.length > 0
  )
}
const searchInputRef = ref<HTMLInputElement | null>(null)

function captureReturnFolderKey(): string {
  if (isOnSearchGalleryRoute(route.name, route.params.folderKey)) {
    return gallerySearchStore.searchReturnFolderKey
  }
  if (route.name === 'category') {
    const raw = route.params.folderKey
    const fk = typeof raw === 'string' ? raw : Array.isArray(raw) ? (raw[0] ?? '') : ''
    if (fk) return fk
  }
  return categoryNavStore.homeFolderKey
}

function navigateAfterSearchScope(scope: GallerySearchScope) {
  if (scope === 'all') {
    const from = captureReturnFolderKey()
    gallerySearchStore.setSearchReturnFolderKey(from)
    void router.push({
      name: 'category',
      params: { folderKey: GALLERY_SEARCH_FOLDER_KEY },
      query: buildGallerySearchQuery({
        query: searchDraft.query,
        categories: [...searchDraft.categories],
        extensions: [...searchDraft.extensions],
        tags: [...searchDraft.tags],
        from,
      }),
    })
    return
  }
  if (isOnSearchGalleryRoute(route.name, route.params.folderKey)) {
    const back = gallerySearchStore.searchReturnFolderKey.trim()
    if (back) void router.push({ name: 'category', params: { folderKey: back } })
    else void router.push({ name: 'home' })
  }
}

function ensureAllScopeFilterOptions() {
  gallerySearchStore.requestGlobalPoolRefresh()
}

const searchDraft = reactive({
  query: '',
  searchScope: 'current' as GallerySearchScope,
  categories: [] as string[],
  extensions: [] as string[],
  tags: [] as string[],
})

const activeFilterOptions = computed(() =>
  gallerySearchStore.filterOptionsForScope(searchDraft.searchScope),
)
const categoryOptions = computed(() => activeFilterOptions.value.categories)
const extensionOptions = computed(() => activeFilterOptions.value.extensions)
const tagOptions = computed(() => activeFilterOptions.value.tags)

function syncDraftFromStore() {
  searchDraft.query = gallerySearchStore.query
  searchDraft.searchScope = gallerySearchStore.searchScope
  searchDraft.categories = [...gallerySearchStore.selectedCategories]
  searchDraft.extensions = [...gallerySearchStore.selectedExtensions]
  searchDraft.tags = [...gallerySearchStore.selectedTags]
}

function close() {
  open.value = false
}

function apply() {
  const scope = searchDraft.searchScope
  gallerySearchStore.setFilters({
    query: searchDraft.query,
    categories: [...searchDraft.categories],
    extensions: [...searchDraft.extensions],
    tags: [...searchDraft.tags],
    searchScope: scope,
  })
  open.value = false
  navigateAfterSearchScope(scope)
}

function reset() {
  const onSearchPage = isOnSearchGalleryRoute(route.name, route.params.folderKey)
  const back = onSearchPage ? gallerySearchStore.searchReturnFolderKey.trim() : ''
  gallerySearchStore.resetFilters()
  searchDraft.query = ''
  searchDraft.searchScope = 'current'
  searchDraft.categories = []
  searchDraft.extensions = []
  searchDraft.tags = []
  if (onSearchPage) {
    if (back) void router.push({ name: 'category', params: { folderKey: back } })
    else void router.push({ name: 'home' })
  }
}

function toggleDraftCategory(id: string) {
  const i = searchDraft.categories.indexOf(id)
  if (i >= 0) searchDraft.categories.splice(i, 1)
  else searchDraft.categories.push(id)
}

function toggleDraftExtension(id: string) {
  const i = searchDraft.extensions.indexOf(id)
  if (i >= 0) searchDraft.extensions.splice(i, 1)
  else searchDraft.extensions.push(id)
}

function toggleDraftTag(id: string) {
  const i = searchDraft.tags.indexOf(id)
  if (i >= 0) searchDraft.tags.splice(i, 1)
  else searchDraft.tags.push(id)
}

function pruneDraftToScopeOptions(scope: GallerySearchScope) {
  const options = gallerySearchStore.filterOptionsForScope(scope)
  if (scope === 'all' && !allScopeFilterOptionsReady()) {
    searchDraft.categories = searchDraft.categories.filter((id) => STATIC_SEARCH_CATEGORY_IDS.has(id))
    const extIds = new Set(options.extensions.map((o) => o.id))
    const tagIds = new Set(options.tags.map((o) => o.id))
    searchDraft.extensions = searchDraft.extensions.filter((id) => extIds.has(id))
    searchDraft.tags = searchDraft.tags.filter((id) => tagIds.has(id))
    return
  }
  const catIds = new Set(options.categories.map((o) => o.id))
  const extIds = new Set(options.extensions.map((o) => o.id))
  const tagIds = new Set(options.tags.map((o) => o.id))
  searchDraft.categories = searchDraft.categories.filter((id) => catIds.has(id))
  searchDraft.extensions = searchDraft.extensions.filter((id) => extIds.has(id))
  searchDraft.tags = searchDraft.tags.filter((id) => tagIds.has(id))
}

watch(
  () => searchDraft.searchScope,
  (scope) => {
    if (scope === 'all') ensureAllScopeFilterOptions()
    pruneDraftToScopeOptions(scope)
  },
)

watch(
  filterOptionsAll,
  () => {
    if (searchDraft.searchScope === 'all') pruneDraftToScopeOptions('all')
  },
  { deep: true },
)

watch(open, (isOpen) => {
  if (isOpen) {
    syncDraftFromStore()
    if (searchDraft.searchScope === 'all') ensureAllScopeFilterOptions()
    nextTick(() => searchInputRef.value?.focus())
  }
})

defineExpose({ reset })
</script>

<style scoped lang="scss">
$ease-brand: cubic-bezier(0.22, 1, 0.36, 1);

/* 继承画廊工具条上的 CSS 变量（--gallery-toolbar-h 等） */
.gallery-search__trigger {
  display: flex;
  align-items: center;
  justify-content: center;
  width: var(--gallery-toolbar-h, 36px);
  height: var(--gallery-toolbar-h, 36px);
  min-width: var(--gallery-toolbar-h, 36px);
  min-height: var(--gallery-toolbar-h, 36px);
  padding: 0;
  margin: 0;
  font: inherit;
  border-radius: var(--gallery-toolbar-radius, 8px);
  background: var(--gallery-toolbar-bg, #111);
  border: none;
  cursor: pointer;
  flex-shrink: 0;
  box-sizing: border-box;
  transition: background 0.15s ease;

  &:hover {
    background: #1f1f1f;
  }
}

.gallery-search__trigger-icon {
  width: 18px;
  height: 18px;
  display: block;
  object-fit: contain;
  opacity: 0.55;
  filter: invert(1);
}

.gallery-search__mount {
  position: fixed;
  inset: 0;
  z-index: 2400;
  display: flex;
  align-items: flex-start;
  justify-content: center;
  padding: 0;
  padding-top: env(safe-area-inset-top, 0px);
  box-sizing: border-box;
  pointer-events: none;
  overflow-x: hidden;
}

.gallery-search__backdrop {
  position: absolute;
  inset: 0;
  background: rgba(255, 255, 255, 0.6);
  backdrop-filter: blur(8px) saturate(1);
  -webkit-backdrop-filter: blur(12px) saturate(1);
  pointer-events: auto;
}

.gallery-search__dialog {
  position: relative;
  z-index: 1;
  width: 100%;
  max-height: min(78vh, 640px);
  display: flex;
  flex-direction: column;
  align-items: center;
  pointer-events: none;
}

.gallery-search__sheet {
  width: 100%;
  margin: 0 auto;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  justify-content: flex-start;
  background:
    radial-gradient(ellipse 95% 65% at 50% 0%, rgba(255, 255, 255, 0.95) 0%, transparent 55%),
    linear-gradient(180deg, #ffffff 0%, #f5f5f5 100%);
  border-radius: 0 0 12px 12px;
  box-shadow: 0 28px 56px -24px rgba(0, 0, 0, 0.16);
  pointer-events: auto;
  transform: translateY(0);
  will-change: transform;
}

.gallery-search-mount-enter-active .gallery-search__backdrop,
.gallery-search-mount-leave-active .gallery-search__backdrop {
  transition: opacity 0.3s $ease-brand;
}

.gallery-search-mount-enter-active .gallery-search__sheet,
.gallery-search-mount-leave-active .gallery-search__sheet {
  transition:
    transform 0.42s $ease-brand,
    opacity 0.32s $ease-brand;
}

.gallery-search-mount-enter-from .gallery-search__backdrop,
.gallery-search-mount-leave-to .gallery-search__backdrop {
  opacity: 0;
}

.gallery-search-mount-enter-from .gallery-search__sheet,
.gallery-search-mount-leave-to .gallery-search__sheet {
  transform: translateY(-100%);
  opacity: 0;
}

.gallery-search-mount-enter-to .gallery-search__backdrop,
.gallery-search-mount-leave-from .gallery-search__backdrop {
  opacity: 1;
}

.gallery-search-mount-enter-to .gallery-search__sheet,
.gallery-search-mount-leave-from .gallery-search__sheet {
  transform: translateY(0);
  opacity: 1;
}

.gallery-search__sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border: 0;
}

.gallery-search__top {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 16px 0 14px;
  flex-shrink: 0;
}

.gallery-search__row {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: stretch;
  gap: 10px;
  border: none;
  border-radius: 10px;
  overflow: visible;
  background: transparent;
}

.gallery-search__input {
  width: 100%;
  max-width: 600px;
  min-width: 0;
  margin: 0;
  padding: 10px 12px;
  font-size: 13px;
  line-height: 1.4;
  border: none;
  color: #000;
  border: 1px solid #000;

  &::placeholder {
    color: #737373;
  }

  &:focus,
  &:focus-visible {
    outline: none;
    box-shadow: none;
  }
}

.gallery-search__submit {
  flex-shrink: 0;
  padding: 0 16px;
  margin: 0;
  border: none;
  background: #000;
  color: #fff;
  font-size: 13px;
  font-weight: 500;
  letter-spacing: 0.04em;
  cursor: pointer;
  transition: background 0.18s $ease-brand;

  &:hover {
    background: #222;
  }

  &:focus,
  &:focus-visible {
    outline: none;
  }
}

.gallery-search__close {
  flex-shrink: 0;
  position: relative;
  width: 32px;
  height: 32px;
  margin: 4px 0 0;
  padding: 0;
  display: block;
  border: none;
  background: transparent;
  color: #404040;
  cursor: pointer;
  transition:
    background 0.12s ease,
    color 0.12s ease;

  &::before,
  &::after {
    content: '';
    position: absolute;
    left: 50%;
    top: 50%;
    width: 20px;
    height: 2px;
    border-radius: 1px;
    background: currentColor;
  }

  &::before {
    transform: translate(-50%, -50%) rotate(45deg);
  }

  &::after {
    transform: translate(-50%, -50%) rotate(-45deg);
  }

  &:hover {
    background: #ebebeb;
    color: #000;
  }

  &:focus,
  &:focus-visible {
    outline: none;
  }
}

.gallery-search__body {
  padding: 4px 0 16px;
  overflow-y: auto;
  flex: 1;
  min-height: 0;
}

.gallery-search__section {
  margin-bottom: 26px;

  &:last-child {
    margin-bottom: 4px;
  }
}

.gallery-search__section-title {
  margin: 0 0 10px;
  font-size: 15px;
  font-weight: 700;
  color: #000;
  letter-spacing: 0.02em;
}

.gallery-search__hint {
  margin: -4px 0 12px;
  font-size: 12px;
  line-height: 1.5;
  color: #737373;
}

.gallery-search__chips {
  display: flex;
  flex-wrap: wrap;
}

.gallery-search__chips--scope {
  gap: 10px 12px;
}

.gallery-search__chip--scope {
  padding: 6px 14px;
  font-size: 13px;
  font-weight: 500;
  color: #303030;
  background: #eee;
  border-radius: 999px;

  &:hover {
    background: #e0e0e0;
    color: #000;
  }

  &.is-on {
    background: #000;
    color: #fff;
    font-weight: 600;
  }
}

.gallery-search__chips--category {
  gap: 10px 12px;
}

.gallery-search__chips--suffix {
  gap: 8px 10px;
}

.gallery-search__chips--tags {
  gap: 8px 10px;
}

.gallery-search__chip {
  cursor: pointer;
  border: none;
  font-family: inherit;
  transition:
    background 0.12s ease,
    color 0.12s ease,
    box-shadow 0.12s ease;
}

.gallery-search__chip--category {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 10px;
  font-size: 12px;
  font-weight: 500;
  color: #303030;
  background: #eee;

  &:hover {
    background: #e0e0e0;
    color: #000;
  }

  &.is-on {
    background: #000;
    color: #fff;
    font-weight: 600;
  }
}

.gallery-search__chip-category-icon {
  flex-shrink: 0;
  width: 16px;
  height: 16px;
  display: block;
  object-fit: contain;
  /* 多色 SVG 压成单色，与字色同一套明暗 */
  filter: brightness(0) saturate(100%);
  opacity: 0.72;
  transition:
    opacity 0.12s ease,
    filter 0.12s ease;
}

.gallery-search__chip--category:hover:not(.is-on) .gallery-search__chip-category-icon {
  opacity: 0.88;
}

.gallery-search__chip--category.is-on .gallery-search__chip-category-icon {
  filter: brightness(0) invert(1);
  opacity: 1;
}

.gallery-search__chip-category-label {
  line-height: 1.2;
}

.gallery-search__chip--suffix {
  padding: 5px 9px;
  font-size: 12px;
  font-weight: 500;
  font-variant-numeric: tabular-nums;
  letter-spacing: 0.02em;
  color: #303030;
  background: #eee;

  &:hover {
    background: #e0e0e0;
    color: #000;
  }

  &.is-on {
    background: #000;
    color: #fff;
    font-weight: 600;
  }
}

.gallery-search__chip--tag {
  padding: 4px 8px;
  font-size: 12px;
  font-weight: 400;
  color: #303030;
  background: #eee;

  &:hover {
    background: #e5e5e5;
    color: #000;
  }

  &.is-on {
    background: #000;
    color: #fff;
    font-weight: 500;
  }
}

.gallery-search__footer {
  display: flex;
  justify-content: flex-start;
  padding: 12px 0 16px;
  flex-shrink: 0;
  border: none;
  background: #f5f5f5;
}

.gallery-search__footer-reset {
  padding: 0;
  border: none;
  background: none;
  font-size: 13px;
  color: #525252;
  text-decoration: underline;
  text-underline-offset: 3px;
  cursor: pointer;

  &:hover {
    color: #000;
  }
}

@media (max-width: 400px) {
  .gallery-search__trigger-icon {
    width: 16px;
    height: 16px;
  }
}
</style>
