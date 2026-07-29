<template>
  <div class="index-box">
    <Header />
    <div class="container-mini">
      <!-- <header v-if="isGlobalSearchView" class="gallery-search-results-head">
        <h2 class="gallery-search-results-head__title">搜索结果</h2>
        <p v-if="hasActiveFilter" class="gallery-search-results-head__meta">
          共 {{ orderedFilteredRows.length }} 项
        </p>
      </header> -->

      <GalleryEmptyStates
        v-if="!showGalleryContent"
        :loading="pageLoading"
        :is-global-search-view="isGlobalSearchView"
        :search-needs-criteria="searchNeedsCriteria"
        :access-blocked="accessBlocked"
        :access-blocked-mode="accessBlockedMode"
        :access-blocked-message="accessBlockedMessage"
        :access-blocked-action-text="accessBlockedActionText"
        :password-input="passwordDialogInput"
        :password-error="passwordDialogError"
        :show-add-gallery-empty="showAddGalleryEmpty"
        :load-error="loadError"
        :nav-load-error="navLoadError"
        :search-scope="searchScope"
        :global-pool-loading="globalPoolLoading"
        :search-source-rows-empty="searchSourceRows.length === 0"
        :filtered-rows-empty="filteredRows.length === 0"
        @blocked-action="onBlockedActionClick"
        @add-gallery="onAddGalleryClick"
        @update:password-input="passwordDialogInput = $event"
        @password-submit="submitFolderPassword"
      />

      <GalleryStage
        v-else
        ref="galleryStageRef"
        :waterfall-stage-visible="waterfallStageVisible"
        :dragging-cards="draggingCards"
        :display-list="displayList"
        :can-drag-sort="canDragSort"
        :effective-item-sort="effectiveItemSort"
        :show-gallery-admin-actions="showGalleryAdminActions"
        :gallery-folder-key="folderKey"
        :get-item-style="getItemStyle"
        :show-pagination="showPagination"
        :order-persisting="orderPersisting"
        :order-persist-hint="orderPersistHint"
        :drag-sort-edit-enabled="dragSortEditEnabled"
        :page-size-menu-open="pageSizeMenuOpen"
        :page-size-select-value="pageSizeSelectValue"
        :page-size="pageSize"
        :page-size-select-options="pageSizeSelectOptions"
        :current-page="currentPage"
        :total-pages="totalPages"
        :ordered-filtered-count="orderedFilteredRows.length"
        @update:display-list="displayList = $event"
        @layout="handleGalleryLayout"
        @reorder="handleMasonryReorder"
        @drag-start="handleDragStart"
        @drag-end="onStageDragEnd"
        @card-click="handleCardClick"
        @delete="handleItemDelete"
        @transfer="openItemTransfer"
        @edit="openItemEdit"
        @view="handleCardView"
        @download="handleItemDownload"
        @copy-link="handleItemCopyLink"
        @aspect-hw="updateItemMasonryAspectHW"
        @toggle="togglePageSizeMenu"
        @select="selectPageSize"
        @prev="goPrevPage"
        @next="goNextPage"
      />
    </div>

    <Footer />

    <GalleryUploadPanel
      ref="uploadPanelUiRef"
      :site-file-drag-active="siteFileDragActive"
      :visible="showUploadFloatingPanel"
      :collapsed="uploadPanelCollapsed"
      :dragging="uploadPanelDragging"
      :panel-style="uploadPanelStyle"
      :tasks="uploadTasks"
      :upload-in-progress-count="uploadInProgressCount"
      :reduce-icon="reduceIcon"
      @panel-drag-start="onUploadPanelDragStart"
      @bubble-click="onUploadBubbleClick"
      @toggle-panel="toggleUploadPanel"
      @open-picker="uploadPanelUiRef?.openFilePicker()"
      @cancel-task="cancelUploadTask"
      @file-selected="onUploadFilesPicked"
    />

    <ImageViewer
      v-model:visible="viewerVisible"
      :images="viewerImages"
      :live-video-urls="viewerLiveVideoUrls"
      :item-ids="viewerImageItemIds"
      :initial-index="viewerInitialIndex"
      :from-rect="startRect"
      :nav-current="viewerNavCurrent"
      :nav-total="viewerNavTotal"
      :slide-direction="viewerSlideDirection"
      :skip-exit-animation="viewerSkipExit"
      :toolbar-pinned="viewerToolbarPinned"
      :adjacent-prev-src="viewerAdjacentPrevSrc"
      :adjacent-next-src="viewerAdjacentNextSrc"
      @edge-navigate="handleViewerNavigate"
      @close="handleViewerClose"
    >
      <template v-if="viewerToolbarItem" #toolbar>
        <GalleryViewerToolbar
          :item="viewerToolbarItem"
          :show-admin-actions="showGalleryAdminActions"
          :show-transfer="canTransferGalleryItem(viewerToolbarItem)"
          @delete="handleItemDelete(viewerToolbarItem)"
          @transfer="openItemTransfer(viewerToolbarItem)"
          @edit="openItemEdit(viewerToolbarItem)"
          @download="handleItemDownload(viewerToolbarItem)"
          @copy-link="handleItemCopyLink(viewerToolbarItem)"
          @pin-change="viewerToolbarPinned = $event"
        />
      </template>
    </ImageViewer>

    <VideoViewer
      v-model:visible="videoViewerVisible"
      :videos="videoViewerUrls"
      :item-ids="videoViewerItemIds"
      :initial-index="videoViewerInitialIndex"
      :from-rect="startRect"
      :nav-current="viewerNavCurrent"
      :nav-total="viewerNavTotal"
      :skip-exit-animation="viewerSkipExit"
      :toolbar-pinned="viewerToolbarPinned"
      @edge-navigate="handleViewerNavigate"
      @close="handleVideoViewerClose"
    >
      <template v-if="viewerToolbarItem" #toolbar>
        <GalleryViewerToolbar
          :item="viewerToolbarItem"
          :show-admin-actions="showGalleryAdminActions"
          :show-transfer="canTransferGalleryItem(viewerToolbarItem)"
          @delete="handleItemDelete(viewerToolbarItem)"
          @transfer="openItemTransfer(viewerToolbarItem)"
          @edit="openItemEdit(viewerToolbarItem)"
          @download="handleItemDownload(viewerToolbarItem)"
          @copy-link="handleItemCopyLink(viewerToolbarItem)"
          @pin-change="viewerToolbarPinned = $event"
        />
      </template>
    </VideoViewer>

    <FileDetailViewer
      :open="fileDetailOpen"
      :payload="fileDetailPayload"
      :nav-current="viewerNavCurrent"
      :nav-total="viewerNavTotal"
      :toolbar-pinned="viewerToolbarPinned"
      @close="closeFileDetail"
      @download="handleFileDetailDownload"
      @copy-link="handleFileDetailCopyLink"
      @navigate="handleViewerNavigate"
    >
      <template v-if="viewerToolbarItem" #toolbar>
        <GalleryViewerToolbar
          :item="viewerToolbarItem"
          :show-admin-actions="showGalleryAdminActions"
          :show-transfer="canTransferGalleryItem(viewerToolbarItem)"
          @delete="handleItemDelete(viewerToolbarItem)"
          @transfer="openItemTransfer(viewerToolbarItem)"
          @edit="openItemEdit(viewerToolbarItem)"
          @download="handleItemDownload(viewerToolbarItem)"
          @copy-link="handleItemCopyLink(viewerToolbarItem)"
          @pin-change="viewerToolbarPinned = $event"
        />
      </template>
    </FileDetailViewer>

    <GalleryItemEditDialog
      :open="editDialogOpen"
      :payload="editPayload"
      :transfer-options="editTransferOptions"
      :transfer-submitting="transferSubmitting"
      :z-index="galleryDialogZIndex"
      @close="closeItemEdit"
      @saved="handleItemEditSaved"
      @transfer="handleItemTransferFromEdit"
    />

  </div>
</template>

<script setup lang="ts">
import { defineAsyncComponent, ref, toRef, watch, computed } from 'vue'
import Header from '@/layout/header.vue'
import Footer from '@/layout/footer.vue'
import GalleryEmptyStates from '@/views/gallery/GalleryEmptyStates.vue'
import GalleryStage from '@/views/gallery/GalleryStage.vue'
import GalleryUploadPanel from '@/views/gallery/GalleryUploadPanel.vue'
import { FileDetailViewer } from '@/components/viewers'
import GalleryItemEditDialog from '@/components/gallery/GalleryItemEditDialog.vue'
import GalleryViewerToolbar from '@/views/gallery/GalleryViewerToolbar.vue'
import { VIEWER_Z } from '@/components/viewers/shared/viewerLayers'
import { useGalleryViewState } from '@/views/gallery/composables'
import reduceIcon from '@/assets/icon/reduce.svg'

const ImageViewer = defineAsyncComponent(() => import('@/components/viewers/ImageViewer.vue'))
const VideoViewer = defineAsyncComponent(() => import('@/components/viewers/VideoViewer.vue'))

const props = defineProps<{
  folderKey: string
}>()

const uploadPanelUiRef = ref<InstanceType<typeof GalleryUploadPanel>>()
const viewerToolbarPinned = ref(false)

const {
  galleryStageRef,
  showGalleryContent,
  pageLoading,
  navLoadError,
  loading,
  loadError,
  accessBlocked,
  accessBlockedMode,
  accessBlockedMessage,
  accessBlockedActionText,
  onBlockedActionClick,
  showAddGalleryEmpty,
  onAddGalleryClick,
  isGlobalSearchView,
  searchNeedsCriteria,
  searchScope,
  globalPoolLoading,
  hasActiveFilter,
  searchSourceRows,
  filteredRows,
  waterfallStageVisible,
  draggingCards,
  displayList,
  canDragSort,
  effectiveItemSort,
  showGalleryAdminActions,
  getItemStyle,
  handleGalleryLayout,
  handleMasonryReorder,
  handleDragStart,
  handleCardClick,
  handleItemDelete,
  openItemTransfer,
  openItemEdit,
  handleCardView,
  handleItemDownload,
  handleItemCopyLink,
  updateItemMasonryAspectHW,
  onStageDragEnd,
  showPagination,
  orderPersisting,
  orderPersistHint,
  dragSortEditEnabled,
  pageSizeMenuOpen,
  pageSizeSelectValue,
  pageSize,
  pageSizeSelectOptions,
  togglePageSizeMenu,
  selectPageSize,
  currentPage,
  totalPages,
  orderedFilteredRows,
  goPrevPage,
  goNextPage,
  showUploadFloatingPanel,
  siteFileDragActive,
  uploadPanelCollapsed,
  uploadPanelDragging,
  uploadPanelStyle,
  uploadTasks,
  uploadInProgressCount,
  onUploadPanelDragStart,
  onUploadBubbleClick,
  toggleUploadPanel,
  cancelUploadTask,
  onUploadFilesPicked,
  viewerVisible,
  viewerImages,
  viewerLiveVideoUrls,
  viewerImageItemIds,
  viewerInitialIndex,
  videoViewerVisible,
  videoViewerUrls,
  videoViewerItemIds,
  videoViewerInitialIndex,
  fileDetailOpen,
  fileDetailPayload,
  viewerNavCurrent,
  viewerNavTotal,
  viewerToolbarItem,
  viewerAdjacentPrevSrc,
  viewerAdjacentNextSrc,
  canTransferGalleryItem,
  viewerSlideDirection,
  viewerSkipExit,
  startRect,
  handleViewerNavigate,
  handleViewerClose,
  handleVideoViewerClose,
  editDialogOpen,
  editPayload,
  closeItemEdit,
  handleItemEditSaved,
  editTransferOptions,
  handleItemTransferFromEdit,
  transferDialogOpen,
  transferPayload,
  transferDialogOptions,
  transferSubmitting,
  closeItemTransfer,
  handleItemTransferConfirm,
  closeFileDetail,
  handleFileDetailDownload,
  handleFileDetailCopyLink,
  passwordDialogInput,
  passwordDialogError,
  submitFolderPassword,
} = useGalleryViewState(toRef(props, 'folderKey'))

const galleryDialogZIndex = computed(() =>
  viewerVisible.value || videoViewerVisible.value || fileDetailOpen.value
    ? VIEWER_Z.dialog
    : undefined,
)

watch([viewerVisible, videoViewerVisible, fileDetailOpen], ([imageOpen, videoOpen, fileOpen]) => {
  if (!imageOpen && !videoOpen && !fileOpen) viewerToolbarPinned.value = false
})
</script>

<style scoped lang="scss">
.index-box {
  width: 100%;
  height: auto;
  --index-gallery-gap: clamp(4px, 1.85vw + 2px, 16px);
}

.gallery-search-results-head {
  padding: 8px 0 12px;
}

.gallery-search-results-head__title {
  margin: 0;
  font-size: 18px;
  font-weight: 700;
  color: #000;
  letter-spacing: 0.02em;
}

.gallery-search-results-head__meta {
  margin: 6px 0 0;
  font-size: 13px;
  color: #737373;
}
</style>
