<template>
  <div
    class="header-box"
    :class="{ 'is-nav-more-open': menuOpen, 'is-user-menu-open': userMenuOpen }"
    :style="{ '--primary-nav-overlay-top': `${primaryNavOverlayTopPx}px` }"
  >
    <div class="container">
      <div class="navigation">
        <div ref="navigationBarRef" class="navigation__bar">
          <router-link class="nav-logo" to="/" aria-label="返回首页">
            <img :src="logoSrc" alt="" />
          </router-link>
          <div
            class="menu-toggle"
            :class="{ open: menuOpen }"
            role="button"
            tabindex="0"
            :aria-label="menuOpen ? '关闭菜单' : '打开菜单'"
            aria-controls="primary-nav-menu"
            :aria-expanded="menuOpen"
            @click="toggleMenu"
            @keydown.enter.prevent="toggleMenu"
            @keydown.space.prevent="toggleMenu"
          >
            <span class="menu-toggle__line menu-toggle__line--top" aria-hidden="true" />
            <span class="menu-toggle__line menu-toggle__line--bottom" aria-hidden="true" />
          </div>
        </div>
        <div id="primary-nav-menu" class="nav-link" :class="{ 'nav-link--open': menuOpen }">
          <div class="nav-link__grid-child">
            <div class="nav-link__backdrop" aria-hidden="true" @click="closeMenu" />
            <div class="nav-link__sheet">
              <template v-if="showNavCustomize">
                <div class="nav-link__sheet-custom">
                  <draggable
                    v-model="primaryList"
                    item-key="id"
                    tag="div"
                    class="nav-link__tg"
                    handle=".primary-nav-drag-handle"
                    :disabled="!primaryNavEditEnabled"
                    :animation="200"
                    :delay="100"
                    :delay-on-touch-only="true"
                    ghost-class="nav-link__drag-ghost"
                    chosen-class="nav-link__drag-chosen"
                    @end="onPrimaryDragEnd"
                  >
                    <template #item="{ element }">
                      <div class="nav-link__drag-row">
                        <img
                          v-if="primaryNavEditEnabled"
                          class="primary-nav-drag-handle"
                          src="@/assets/icon/drag.svg"
                          alt=""
                          width="14"
                          height="14"
                          aria-hidden="true"
                        />
                        <router-link class="nav-link__item" :to="element.link" @click="closeMenu">
                          <span class="nav-link__item-label">{{ element.name }}</span>
                          <img
                            v-if="element.showInvisibleHint"
                            class="nav-link__item-invisible"
                            src="@/assets/icon/invisible.svg"
                            alt=""
                            title="未登录时此分类入口不显示"
                            width="14"
                            height="14"
                            aria-hidden="true"
                          />
                        </router-link>
                      </div>
                    </template>
                  </draggable>
                  <button
                    type="button"
                    class="nav-link__add-btn"
                    title="添加导航"
                    aria-label="添加导航"
                    @click="openNavAdd"
                  >
                    <img
                      src="@/assets/icon/add.svg"
                      alt=""
                      width="18"
                      height="18"
                      aria-hidden="true"
                    />
                  </button>
                </div>
              </template>
              <template v-else>
                <router-link
                  v-for="item in navLinks"
                  :key="item.folderKey"
                  class="nav-link__item"
                  :to="item.link"
                  @click="closeMenu"
                >
                  <span class="nav-link__item-label">{{ item.name }}</span>
                  <img
                    v-if="item.showInvisibleHint"
                    class="nav-link__item-invisible"
                    src="@/assets/icon/invisible.svg"
                    alt=""
                    title="未登录时此分类入口不显示"
                    width="14"
                    height="14"
                    aria-hidden="true"
                  />
                </router-link>
              </template>
              <div class="nav-link__separator-line" aria-hidden="true"></div>
              <div
                ref="avatarTriggerRef"
                class="login nav-user-trigger nav-user-trigger--sheet"
                :aria-label="accountAria"
                @click.stop="handleAvatarClick"
              >
                <img :src="sheetAvatarSrc" alt="" />
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>

  <Teleport to="body">
    <div
      v-if="userMenuOpen"
      ref="userMenuRef"
      class="nav-user-menu"
      :class="{ 'nav-user-menu--mobile': isMobileNav }"
      :style="userMenuStyle"
      @click.stop
      @mousedown.stop
    >
      <div class="nav-user-menu__head">
        {{
          authStore.currentUser?.displayName ||
          authStore.currentUser?.email ||
          authStore.currentUser?.username ||
          '账户'
        }}
      </div>
      <button
        v-for="item in userMenuItems"
        :key="item.key"
        type="button"
        class="nav-user-menu__item"
        :class="{
          'nav-user-menu__item--danger': item.danger,
          'nav-user-menu__item--divider': item.divider,
        }"
        @click.stop="onUserMenuItemClick(item.key)"
      >
        <img class="nav-user-menu__icon-image" :src="item.iconSrc" alt="" aria-hidden="true" />
        <span class="nav-user-menu__text">{{ item.label }}</span>
      </button>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, watch, nextTick, onMounted, onUnmounted, computed } from 'vue'
import { storeToRefs } from 'pinia'
import { useRoute } from 'vue-router'
import draggable from 'vuedraggable'
import { useCategoryNavStore } from '@/stores/categoryNav'
import type { PrimaryNavLink } from '@/stores/categoryNav'
import { useAuthStore } from '@/stores/auth'
import { patchCategoriesNavOrder } from '@/api/adminApi'
import { useNavAddModalStore } from '@/stores/navAddModal'
import { useDragSortEditStore } from '@/stores/dragSortEdit'
import logoSrc from '@/assets/images/logo3-text.png'
import personSrc from '@/assets/icon/person.svg'
import privacySrc from '@/assets/icon/privacy.svg'
import exitSrc from '@/assets/icon/exit.svg'
import recycleSrc from '@/assets/icon/delete.svg'
import guestAvatarFallbackSrc from '@/assets/icon/Notloggedin.svg'

const categoryNavStore = useCategoryNavStore()
const authStore = useAuthStore()
const navAddModalStore = useNavAddModalStore()
const dragSortEditStore = useDragSortEditStore()
const { primaryNavLinks } = storeToRefs(categoryNavStore)
const { authenticated, authConfigured, authBootstrapping, showsLoggedInNav } = storeToRefs(authStore)
const navLinks = computed(() => primaryNavLinks.value)
const userMenuOpen = ref(false)
const userMenuRef = ref<HTMLElement | null>(null)
const avatarTriggerRef = ref<HTMLElement | null>(null)
const userMenuStyle = ref<Record<string, string>>({})
const navWideMq = typeof window !== 'undefined' ? window.matchMedia('(min-width: 576px)') : null
const isMobileNav = computed(() => !(navWideMq?.matches ?? true))

type UserMenuKey = 'profile' | 'account' | 'recycle' | 'signout'

type UserMenuItem = {
  key: UserMenuKey
  label: string
  iconSrc: string
  divider?: boolean
  danger?: boolean
}

const canManageAccounts = computed(() =>
  !!(authStore.currentUser?.permissions ?? []).includes('manage_accounts'),
)

const userMenuItems = computed<UserMenuItem[]>(() => {
  const items: UserMenuItem[] = [
    { key: 'profile', label: '账户设置', iconSrc: personSrc },
    { key: 'account', label: '隐私设置', iconSrc: privacySrc },
  ]
  if (canManageAccounts.value) {
    items.push({ key: 'recycle', label: '回收站', iconSrc: recycleSrc })
  }
  items.push({ key: 'signout', label: 'Sign Out', iconSrc: exitSrc })
  return items
})

const showNavCustomize = computed(() => showsLoggedInNav.value && authConfigured.value)
const { enabled: dragSortEditEnabled } = storeToRefs(dragSortEditStore)
const primaryNavEditEnabled = computed(() => showNavCustomize.value && dragSortEditEnabled.value)

const primaryList = ref<PrimaryNavLink[]>([])

watch(
  () => primaryNavLinks.value.map((l) => l.id).join('|'),
  () => {
    primaryList.value = primaryNavLinks.value.map((x) => ({ ...x }))
  },
  { immediate: true },
)

async function onPrimaryDragEnd() {
  if (!primaryNavEditEnabled.value) return
  const ordered = primaryList.value.map((l) => l.id)
  const prev = primaryNavLinks.value.map((l) => l.id)
  if (ordered.length === prev.length && ordered.every((id, i) => id === prev[i])) return
  try {
    const doc = await patchCategoriesNavOrder({ primaryMajorIds: ordered })
    categoryNavStore.replaceDoc(doc)
  } catch (e) {
    console.error(e)
    primaryList.value = primaryNavLinks.value.map((x) => ({ ...x }))
    window.alert('导航顺序保存失败，请检查权限或网络')
  }
}

function openNavAdd() {
  navAddModalStore.openPrimary()
  closeMenu()
}

const sheetAvatarSrc = computed(() => {
  if (showsLoggedInNav.value) {
    return authStore.resolvedAvatarUrl() || personSrc
  }
  return authStore.guestAvatarSrc() || guestAvatarFallbackSrc
})

const accountAria = computed(() => {
  if (authBootstrapping.value) return '正在恢复登录状态'
  if (showsLoggedInNav.value && authenticated.value) {
    return isMobileNav.value ? '账户设置' : '账户信息，点击查看或退出'
  }
  return '未登录，点击登录'
})

async function handleAvatarClick() {
  if (authBootstrapping.value) {
    void authStore.refreshStatus()
  }
  if (authenticated.value || (authBootstrapping.value && authStore.hasStoredSession())) {
    if (isMobileNav.value) {
      userMenuOpen.value = false
      await authStore.openProfileModal()
      closeMenu()
      return
    }
    userMenuOpen.value = !userMenuOpen.value
    if (userMenuOpen.value) {
      await nextTick()
      syncUserMenuPosition()
    }
    return
  }
  authStore.openLoginModal(
    authStore.authStatusError
      ? authStore.authStatusError
      : authConfigured.value
        ? '登录后可查看私密目录；加密目录需额外输入查看密码'
        : '当前服务器未启用登录（无 accounts.json 用户且无 LUMEHUB_PASSWORD），仅供浏览公开内容。',
  )
  closeMenu()
}

async function openProfileFromMenu() {
  userMenuOpen.value = false
  await authStore.openProfileModal()
  closeMenu()
}

async function openPrivacyFromMenu() {
  userMenuOpen.value = false
  await authStore.openProfileModal('nav')
  closeMenu()
}

async function openRecycleFromMenu() {
  userMenuOpen.value = false
  await authStore.openProfileModal('recycle')
  closeMenu()
}

async function logoutFromMenu() {
  await authStore.logout()
  userMenuOpen.value = false
  closeMenu()
}

function onUserMenuItemClick(key: UserMenuKey) {
  if (key === 'profile') {
    void openProfileFromMenu()
    return
  }
  if (key === 'account') {
    void openPrivacyFromMenu()
    return
  }
  if (key === 'recycle') {
    void openRecycleFromMenu()
    return
  }
  if (key === 'signout') {
    void logoutFromMenu()
    return
  }
  userMenuOpen.value = false
}

const menuOpen = defineModel<boolean>('menuOpen', { default: false })
const route = useRoute()

const navigationBarRef = ref<HTMLElement | null>(null)
const primaryNavOverlayTopPx = ref(56)

function measureNavigationBar() {
  const el = navigationBarRef.value
  if (!el) return
  const r = el.getBoundingClientRect()
  primaryNavOverlayTopPx.value = Math.max(0, Math.round(r.bottom))
}

let navigationBarResizeObserver: ResizeObserver | null = null

function syncUserMenuPosition() {
  if (!userMenuOpen.value) return
  if (isMobileNav.value) {
    userMenuStyle.value = { zIndex: '2500' }
    return
  }
  const el = avatarTriggerRef.value
  if (!el) return
  const r = el.getBoundingClientRect()
  userMenuStyle.value = {
    position: 'fixed',
    top: `${Math.round(r.bottom + 10)}px`,
    right: `${Math.max(12, Math.round(window.innerWidth - r.right))}px`,
    left: 'auto',
    width: '272px',
    zIndex: '2500',
  }
}

function closeMenu() {
  menuOpen.value = false
  userMenuOpen.value = false
}

function toggleMenu() {
  menuOpen.value = !menuOpen.value
}

function onNavWideMqChange() {
  if (navWideMq?.matches) menuOpen.value = false
}

watch(
  () => route.fullPath,
  () => {
    menuOpen.value = false
    userMenuOpen.value = false
  },
)

watch(
  () => showNavCustomize.value,
  (ok) => {
    if (!ok) dragSortEditStore.setEnabled(false)
  },
)

watch(userMenuOpen, (open) => {
  if (open) void nextTick(() => syncUserMenuPosition())
})

watch(menuOpen, (open) => {
  if (open) nextTick(() => measureNavigationBar())
})

onMounted(() => {
  measureNavigationBar()
  if (typeof ResizeObserver !== 'undefined' && navigationBarRef.value) {
    navigationBarResizeObserver = new ResizeObserver(measureNavigationBar)
    navigationBarResizeObserver.observe(navigationBarRef.value)
  }
  window.addEventListener('resize', measureNavigationBar)
  window.addEventListener('resize', syncUserMenuPosition)
  window.addEventListener('scroll', syncUserMenuPosition, true)
  window.addEventListener('click', onWindowClick)
  navWideMq?.addEventListener('change', onNavWideMqChange)
  navWideMq?.addEventListener('change', syncUserMenuPosition)
})

onUnmounted(() => {
  navigationBarResizeObserver?.disconnect()
  navigationBarResizeObserver = null
  window.removeEventListener('resize', measureNavigationBar)
  window.removeEventListener('resize', syncUserMenuPosition)
  window.removeEventListener('scroll', syncUserMenuPosition, true)
  window.removeEventListener('click', onWindowClick)
  navWideMq?.removeEventListener('change', onNavWideMqChange)
  navWideMq?.removeEventListener('change', syncUserMenuPosition)
})

function onWindowClick(event: MouseEvent) {
  if (!userMenuOpen.value) return
  const target = event.target as Node | null
  if (!target) return
  const menuEl = userMenuRef.value
  if (menuEl?.contains(target)) return
  userMenuOpen.value = false
}
</script>

<style scoped lang="scss">
$ease-brand: cubic-bezier(0.22, 1, 0.36, 1);

.header-box {
  width: 100%;
  padding: 1rem 0;
  position: relative;
  left: 0;
  top: 0;

  &.is-nav-more-open {
    z-index: 2100;
  }

  &.is-user-menu-open {
    z-index: 2500;
  }

  &.is-nav-more-open .navigation__bar {
    position: relative;
    z-index: 2101;
  }

  .navigation {
    display: flex;
    align-items: center;
    justify-content: space-between;
    flex-wrap: wrap;
    margin: 1rem 0;
    gap: 0.75rem;
  }

  .navigation__bar {
    display: flex;
    align-items: center;
    gap: 1rem;
    flex-shrink: 0;
  }

  .nav-logo {
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    text-decoration: none;
    -webkit-tap-highlight-color: transparent;

    img {
      display: block;
      width: auto;
      max-width: min(230px, 38vw);
      object-fit: contain;
    }

    &:focus-visible {
      outline: 2px solid rgba(0, 0, 0, 0.35);
      outline-offset: 3px;
      border-radius: 6px;
    }
  }

  .nav-user-trigger {
    display: flex;
    align-items: center;
    justify-content: center;
    width: clamp(40px, 10vw, 44px);
    height: clamp(40px, 10vw, 44px);
    padding: 0;
    border: none;
    border-radius: 50%;
    overflow: hidden;
    cursor: pointer;
    background: transparent;
    flex-shrink: 0;
    -webkit-tap-highlight-color: transparent;

    img {
      width: 100%;
      height: 100%;
      object-fit: cover;
      object-position: center;
    }

    &:focus-visible {
      outline: 2px solid rgba(0, 0, 0, 0.35);
      outline-offset: 2px;
    }
  }

  .menu-toggle {
    display: none;
    flex-shrink: 0;
    width: 44px;
    height: 44px;
    margin: 0;
    padding: 0;
    border-radius: 10px;
    background: transparent;
    cursor: pointer;
    position: relative;
    z-index: 2102;
    box-sizing: border-box;
    -webkit-tap-highlight-color: transparent;
    user-select: none;
    transition: background 0.22s $ease-brand;

    &:hover {
      background: rgba(0, 0, 0, 0.045);
    }

    &__line {
      position: absolute;
      left: 50%;
      width: 22px;
      height: 2px;
      margin-left: -11px;
      border-radius: 1px;
      background: #111;
      transform-origin: 50% 50%;
      transition:
        transform 0.3s $ease-brand,
        top 0.3s $ease-brand,
        bottom 0.3s $ease-brand;

      &--top {
        top: 15px;
        bottom: auto;
        transform: translateY(0) rotate(0deg);
      }

      &--bottom {
        top: auto;
        bottom: 15px;
        transform: translateY(0) rotate(0deg);
      }
    }

    &.open {
      .menu-toggle__line--top {
        top: 50%;
        bottom: auto;
        transform: translateY(-50%) rotate(45deg);
      }

      .menu-toggle__line--bottom {
        top: 50%;
        bottom: auto;
        transform: translateY(-50%) rotate(-45deg);
      }
    }

    &:focus {
      outline: none;
    }

    &:focus-visible {
      outline: none;
      background: rgba(0, 0, 0, 0.07);
    }
  }

  .nav-link {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 2px;
    flex: 1;
    justify-content: flex-end;
    min-width: 0;

    &__grid-child {
      display: contents;
    }

    &__backdrop {
      display: none;
    }

    &__sheet {
      display: flex;
      align-items: center;
      flex-wrap: wrap;
      gap: 2px;
      position: relative;
    }

    &__sheet-custom {
      display: flex;
      flex-wrap: wrap;
      align-items: center;
      gap: 2px;
      flex: 1;
      min-width: 0;
      justify-content: flex-end;
    }

    &__tg {
      display: flex;
      flex-wrap: wrap;
      align-items: center;
      gap: 2px;
      flex: 1;
      min-width: 0;
      justify-content: flex-end;
    }

    :deep(.nav-link__drag-ghost) {
      opacity: 0.45;
    }

    :deep(.nav-link__drag-chosen) {
      cursor: grabbing;
    }

    &__sheet .nav-link__drag-row {
      display: inline-flex;
      flex-direction: row;
      align-items: stretch;
      max-width: 100%;
      border-radius: 8px;
      -webkit-tap-highlight-color: transparent;

      > a.nav-link__item {
        flex: 1;
        min-width: 0;
        position: relative;
      }
    }

    .primary-nav-drag-handle {
      width: 14px;
      height: 14px;
      flex-shrink: 0;
      align-self: center;
      display: block;
      object-fit: contain;
      opacity: 0.62;
      cursor: grab;
      margin-right: 4px;
    }

    :deep(.nav-link__drag-chosen) .primary-nav-drag-handle {
      cursor: grabbing;
      opacity: 0.92;
    }

    &__sheet .nav-link__add-btn {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      padding: 8px clamp(8px, 2.4vw, 12px);
      margin: 0;
      border: none;
      border-radius: 8px;
      background: transparent;
      cursor: pointer;
      flex-shrink: 0;
      -webkit-tap-highlight-color: transparent;
      transition: background-color 0.22s $ease-brand;

      &:hover {
        background: rgba(0, 0, 0, 0.05);
      }

      &:focus-visible {
        outline: 2px solid rgba(0, 0, 0, 0.35);
        outline-offset: 2px;
      }

      img {
        display: block;
        width: clamp(16px, 4vw, 18px);
        height: clamp(16px, 4vw, 18px);
        object-fit: contain;
        opacity: 0.72;
      }

      &:hover img {
        opacity: 1;
      }
    }

    &__sheet a.nav-link__item {
      display: inline-flex;
      align-items: center;
      justify-content: space-between;
      gap: 0.35rem;
      max-width: 100%;
      box-sizing: border-box;
    }

    &__sheet .nav-link__item-label {
      min-width: 0;
      font-family: 'Smiley Sans', sans-serif;
    }

    &__sheet .nav-link__item-invisible {
      flex-shrink: 0;
      width: 14px;
      height: 14px;
      display: block;
      object-fit: contain;
      opacity: 0.55;
      
      position: absolute;
      right: 0;
      top: 0;
      bottom: 0;
      margin: auto;
    }

    &__sheet a {
      font-size: 20px;
      letter-spacing: 0.01em;
      color: #666;
      padding: 8px clamp(8px, 2.4vw, 16px);
      border-radius: 8px;
      transition:
        background-color 0.22s $ease-brand,
        color 0.22s $ease-brand;

      &:hover {
        color: #000;
      }

      &.router-link-active {
        color: #000;
      }
    }

    &__separator-line {
      display: none;
    }

    &__sheet .nav-user-trigger--sheet {
      margin-left: 1.5rem;
      width: 35px;
      height: 35px;
    }
  }
}

.nav-user-menu {
  position: absolute;
  right: 0;
  top: calc(100% + 10px);
  width: 272px;
  background: #ffffff;
  border: 1px solid #d2d2d2;
  border-radius: 8px;
  box-shadow: 0 14px 26px -16px rgba(0, 0, 0, 0.42);
  z-index: 2500;
  overflow: hidden;
  pointer-events: auto;

  &::before {
    content: '';
    position: absolute;
    top: -7px;
    right: 26px;
    width: 12px;
    height: 12px;
    background: #fff;
    border-left: 1px solid #d2d2d2;
    border-top: 1px solid #d2d2d2;
    transform: rotate(45deg);
  }

  &__head {
    padding: 12px 14px;
    border-bottom: 1px solid #ececec;
    font-size: 18px;
    font-weight: 700;
    color: #000;
    position: relative;
    z-index: 1;
  }

  &__item {
    width: 100%;
    text-align: left;
    display: flex;
    align-items: center;
    gap: 10px;
    border: none;
    background: #fff;
    padding: 12px 14px;
    font-size: 12px;
    font-weight: 700;
    color: #111;
    cursor: pointer;
    border-bottom: 1px solid #f0f0f0;

    &:hover {
      background: #f6f6f6;
    }
  }

  &__icon {
    width: 18px;
    flex: 0 0 18px;
    text-align: center;
    font-size: 14px;
    line-height: 1;
  }

  &__icon-image {
    width: 18px;
    height: 18px;
    flex: 0 0 18px;
    display: block;
    object-fit: contain;
  }

  &__text {
    flex: 1;
    min-width: 0;
  }

  &__item--divider {
    border-top: 1px solid #e5e5e5;
  }

  &__item--danger {
    color: #d70015;
  }
}

@media (max-width: 575px) {
  .header-box {
    .menu-toggle {
      display: block;
    }

    .navigation {
      flex-direction: column;
      align-items: stretch;
      flex-wrap: nowrap;
      gap: 0;
    }

    .navigation__bar {
      justify-content: space-between;
      width: 100%;
    }

    .nav-link {
      position: fixed;
      left: 0;
      right: 0;
      top: var(--primary-nav-overlay-top, 56px);
      bottom: 0;
      z-index: 2000;
      flex: none;
      justify-content: stretch;
      width: 100%;
      display: grid;
      grid-template-rows: 0fr;
      transition: grid-template-rows 0.38s $ease-brand;
      pointer-events: none;
      min-height: 0;
      margin: 0;
      padding: 0;
      gap: 0;

      &--open {
        grid-template-rows: 1fr;
        pointer-events: auto;
        overscroll-behavior: contain;

        .nav-link__backdrop {
          opacity: 1;
          pointer-events: auto;
        }
      }

      &__grid-child {
        display: flex;
        flex-direction: column;
        min-height: 0;
        overflow: hidden;
        height: 100%;
        position: relative;
      }

      &__backdrop {
        display: block;
        position: fixed;
        left: 0;
        right: 0;
        top: var(--primary-nav-overlay-top, 56px);
        bottom: 0;
        z-index: 0;
        margin: 0;
        padding: 0;
        border: none;
        background: rgba(255, 255, 255, 0.76);
        backdrop-filter: blur(16px) saturate(1);
        -webkit-backdrop-filter: blur(16px) saturate(1);
        opacity: 0;
        transition: opacity 0.34s $ease-brand;
        pointer-events: none;
      }

      &__sheet {
        position: relative;
        z-index: 1;
        flex: 1;
        display: flex;
        flex-direction: column;
        align-items: stretch;
        /* flex-start：避免 justify-content:center 在内容溢出时产生“上下被裁切、滚不到头尾”的 flex 溢出问题 */
        justify-content: center;
        flex-wrap: nowrap;
        gap: 0;
        width: 100%;
        box-sizing: border-box;
        padding: 1.35rem 1.25rem 1.5rem;
        padding-bottom: calc(1.5rem + env(safe-area-inset-bottom, 0px));
        color: #1a1a1a;
        text-align: center;
        overflow-x: hidden;
        overflow-y: auto;
        min-height: 0;
        -webkit-overflow-scrolling: touch;
        overscroll-behavior: contain;
        touch-action: pan-y;
        border-radius: 12px 12px 0 0;
        background:
          radial-gradient(ellipse 95% 55% at 50% 0%, rgba(255, 255, 255, 0.98) 0%, transparent 58%),
          radial-gradient(ellipse 70% 45% at 100% 100%, rgba(0, 0, 0, 0.03) 0%, transparent 52%),
          linear-gradient(178deg, #ffffff 0%, #f0f0f0 100%);
        box-shadow: 0 -20px 56px -18px rgba(0, 0, 0, 0.14);
      }

      &__sheet-custom {
        flex-direction: column;
        align-items: stretch;
        width: 100%;
        justify-content: flex-start;
        flex: none;
      }

      &__tg {
        flex-direction: column;
        align-items: stretch;
        width: 100%;
        justify-content: flex-start;
        flex: none;
      }

      &__sheet .nav-link__drag-row {
        width: 100%;
        justify-content: center;
        cursor: grab;
        flex-direction: column;
        align-items: stretch;
      }

      &__sheet .nav-link__add-btn {
        width: auto;
        margin: 0.35rem auto 0;
        padding: 0.65rem 14px;

        img {
          width: 20px;
          height: 20px;
        }
      }

      &__sheet a.nav-link__item {
        justify-content: center;
        gap: 0.5rem;
      }

      &__sheet .nav-link__item-label {
        text-align: center;
        flex: 1;
        min-width: 0;
      }

      &__sheet .nav-link__item-invisible {
        margin-left: auto;
        flex-shrink: 0;
      }

      &__sheet a {
        letter-spacing: 0.015em;
        color: #666;
        padding: 0.72rem 14px;
        border-radius: 10px;
        text-decoration: none;
        width: 100%;
        box-sizing: border-box;
        transition:
          background-color 0.22s $ease-brand,
          color 0.22s $ease-brand;

        &:hover {
          color: #000;
        }

        &.router-link-active {
          color: #000;
        }
      }

      &__separator-line {
        display: block;
        align-self: center;
        flex-shrink: 0;
        width: 20px;
        height: 1px;
        margin: 1rem auto;
        background: rgba(0, 0, 0, 0.05);
      }

      &__sheet .nav-user-trigger--sheet {
        width: 50px;
        height: 50px;
        margin: 0.65rem auto 0;
        flex-shrink: 0;
      }
    }
  }

  .nav-user-menu {
    position: fixed;
    left: 12px;
    right: 12px;
    top: auto;
    bottom: calc(env(safe-area-inset-bottom, 0px) + 12px);
    width: auto;
    max-height: min(70vh, 420px);
    overflow-y: auto;
    z-index: 2500;

    &.nav-user-menu--mobile {
      position: fixed;
    }

    &::before {
      display: none;
    }
  }
}

@media (max-width: 640px) {
  .header-box {
    padding: 0.75rem 0;

    .navigation {
      margin: 0.5rem 0;
    }
  }
}
</style>
