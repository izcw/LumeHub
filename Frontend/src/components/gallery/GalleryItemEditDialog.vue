<template>
  <Dialog
    :open="open"
    title="编辑资源"
    fullscreen
    :body-padded="true"
    :show-actions="false"
    :z-index="zIndex"
    @close="emitClose"
  >
    <div
      class="item-edit"
      :class="{
        'is-stack-layout': isStackLayout,
        'is-crop-tab': activeTab === 'crop',
        'is-basic-tab': activeTab === 'basic',
      }"
    >
      <nav class="item-edit__nav-rail" aria-label="编辑导航">
        <div
          v-for="tab in visibleTabs"
          :key="tab.id"
          class="item-edit__nav-btn"
          :class="{ 'is-active': activeTab === tab.id }"
          :title="tab.label"
          role="button"
          tabindex="0"
          @click="activeTab = tab.id"
          @keydown.enter.prevent="activeTab = tab.id"
        >
          <img class="item-edit__nav-icon" :src="tab.icon" :alt="tab.label" width="20" height="20" />
          <span class="item-edit__nav-text">{{ tab.short }}</span>
        </div>
      </nav>

      <div class="item-edit__content">
        <div class="item-edit__content-head">
          <h3 class="item-edit__content-title">{{ activeTabLabel }}</h3>
          <p v-if="activeTabDesc" class="item-edit__content-desc">{{ activeTabDesc }}</p>
        </div>

        <div class="item-edit__content-body">
          <template v-if="activeTab === 'basic'">
            <div class="item-edit__section">
              <div class="info-grid">
                <div v-for="row in infoRows" :key="row.label" class="info-row">
                  <span class="info-row__label">{{ row.label }}</span>
                  <div class="info-row__value-wrap">
                    <span v-if="row.kind === 'tag'" class="info-tag">{{ row.value }}</span>
                    <button
                      v-else-if="row.copyable"
                      type="button"
                      class="info-link"
                      title="点击复制"
                      @click="row.copyKind === 'shareableShortLink' ? copyShareableShortLink() : copyInfoValue(row.value)"
                    >
                      {{ row.value }}
                    </button>
                    <span v-else class="info-row__value" :title="row.value">{{ row.value }}</span>
                  </div>
                </div>
              </div>
            </div>

            <DetailsCollapse v-if="exifRows.length" summary="镜头信息">
              <div class="info-grid info-grid--compact">
                <div v-for="row in exifRows" :key="row.label" class="info-row">
                  <span class="info-row__label">{{ row.label }}</span>
                  <span class="info-row__value">{{ row.value }}</span>
                </div>
              </div>
            </DetailsCollapse>

            <div class="item-edit__divider" aria-hidden="true" />

            <div class="item-edit__section">
              <label class="item-edit__field">
                <span>短链接别名</span>
                <div class="item-edit__link-row">
                  <Input v-model="linkStemModel" type="text" placeholder="例如 123" />
                  <span class="item-edit__link-ext">{{ linkNameExt }}</span>
                </div>
                <!-- <p class="item-edit__hint">后缀由原版文件决定。</p> -->
                <button
                  v-if="shareableShortLinkPreview"
                  type="button"
                  class="item-edit__copy-chip"
                  title="点击复制短链"
                  @click="copyShareableShortLink"
                >
                  {{ shareableShortLinkPreview }}
                </button>
                <p v-else class="item-edit__hint item-edit__hint--placeholder">
                  预览：{{ resourceOrigin }}/resource/别名
                </p>
              </label>
              <label class="item-edit__field">
                <span>标题</span>
                <Input v-model="titleModel" type="text" placeholder="可选" />
              </label>
              <div class="item-edit__field">
                <span>标签</span>
                <div class="tag-picker">
                  <div class="tag-picker__chips">
                    <button
                      v-for="tag in allTagOptions"
                      :key="tag"
                      type="button"
                      class="tag-picker__chip"
                      :class="{ 'is-selected': selectedTags.includes(tag) }"
                      @click="toggleTag(tag)"
                    >
                      {{ tag }}
                    </button>
                  </div>
                  <div class="tag-picker__add">
                    <Input
                      v-model="newTagInput"
                      type="text"
                      placeholder="新增标签，回车添加"
                      @keydown.enter.prevent="addNewTag"
                    />
                    <Button type="info"  native-type="button" @click="addNewTag">添加</Button>
                  </div>
                </div>
              </div>
              <label class="item-edit__field">
                <span>替换原文件</span>
                <div class="item-edit__replace">
                  <div class="item-edit__replace-name">{{ replaceFileName || '未选择文件' }}</div>
                  <Button type="info" native-type="button" @click="openReplacePicker">选择</Button>
                  <input ref="replaceInputRef" class="item-edit__file-input" type="file" @change="onReplaceSelected" />
                </div>
              </label>
              <div v-if="transferOptions.length" class="item-edit__field">
                <span>转移到</span>
                <div class="item-edit__transfer-row">
                  <Select
                    v-model="transferTargetFolderKey"
                    :options="transferOptions"
                    :menu-z-index="selectMenuZIndex"
                    trigger-label="目标画廊"
                    menu-aria-label="目标画廊选项"
                  />
                  <Button
                    type="info"
                    native-type="button"
                    :disabled="!transferTargetFolderKey || transferSubmitting"
                    @click="onTransfer"
                  >
                    {{ transferSubmitting ? '转移中…' : '转移' }}
                  </Button>
                </div>
              </div>
              <div v-if="payload?.isLivePhoto" class="info-note info-note--live">
                实况图包含静图与短视频；编辑保存仅影响静图，实况视频会保留。
              </div>
              <div v-if="payload?.useEdited" class="info-note">
                当前展示编辑版；保存时写入 edited/，不会覆盖 original/ 原图。
              </div>
              <div v-if="folderAccessHint" class="info-note">{{ folderAccessHint }}</div>
            </div>
          </template>

          <template v-else-if="activeTab === 'crop'">
            <div v-if="cropSizeRows.length" class="info-grid item-edit__size-info">
              <div v-for="row in cropSizeRows" :key="row.label" class="info-row">
                <span class="info-row__label">{{ row.label }}</span>
                <span class="info-row__value">{{ row.value }}</span>
              </div>
            </div>

            <div class="item-edit__section">
              <h4 class="item-edit__sub-title">变换</h4>
              <div class="item-edit__toggle-group item-edit__toggle-group--triple">
                <button
                  type="button"
                  class="item-edit__toggle-btn"
                  :class="{ 'is-active': flipH }"
                  @click="toggleFlipHEdit"
                >
                  水平镜像
                </button>
                <button
                  type="button"
                  class="item-edit__toggle-btn"
                  :class="{ 'is-active': flipV }"
                  @click="toggleFlipVEdit"
                >
                  垂直翻转
                </button>
                <button
                  type="button"
                  class="item-edit__toggle-btn"
                  :class="{ 'is-active': rotationQuarter !== 0 }"
                  @click="rotateImageCCW"
                >
                  逆时针旋转
                </button>
              </div>
              <div class="item-edit__panel-actions">
                <Button type="info" size="small" native-type="button" @click="resetTransformEdit">重置变换</Button>
              </div>
            </div>

            <div class="item-edit__section">
              <h4 class="item-edit__sub-title">输出尺寸</h4>
              <div class="item-edit__size-row">
                <label class="item-edit__size-col">
                  <span class="item-edit__size-label">宽</span>
                  <Input v-model="outWidthText" type="text" inputmode="numeric" placeholder="自动" @focus="markEditHistory" />
                </label>
                <span class="item-edit__size-sep" aria-hidden="true">×</span>
                <label class="item-edit__size-col">
                  <span class="item-edit__size-label">高</span>
                  <Input v-model="outHeightText" type="text" inputmode="numeric" placeholder="自动" @focus="markEditHistory" />
                </label>
              </div>
              <label class="item-edit__field item-edit__field--check">
                <Checkbox v-model="lockAspect" @click="markEditHistory">锁定比例</Checkbox>
              </label>
              <div class="item-edit__panel-actions">
                <Button type="info" size="small" native-type="button" @click="syncSizeFromCropEdit">同步裁切区</Button>
                <Button type="info" size="small" native-type="button" @click="clearOutputSizeEdit">清除尺寸</Button>
                <Button type="info" size="small" native-type="button" @click="editor.resetCrop(); repaint()">重置裁切</Button>
              </div>
            </div>

            <div class="item-edit__section">
              <h4 class="item-edit__sub-title">尺寸预设</h4>
              <div class="item-edit__preset-grid">
                <button
                  v-for="preset in allSizePresets"
                  :key="preset.id"
                  type="button"
                  class="item-edit__preset-card"
                  :class="{ 'is-custom': preset.custom }"
                  @click="applyPreset(preset)"
                >
                  <span class="item-edit__preset-name">
                    {{ preset.label }}
                    <span v-if="preset.custom" class="item-edit__preset-badge">自定义</span>
                  </span>
                  <span class="item-edit__preset-size">{{ preset.width }} × {{ preset.height }}</span>
                  <span v-if="preset.desc" class="item-edit__preset-desc">{{ preset.desc }}</span>
                  <span
                    v-if="preset.custom"
                    class="item-edit__preset-remove"
                    role="button"
                    tabindex="0"
                    title="删除预设"
                    @click.stop="removeCustomPreset(preset.id)"
                    @keydown.enter.prevent.stop="removeCustomPreset(preset.id)"
                  >
                    删除
                  </span>
                </button>
              </div>
            </div>

            <DetailsCollapse summary="添加自定义预设">
              <label class="item-edit__field">
                <span>预设名称</span>
                <Input v-model="customPresetLabel" type="text" placeholder="例如 我的证件照" />
              </label>
              <div class="item-edit__size-row">
                <label class="item-edit__size-col">
                  <span class="item-edit__size-label">宽</span>
                  <Input v-model="customPresetWidth" type="text" inputmode="numeric" />
                </label>
                <span class="item-edit__size-sep" aria-hidden="true">×</span>
                <label class="item-edit__size-col">
                  <span class="item-edit__size-label">高</span>
                  <Input v-model="customPresetHeight" type="text" inputmode="numeric" />
                </label>
              </div>
              <div class="item-edit__panel-actions">
                <Button type="info" native-type="button" @click="fillCustomPresetFromCurrent">使用当前</Button>
                <Button native-type="button" @click="saveCustomPreset">保存预设</Button>
              </div>
            </DetailsCollapse>
          </template>

          <template v-else-if="activeTab === 'watermark'">
            <label class="item-edit__field">
              <span>水印文字</span>
              <div class="tag-picker__add">
                <Input v-model="watermarkText" type="text" placeholder="例如 LumeHub" @focus="markEditHistory" />
                <Button type="info"  native-type="button" @click="clearWatermarkView">清空</Button>
              </div>
            </label>

            <div class="item-edit__section">
              <span class="item-edit__field-label">预设</span>
              <div class="item-edit__chip-grid">
                <button
                  v-for="preset in allWatermarkPresets"
                  :key="preset.id"
                  type="button"
                  class="item-edit__chip-btn"
                  :class="{ 'is-custom': preset.custom }"
                  @click="applyWatermarkPreset(preset)"
                >
                  {{ preset.label }}
                  <span
                    v-if="preset.custom"
                    class="item-edit__chip-remove"
                    title="删除"
                    @click.stop="removeCustomWatermark(preset.id)"
                  >
                    ×
                  </span>
                </button>
              </div>
            </div>

            <DetailsCollapse summary="保存当前为自定义预设">
              <div class="tag-picker__add">
                <Input v-model="customWatermarkLabel" type="text" placeholder="预设名称，留空则用当前文字" />
                <Button native-type="button" @click="saveCustomWatermarkPreset">保存</Button>
              </div>
            </DetailsCollapse>

            <label class="item-edit__field item-edit__field--slider">
              <span>字体大小 <em>{{ watermarkFontSize > 0 ? `${watermarkFontSize}px` : '自动' }}</em></span>
              <RangeSlider v-model="watermarkFontSize" :min="0" :max="120" accent-color="#000" @pointerdown="markEditHistory" />
            </label>
            <label class="item-edit__field item-edit__field--slider">
              <span>文字紧凑度 <em>{{ watermarkCompactnessLabel }}</em></span>
              <RangeSlider
                v-model="watermarkCompactness"
                :min="50"
                :max="150"
                accent-color="#000"
                @pointerdown="markEditHistory"
              />
            </label>
            <label class="item-edit__field item-edit__field--slider">
              <span>透明度 <em>{{ watermarkOpacity }}%</em></span>
              <RangeSlider v-model="watermarkOpacity" :min="10" :max="100" accent-color="#000" @pointerdown="markEditHistory" />
            </label>
            <label class="item-edit__field item-edit__field--slider">
              <span>旋转 <em>{{ watermarkRotation }}°</em></span>
              <RangeSlider v-model="watermarkRotation" :min="-180" :max="180" accent-color="#000" @pointerdown="markEditHistory" />
            </label>
            <label class="item-edit__field">
              <span>位置</span>
              <Select
                v-model="watermarkPosition"
                :options="watermarkPositionOptions"
                :menu-z-index="selectMenuZIndex"
              />
            </label>
            <template v-if="watermarkPosition === 'custom'">
              <label class="item-edit__field item-edit__field--slider">
                <span>水平 <em>{{ watermarkCustomX }}%</em></span>
                <RangeSlider v-model="watermarkCustomX" accent-color="#000" @pointerdown="markEditHistory" />
              </label>
              <label class="item-edit__field item-edit__field--slider">
                <span>垂直 <em>{{ watermarkCustomY }}%</em></span>
                <RangeSlider v-model="watermarkCustomY" accent-color="#000" @pointerdown="markEditHistory" />
              </label>
            </template>
          </template>

          <template v-else-if="activeTab === 'export'">
            <p class="item-edit__hint">
              “下载原图”会直接下载原始文件；“下载压缩图片”按当前裁切、尺寸、水印和质量设置导出。
              <template v-if="payload?.fileSize">
                当前原版约 {{ formatFileSize(payload.fileSize) }}。
              </template>
            </p>

            <label class="item-edit__field item-edit__field--slider">
              <span>图片质量 <em>{{ exportQualitySummaryText }}</em></span>
              <RangeSlider v-model="exportQualityPercent" :min="10" :max="100" accent-color="#000" />
            </label>

            <div class="item-edit__export-estimate" aria-live="polite">
              <span class="item-edit__field-label">预估大小</span>
              <span
                class="item-edit__export-estimate-value"
                :class="{ 'is-loading': exportEstimateLoading }"
              >
                {{ exportEstimateDisplay }}
              </span>
              <p class="item-edit__hint item-edit__hint--tight">
                按当前裁切、输出尺寸与水印试编码（{{ exportEstimateFormatHint }}），与磁盘上的原版 JPEG 体积可能相差较大，仅供参考。
              </p>
            </div>

            <div class="item-edit__section">
              <span class="item-edit__field-label">预设</span>
              <div class="item-edit__chip-grid">
                <button
                  v-for="preset in allExportQualityPresets"
                  :key="preset.id"
                  type="button"
                  class="item-edit__chip-btn"
                  :class="{
                    'is-active': isExportQualityPresetActive(preset),
                    'is-custom': preset.custom,
                  }"
                  @click="applyExportQualityPreset(preset)"
                >
                  {{ preset.label }}
                  <span v-if="preset.desc && !preset.custom" class="item-edit__chip-meta">{{ preset.desc }}</span>
                  <span
                    v-if="preset.custom"
                    class="item-edit__chip-remove"
                    title="删除"
                    @click.stop="removeCustomExportQuality(preset.id)"
                  >
                    ×
                  </span>
                </button>
              </div>
            </div>

            <DetailsCollapse summary="保存当前为自定义预设">
              <div class="tag-picker__add">
                <Input v-model="customExportLabel" type="text" placeholder="预设名称" />
                <Button native-type="button" @click="saveCustomExportQualityPreset">保存</Button>
              </div>
            </DetailsCollapse>

            <div class="item-edit__panel-actions item-edit__panel-actions--export">
              <Button
                native-type="button"
                :disabled="!cropReady || saving"
                @click="onExportDownload"
              >
                下载压缩图片
              </Button>
              <Button
                type="info"
                native-type="button"
                :disabled="saving"
                @click="onOriginalDownload"
              >
                下载原图<span v-if="payload?.fileSize">（约 {{ formatFileSize(payload.fileSize) }}）</span>
              </Button>
            </div>
          </template>

          <p v-if="errorText" class="item-edit__error">{{ errorText }}</p>
        </div>

        <div v-if="showContentFooter" class="item-edit__footer">
          <Button
            v-if="showFooterCloseAndSave"
            type="info"
            native-type="button"
            @click="emitClose"
          >
            关闭
          </Button>
          <div v-if="showFooterCloseAndSave || canRevertEdited" class="item-edit__footer-main">
            <Button
              v-if="canRevertEdited"
              type="info"
              native-type="button"
              :disabled="saving"
              @click="onRevertEdited"
            >
              还原原图
            </Button>
            <Button
              v-if="showFooterCloseAndSave"
              native-type="button"
              :disabled="saving || !hasPendingChanges"
              @click="onSave()"
            >
              {{ saving ? '保存中…' : '应用编辑' }}
            </Button>
          </div>
        </div>
      </div>

      <section v-if="isImage" class="item-edit__preview">
        <div class="item-edit__preview-head">
          <div class="item-edit__preview-meta">
            <span class="item-edit__preview-dim">
              <span class="item-edit__preview-dim-label">原图</span>
              <span class="item-edit__preview-dim-value">{{ naturalSizeText }}</span>
            </span>
            <span class="item-edit__preview-divider" aria-hidden="true" />
            <span class="item-edit__preview-dim">
              <span class="item-edit__preview-dim-label">裁切</span>
              <span class="item-edit__preview-dim-value">{{ cropSizeText }}</span>
            </span>
            <template v-if="outputSizeText">
              <span class="item-edit__preview-divider" aria-hidden="true" />
              <span class="item-edit__preview-dim">
                <span class="item-edit__preview-dim-label">输出</span>
                <span class="item-edit__preview-dim-value">{{ outputSizeText }}</span>
              </span>
            </template>
          </div>
          <div class="item-edit__preview-toolbar">
            <div class="item-edit__history">
              <button
                type="button"
                class="item-edit__history-btn"
                :disabled="!canUndo"
                title="撤回"
                aria-label="撤回"
                @click="undoEditStep"
              >
                <img :src="iconRevoke" alt="" width="16" height="16" />
              </button>
              <button
                type="button"
                class="item-edit__history-btn item-edit__history-btn--redo"
                :disabled="!canRedo"
                title="重做"
                aria-label="重做"
                @click="redoEditStep"
              >
                <img :src="iconRevoke" alt="" width="16" height="16" />
              </button>
            </div>
            <template v-if="isZoomed">
              <span class="item-edit__zoom-badge">{{ Math.round(zoom * 100) }}%</span>
              <button type="button" class="item-edit__zoom-reset" @click="resetZoomView">重置视图</button>
            </template>
          </div>
        </div>
        <div class="item-edit__viewport" :class="{ 'is-compact-stage': isStackLayout }">
          <div v-if="showPreviewRulers" class="item-edit__ruler-corner" aria-hidden="true">
            <span class="item-edit__ruler-unit">px</span>
          </div>
          <div
            v-if="showPreviewRulers"
            class="item-edit__ruler-band item-edit__ruler-band--top"
            title="向下拖动创建横向辅助线"
            @pointerdown="onTopRulerPointerDown"
          >
            <div class="item-edit__ruler-track" :style="topRulerTrackStyle">
              <span
                v-for="(mark, idx) in topRulerMarks"
                :key="`rt-${idx}`"
                class="item-edit__ruler-tick"
                :class="{ 'is-major': mark.major }"
                :style="{ left: `${mark.pos}px` }"
              >
                <i class="item-edit__ruler-tick-line" />
                <em v-if="mark.label">{{ mark.label }}</em>
              </span>
            </div>
          </div>
          <div
            v-if="showPreviewRulers"
            class="item-edit__ruler-band item-edit__ruler-band--left"
            title="向右拖动创建纵向辅助线"
            @pointerdown="onLeftRulerPointerDown"
          >
            <div class="item-edit__ruler-track item-edit__ruler-track--v" :style="leftRulerTrackStyle">
              <span
                v-for="(mark, idx) in leftRulerMarks"
                :key="`rl-${idx}`"
                class="item-edit__ruler-tick item-edit__ruler-tick--v"
                :class="{ 'is-major': mark.major }"
                :style="{ top: `${mark.pos}px` }"
              >
                <i class="item-edit__ruler-tick-line" />
                <em v-if="mark.label">{{ mark.label }}</em>
              </span>
            </div>
          </div>
          <div
            ref="stageRef"
            class="item-edit__stage"
            :class="{ 'is-panning': isPanning, 'is-pinching': isPinching }"
            @wheel.prevent="onStageWheel"
            @contextmenu.prevent
            @pointerdown.capture="onStagePointerCapture"
            @pointermove.capture="onStagePointerCaptureMove"
            @pointerup.capture="onStagePointerCaptureEnd"
            @pointercancel.capture="onStagePointerCaptureEnd"
            @pointerdown="onStagePointerDown"
          >
            <div v-if="cropReady" class="item-edit__guides" aria-hidden="true">
              <div
                v-for="guide in rulerGuides"
                :key="guide.id"
                class="item-edit__guide"
                :class="`item-edit__guide--${guide.axis}`"
                :style="guideLineStyle(guide)"
                title="拖动移动 · 双击删除"
                @pointerdown.stop.prevent="onGuideLinePointerDown($event, guide)"
                @dblclick.stop.prevent="removeRulerGuide(guide.id)"
              />
            </div>
            <div v-if="cropReady" class="item-edit__layer" :style="layerStyle">
              <div
                class="item-edit__layer-inner"
                :class="{ 'is-transform-anim': transformAnimating }"
                :style="layerInnerStyle"
              >
                <canvas ref="canvasRef" class="item-edit__canvas" />
                <div
                  v-if="showCropOverlay"
                  class="item-edit__crop"
                  :style="cropBoxStyle"
                  @pointerdown="onCropOverlayPointerDown"
                >
                  <span class="item-edit__crop-shade" aria-hidden="true" />
                  <span class="item-edit__crop-grid" aria-hidden="true" />
                  <span class="item-edit__crop-frame" aria-hidden="true" />
                  <span class="crop-handle crop-handle--n" />
                  <span class="crop-handle crop-handle--s" />
                  <span class="crop-handle crop-handle--e" />
                  <span class="crop-handle crop-handle--w" />
                  <span class="crop-handle crop-handle--nw" />
                  <span class="crop-handle crop-handle--ne" />
                  <span class="crop-handle crop-handle--sw" />
                  <span class="crop-handle crop-handle--se" />
                </div>
              </div>
            </div>
          </div>
        </div>
      </section>

      <section v-else-if="isEditableText" class="item-edit__preview item-edit__preview--markdown">
        <div class="item-edit__preview-head item-edit__preview-head--media">
          <div class="item-edit__preview-meta">
            <span class="item-edit__preview-dim">
              <span class="item-edit__preview-dim-label">类型</span>
              <span class="item-edit__preview-dim-value">{{ isMarkdown ? 'Markdown' : 'TXT' }}</span>
            </span>
            <span class="item-edit__preview-divider" aria-hidden="true" />
            <span class="item-edit__preview-dim">
              <span class="item-edit__preview-dim-label">大小</span>
              <span class="item-edit__preview-dim-value">{{ previewSizeText }}</span>
            </span>
          </div>
          <div class="item-edit__markdown-actions">
            <Button
              v-if="!markdownEditing"
              type="info"
              size="small"
              native-type="button"
              :disabled="markdownLoading || saving"
              @click="markdownEditing = true"
            >
              编辑
            </Button>
            <template v-else>
              <Button
                type="info"
                size="small"
                native-type="button"
                :disabled="saving"
                @click="onMarkdownCancel"
              >
                取消
              </Button>
              <Button
                size="small"
                native-type="button"
                :disabled="saving || markdownLoading"
                @click="onMarkdownSave"
              >
                保存
              </Button>
            </template>
          </div>
        </div>
        <div class="item-edit__markdown-viewport">
          <div v-if="markdownLoading" class="item-edit__markdown-state">正在加载 Markdown…</div>
          <div v-else-if="markdownError" class="item-edit__markdown-state item-edit__markdown-state--error">
            {{ markdownError }}
          </div>
          <MarkdownEditor
            v-if="isMarkdown"
            v-model="markdownContent"
            :editing="markdownEditing"
            height="100%"
          />
          <pre
            v-else-if="!markdownEditing"
            class="item-edit__text-document"
          >{{ markdownContent }}</pre>
          <textarea
            v-else
            v-model="markdownContent"
            class="item-edit__text-document-editor"
            spellcheck="false"
          />
        </div>
      </section>

      <section v-else-if="payload" class="item-edit__preview item-edit__preview--media">
        <div class="item-edit__preview-head item-edit__preview-head--media">
          <div class="item-edit__preview-meta">
            <span class="item-edit__preview-dim">
              <span class="item-edit__preview-dim-label">类型</span>
              <span class="item-edit__preview-dim-value">{{ previewKindLabel }}</span>
            </span>
            <span class="item-edit__preview-divider" aria-hidden="true" />
            <span class="item-edit__preview-dim">
              <span class="item-edit__preview-dim-label">格式</span>
              <span class="item-edit__preview-dim-value">{{ previewFormatLabel }}</span>
            </span>
            <span v-if="previewSizeText !== '—'" class="item-edit__preview-divider" aria-hidden="true" />
            <span v-if="previewSizeText !== '—'" class="item-edit__preview-dim">
              <span class="item-edit__preview-dim-label">大小</span>
              <span class="item-edit__preview-dim-value">{{ previewSizeText }}</span>
            </span>
          </div>
        </div>
        <div class="item-edit__media-viewport">
          <div v-if="isVideoPreview" class="item-edit__media-stage item-edit__media-stage--video">
            <video
              class="item-edit__media-video"
              :src="mediaPreviewSrc"
              :poster="videoPosterSrc || undefined"
              controls
              playsinline
              preload="metadata"
            />
          </div>
          <div v-else class="item-edit__media-stage item-edit__media-stage--file">
            <GalleryFilePreview
              compact
              :src="payload.fullSrc"
              :file-name="previewFileName"
              :format="payload.format"
              :media-kind="payload.mediaKind"
              :file-size="payload.fileSize"
            />
          </div>
        </div>
      </section>
    </div>
  </Dialog>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import Dialog from '@/components/shared-ui/Dialog.vue'
import Input from '@/components/shared-ui/Input.vue'
import Button from '@/components/shared-ui/Button.vue'
import Checkbox from '@/components/shared-ui/Checkbox.vue'
import Select from '@/components/shared-ui/Select.vue'
import RangeSlider from '@/components/shared-ui/RangeSlider.vue'
import DetailsCollapse from '@/components/shared-ui/DetailsCollapse.vue'
import {
  patchCategoryItem,
  patchCategoryItemWithFile,
  revertCategoryItemEdit,
} from '@/api/gallery'
import GalleryFilePreview from '@/components/gallery/GalleryFilePreview.vue'
import MarkdownEditor from '@/components/gallery/MarkdownEditor.vue'
import {
  galleryMediaKindFromUrl,
  galleryExtFromUrl,
  galleryMediaKindLabel,
  type GalleryMediaKind,
} from '@/utils/galleryMedia'
import { formatFileSize } from '@/utils/formatFileSize'
import { readImageExif, exifToDisplayRows, type ExifResult } from '@/utils/imageExif'
import {
  toAbsoluteResourceUrl,
  buildShareableResourceUrl,
  buildShortLinkPreview,
  composeLinkName,
  linkNameExtension,
  linkNameStem,
  resolveEditorOriginalUrl,
  resolveEditorThumbnailUrl,
  resolveLinkStem,
} from '@/utils/resourceUrl'
import iconInformation from '@/assets/icon/information.svg?url'
import iconCropping from '@/assets/icon/cropping.svg?url'
import iconWatermark from '@/assets/icon/watermark.svg?url'
import iconDownload from '@/assets/icon/download.svg?url'
import iconRevoke from '@/assets/icon/revoke.svg?url'
import {
  addCustomSizePreset,
  loadCustomSizePresets,
  removeCustomSizePreset,
} from '@/utils/customSizePresets'
import {
  addCustomWatermarkPreset,
  loadCustomWatermarkPresets,
  removeCustomWatermarkPreset,
} from '@/utils/customWatermarkPresets'
import {
  addCustomExportQualityPreset,
  loadCustomExportQualityPresets,
  removeCustomExportQualityPreset,
} from '@/utils/customExportQualityPresets'
import { buildRulerMarks } from '@/utils/imageRuler'
import { copyTextToClipboard } from '@/utils/clipboard'
import {
  effectiveFolderAccessPolicy,
  folderAccessPolicyHint,
  folderResourceRequiresViewKey,
  lookupGalleryFolder,
} from '@/utils/galleryAccess'
import { getGalleryViewGrant } from '@/utils/galleryViewGrant'
import { linkNameFromResourceUrl } from '@/views/gallery/utils'
import { useMessageStore } from '@/stores/message'
import { useCategoryNavStore } from '@/stores/categoryNav'
import { GALLERY_TAG_OPTIONS } from '@/stores/gallerySearch'
import {
  IMAGE_PRESETS,
  WATERMARK_PRESETS,
  EXPORT_QUALITY_PRESETS,
  exportQualitySummary,
  isExportPresetActive as matchExportQualityPreset,
  useGalleryImageEditor,
  type SizePreset,
  type WatermarkPreset,
  type ExportQualityPreset,
} from '@/composables/useGalleryImageEditor'

export type GalleryItemEditPayload = {
  folderKey: string
  itemId: string
  categoryName?: string
  fullSrc: string
  /** 列表缩略图（视频封面等） */
  thumbSrc?: string
  originalUrl?: string
  editedUrl?: string
  useEdited?: boolean
  isLivePhoto?: boolean
  liveVideoUrl?: string
  linkName?: string
  shortUrl?: string
  title?: string
  tags?: readonly string[]
  uploadedAt?: string
  updatedAt?: string
  fileSize?: number
  format?: string
  mediaKind?: string
}

type EditTab = 'basic' | 'crop' | 'watermark' | 'export'

type InfoRow = {
  label: string
  value: string
  kind?: 'text' | 'tag' | 'tags'
  copyable?: boolean
  copyKind?: 'shareableShortLink'
  tags?: readonly string[]
}

type TransferOption = { label: string; value: string; disabled?: boolean }

const TAB_META: Record<EditTab, { label: string; short: string; icon: string; desc: string }> = {
  basic: { label: '基本信息', short: '基本', icon: iconInformation, desc: '文件属性与元数据' },
  crop: {
    label: '裁切与尺寸',
    short: '裁切',
    icon: iconCropping,
    desc: '裁切范围、输出尺寸、镜像翻转与尺寸预设',
  },
  watermark: { label: '水印', short: '水印', icon: iconWatermark, desc: '文字水印、字号与位置' },
  export: {
    label: '导出',
    short: '导出',
    icon: iconDownload,
    desc: 'JPEG 压缩质量、预设与下载',
  },
}

const props = withDefaults(
  defineProps<{
    open: boolean
    payload: GalleryItemEditPayload | null
    zIndex?: number
    transferOptions?: TransferOption[]
    transferSubmitting?: boolean
  }>(),
  { zIndex: 2700 },
)

const selectMenuZIndex = computed(() => props.zIndex + 50)
const transferOptions = computed(() => props.transferOptions ?? [])
const transferSubmitting = computed(() => props.transferSubmitting === true)
const emit = defineEmits<{
  close: []
  saved: []
  transfer: [targetFolderKey: string]
}>()
const messageStore = useMessageStore()
const categoryNavStore = useCategoryNavStore()

const folderAccessContext = computed(() => {
  const fk = props.payload?.folderKey?.trim()
  if (!fk) return null
  return lookupGalleryFolder(categoryNavStore.doc, fk)
})

const folderAccessHint = computed(() => {
  const ctx = folderAccessContext.value
  if (!ctx) return ''
  return folderAccessPolicyHint(effectiveFolderAccessPolicy(ctx.major, ctx.sub))
})

const folderRequiresViewKey = computed(() => {
  const ctx = folderAccessContext.value
  if (!ctx) return false
  return folderResourceRequiresViewKey(ctx.major, ctx.sub)
})

const shareableShortLinkPreview = computed(() => {
  const preview = shortLinkPreview.value
  if (!preview) return ''
  const fk = props.payload?.folderKey?.trim() ?? ''
  const viewKey = folderRequiresViewKey.value && fk ? getGalleryViewGrant(fk) : ''
  return buildShareableResourceUrl(preview, {
    requiresViewKey: folderRequiresViewKey.value,
    viewKey,
  })
})

const editor = useGalleryImageEditor()
const {
  cropReady,
  cropBoxStyle,
  layerStyle,
  layerInnerStyle,
  transformAnimating,
  naturalSizeText,
  cropSizeText,
  outWidthText,
  outHeightText,
  lockAspect,
  flipH,
  flipV,
  rotationQuarter,
  watermarkText,
  watermarkOpacity,
  watermarkRotation,
  watermarkPosition,
  watermarkCustomX,
  watermarkCustomY,
  watermarkFontSize,
  watermarkCompactness,
  exportQualityPercent,
  exportEncoding,
  crop,
  zoom,
  isZoomed,
  isPanning,
  isPinching,
  canUndo,
  canRedo,
} = editor
const activeTab = ref<EditTab>('basic')
const linkStemModel = ref('')
const titleModel = ref('')
const selectedTags = ref<string[]>([])
const newTagInput = ref('')
const transferTargetFolderKey = ref('')
const markdownContent = ref('')
const markdownOriginalContent = ref('')
const markdownLoading = ref(false)
const markdownError = ref('')
const markdownEditing = ref(false)
const customPresets = ref(loadCustomSizePresets())
const customPresetLabel = ref('')
const customPresetWidth = ref('')
const customPresetHeight = ref('')
const customWatermarkPresets = ref(loadCustomWatermarkPresets())
const customWatermarkLabel = ref('')
const customExportQualityPresets = ref(loadCustomExportQualityPresets())
const customExportLabel = ref('')
const exportEstimateBytes = ref<number | null>(null)
const exportEstimateLoading = ref(false)
let exportEstimateTimer: ReturnType<typeof setTimeout> | null = null
let exportEstimateSeq = 0
const replaceFile = ref<File | null>(null)
const replaceFileName = ref('')
const replaceInputRef = ref<HTMLInputElement | null>(null)
const saving = ref(false)
const errorText = ref('')
const exifData = ref<ExifResult>({})
const stageRef = ref<HTMLDivElement | null>(null)

const stackLayoutMq =
  typeof window !== 'undefined' ? window.matchMedia('(max-width: 960px)') : null
const isStackLayout = ref(stackLayoutMq?.matches ?? false)

function syncStackLayout() {
  isStackLayout.value = stackLayoutMq?.matches ?? false
}

const showPreviewRulers = computed(() => cropReady.value && !isStackLayout.value)
const canvasRef = ref<HTMLCanvasElement | null>(null)
let stageResizeObserver: ResizeObserver | null = null

type RulerGuide = { id: string; axis: 'v' | 'h'; pos: number }

const rulerGuides = ref<RulerGuide[]>([])
let rulerGuideDrag: { id: string; axis: 'v' | 'h' } | null = null

const watermarkPositionOptions = [
  { label: '右下角', value: 'bottom-right' },
  { label: '左下角', value: 'bottom-left' },
  { label: '右上角', value: 'top-right' },
  { label: '左上角', value: 'top-left' },
  { label: '居中', value: 'center' },
  { label: '满屏平铺', value: 'tile' },
  { label: '自定义位置', value: 'custom' },
]

const isImage = computed(() => {
  const src = props.payload?.fullSrc?.trim() ?? ''
  return src !== '' && galleryMediaKindFromUrl(src) === 'image'
})

const isMarkdown = computed(() => {
  const p = props.payload
  const format = (p?.format || galleryExtFromUrl(p?.fullSrc ?? '')).trim().toLowerCase()
  return format === 'md' || format === 'markdown'
})

const isTextDocument = computed(() => {
  const p = props.payload
  const format = (p?.format || galleryExtFromUrl(p?.fullSrc ?? '')).trim().toLowerCase()
  return format === 'txt'
})

const isEditableText = computed(() => isMarkdown.value || isTextDocument.value)

const previewMediaKind = computed((): GalleryMediaKind => {
  const p = props.payload
  const declared = p?.mediaKind?.trim().toLowerCase()
  if (
    declared === 'video' ||
    declared === 'audio' ||
    declared === 'archive' ||
    declared === 'document' ||
    declared === 'other'
  ) {
    return declared
  }
  return galleryMediaKindFromUrl(p?.fullSrc ?? '')
})

const isVideoPreview = computed(() => previewMediaKind.value === 'video')
const supportsThumbnail = computed(() => isImage.value || isVideoPreview.value)

const mediaPreviewSrc = computed(() =>
  toAbsoluteResourceUrl(props.payload?.fullSrc?.trim() || ''),
)

const videoPosterSrc = computed(() => {
  const thumb = props.payload?.thumbSrc?.trim() ?? ''
  if (thumb && galleryMediaKindFromUrl(thumb) === 'image') {
    return toAbsoluteResourceUrl(thumb)
  }
  return ''
})

const previewKindLabel = computed(() => galleryMediaKindLabel(previewMediaKind.value))

const previewFormatLabel = computed(() => {
  const p = props.payload
  return (p?.format || galleryExtFromUrl(p?.fullSrc ?? '')).toUpperCase() || '—'
})

const previewSizeText = computed(() => formatFileSize(props.payload?.fileSize))

const previewFileName = computed(() => {
  const p = props.payload
  return p?.title?.trim() || p?.linkName?.trim() || ''
})

const visibleTabs = computed(() => {
  const ids: EditTab[] = ['basic']
  if (isImage.value) ids.push('crop', 'watermark', 'export')
  return ids.map((id) => ({ id, ...TAB_META[id] }))
})

const showCropOverlay = computed(
  () => cropReady.value && isImage.value && activeTab.value === 'crop',
)

const outputSizeText = computed(() => {
  if (!outWidthText.value && !outHeightText.value) return ''
  return `${outWidthText.value || '自动'} × ${outHeightText.value || '自动'}`
})

const exportQualitySummaryText = computed(() =>
  exportQualitySummary(exportQualityPercent.value, exportEncoding.value),
)

const exportEstimateFormatHint = computed(() =>
  exportEncoding.value === 'png' ? 'PNG 无损' : 'JPEG',
)

function isExportQualityPresetActive(preset: ExportQualityPreset): boolean {
  return matchExportQualityPreset(preset, exportQualityPercent.value, exportEncoding.value)
}

const exportEstimateDisplay = computed(() => {
  if (!cropReady.value) return '—'
  if (exportEstimateLoading.value) return '计算中…'
  if (exportEstimateBytes.value == null) return '—'
  return `约 ${formatFileSize(exportEstimateBytes.value)}`
})

const resourceOrigin = computed(() => window.location.origin)

const originalAbsoluteUrl = computed(() => {
  const p = props.payload
  return toAbsoluteResourceUrl(p?.originalUrl?.trim() || p?.fullSrc?.trim() || '')
})

const thumbnailAbsoluteUrl = computed(() => {
  const p = props.payload
  if (!p) return ''
  return toAbsoluteResourceUrl(resolveEditorThumbnailUrl(p))
})


const linkNameExt = computed(() => {
  const fromOriginal = linkNameExtension(props.payload?.originalUrl || props.payload?.fullSrc || '')
  if (fromOriginal) return fromOriginal
  return linkNameExtension(props.payload?.linkName || '') || '.jpg'
})

const allSizePresets = computed(() => [...IMAGE_PRESETS, ...customPresets.value])

const watermarkCompactnessLabel = computed(() => {
  const c = watermarkCompactness.value
  if (c === 100) return '标准'
  if (c < 100) return `松散 ${c}%`
  return `紧凑 ${c}%`
})

const allWatermarkPresets = computed(() => [...WATERMARK_PRESETS, ...customWatermarkPresets.value])
const allExportQualityPresets = computed(() => [
  ...EXPORT_QUALITY_PRESETS,
  ...customExportQualityPresets.value,
])

const topRulerMarks = computed(() => {
  const img = editor.sourceImg.value
  const d = editor.baseDisplay.value
  if (!img || d.w <= 0) return []
  return buildRulerMarks(img.naturalWidth, d.w * zoom.value)
})

const leftRulerMarks = computed(() => {
  const img = editor.sourceImg.value
  const d = editor.baseDisplay.value
  if (!img || d.h <= 0) return []
  return buildRulerMarks(img.naturalHeight, d.h * zoom.value)
})

const topRulerTrackStyle = computed(() => {
  const d = editor.baseDisplay.value
  return {
    width: `${Math.max(0, d.w * zoom.value)}px`,
    marginLeft: `${d.offsetX + editor.panX.value}px`,
  }
})

const leftRulerTrackStyle = computed(() => {
  const d = editor.baseDisplay.value
  return {
    height: `${Math.max(0, d.h * zoom.value)}px`,
    marginTop: `${d.offsetY + editor.panY.value}px`,
  }
})

function clearRulerGuides() {
  rulerGuides.value = []
  endRulerGuideDrag()
}

function nextRulerGuideId() {
  return `guide-${Date.now()}-${Math.random().toString(36).slice(2, 7)}`
}

function clampGuidePos(axis: 'v' | 'h', pos: number): number {
  const img = editor.sourceImg.value
  if (!img) return 0
  const max = axis === 'v' ? img.naturalWidth : img.naturalHeight
  return Math.min(max, Math.max(0, pos))
}

function roundGuidePos(axis: 'v' | 'h', pos: number): number {
  return clampGuidePos(axis, Math.round(pos))
}

function imageXFromClientX(clientX: number): number {
  const img = editor.sourceImg.value
  const d = editor.baseDisplay.value
  const stage = stageRef.value
  if (!img || !stage || d.w <= 0) return 0
  const rect = stage.getBoundingClientRect()
  const rel = clientX - rect.left - d.offsetX - editor.panX.value
  const displayW = d.w * zoom.value
  if (displayW <= 0) return 0
  return clampGuidePos('v', (rel / displayW) * img.naturalWidth)
}

function imageYFromClientY(clientY: number): number {
  const img = editor.sourceImg.value
  const d = editor.baseDisplay.value
  const stage = stageRef.value
  if (!img || !stage || d.h <= 0) return 0
  const rect = stage.getBoundingClientRect()
  const rel = clientY - rect.top - d.offsetY - editor.panY.value
  const displayH = d.h * zoom.value
  if (displayH <= 0) return 0
  return clampGuidePos('h', (rel / displayH) * img.naturalHeight)
}

function guideLineStyle(guide: RulerGuide): Record<string, string> {
  const img = editor.sourceImg.value
  const d = editor.baseDisplay.value
  if (!img || d.w <= 0 || d.h <= 0) return { display: 'none' }

  if (guide.axis === 'v') {
    const left = d.offsetX + editor.panX.value + (guide.pos / img.naturalWidth) * d.w * zoom.value
    return { left: `${left}px` }
  }
  const top = d.offsetY + editor.panY.value + (guide.pos / img.naturalHeight) * d.h * zoom.value
  return { top: `${top}px` }
}

function endRulerGuideDrag() {
  if (!rulerGuideDrag) return
  rulerGuideDrag = null
  window.removeEventListener('pointermove', onRulerGuidePointerMove)
}

function onRulerGuidePointerMove(event: PointerEvent) {
  if (!rulerGuideDrag) return
  const guide = rulerGuides.value.find((g) => g.id === rulerGuideDrag!.id)
  if (!guide) return
  if (guide.axis === 'v') {
    guide.pos = roundGuidePos('v', imageXFromClientX(event.clientX))
  } else {
    guide.pos = roundGuidePos('h', imageYFromClientY(event.clientY))
  }
}

function onRulerGuidePointerUp(event: PointerEvent) {
  if (!rulerGuideDrag) return
  const guide = rulerGuides.value.find((g) => g.id === rulerGuideDrag!.id)
  const stage = stageRef.value
  if (guide && stage) {
    const rect = stage.getBoundingClientRect()
    const removeH = guide.axis === 'h' && event.clientY < rect.top - 4
    const removeV = guide.axis === 'v' && event.clientX < rect.left - 4
    if (removeH || removeV) removeRulerGuide(guide.id)
  }
  endRulerGuideDrag()
}

function beginRulerGuideDrag(id: string, axis: 'v' | 'h', _pointerId?: number) {
  endRulerGuideDrag()
  rulerGuideDrag = { id, axis }
  window.addEventListener('pointermove', onRulerGuidePointerMove)
  window.addEventListener('pointerup', onRulerGuidePointerUp, { once: true })
  window.addEventListener('pointercancel', onRulerGuidePointerUp, { once: true })
}

function removeRulerGuide(id: string) {
  rulerGuides.value = rulerGuides.value.filter((g) => g.id !== id)
}

function onTopRulerPointerDown(event: PointerEvent) {
  if (event.button !== 0 || !cropReady.value) return
  event.preventDefault()
  const el = event.currentTarget as HTMLElement
  el.setPointerCapture(event.pointerId)
  const id = nextRulerGuideId()
  rulerGuides.value.push({
    id,
    axis: 'h',
    pos: roundGuidePos('h', imageYFromClientY(event.clientY)),
  })
  beginRulerGuideDrag(id, 'h', event.pointerId)
}

function onLeftRulerPointerDown(event: PointerEvent) {
  if (event.button !== 0 || !cropReady.value) return
  event.preventDefault()
  const el = event.currentTarget as HTMLElement
  el.setPointerCapture(event.pointerId)
  const id = nextRulerGuideId()
  rulerGuides.value.push({
    id,
    axis: 'v',
    pos: roundGuidePos('v', imageXFromClientX(event.clientX)),
  })
  beginRulerGuideDrag(id, 'v', event.pointerId)
}

function onGuideLinePointerDown(event: PointerEvent, guide: RulerGuide) {
  if (event.button !== 0) return
  event.preventDefault()
  ;(event.currentTarget as HTMLElement).setPointerCapture(event.pointerId)
  beginRulerGuideDrag(guide.id, guide.axis, event.pointerId)
}

const shortLinkPreview = computed(() => {
  const alias = linkStemModel.value.trim()
  if (!alias) return ''
  return buildShortLinkPreview(alias)
})

const allTagOptions = computed(() => {
  const set = new Set<string>()
  GALLERY_TAG_OPTIONS.forEach((o) => set.add(o.label))
  ;(props.payload?.tags ?? []).forEach((t) => set.add(t))
  selectedTags.value.forEach((t) => set.add(t))
  return [...set]
})

const activeTabLabel = computed(() => TAB_META[activeTab.value].label)
const activeTabDesc = computed(() => TAB_META[activeTab.value].desc)

const canRevertEdited = computed(() => Boolean(props.payload?.useEdited))

function normalizedTags(tags: readonly string[] | undefined): string[] {
  return (tags ?? []).map((tag) => tag.trim()).filter(Boolean)
}

const hasPendingChanges = computed(() => {
  const payload = props.payload
  if (!payload) return false

  const originalLinkStem = resolveLinkStem({
    linkName: payload.linkName,
    shortUrl: payload.shortUrl,
  })
  const originalTags = normalizedTags(payload.tags)
  const currentTags = normalizedTags(selectedTags.value)

  const hasTextChanges =
    isEditableText.value && markdownContent.value !== markdownOriginalContent.value
  const hasMetadataChanges =
    linkStemModel.value.trim() !== originalLinkStem.trim() ||
    titleModel.value.trim() !== (payload.title?.trim() ?? '') ||
    JSON.stringify(currentTags) !== JSON.stringify(originalTags) ||
    replaceFile.value != null ||
    (isImage.value && editor.hasEdits())

  // 文本正文只能通过右侧编辑器的“保存”提交，避免左侧按钮误保存或关闭未保存正文。
  return hasMetadataChanges && !hasTextChanges
})

/** 基本页需要保存元数据；导出页仅负责导出下载 */
const showFooterCloseAndSave = computed(
  () =>
    activeTab.value !== 'export' ||
    replaceFile.value != null,
)

const showContentFooter = computed(
  () => showFooterCloseAndSave.value || canRevertEdited.value || replaceFile.value != null,
)

function formatDate(iso?: string): string {
  const raw = iso?.trim()
  if (!raw) return '—'
  const t = Date.parse(raw)
  if (Number.isNaN(t)) return raw
  return new Date(t).toLocaleString('zh-CN', { hour12: false })
}

const cropSizeRows = computed(() => {
  const rows: { label: string; value: string }[] = [
    { label: '原图', value: naturalSizeText.value },
    { label: '裁切区', value: cropSizeText.value },
  ]
  if (outputSizeText.value) {
    rows.push({ label: '输出', value: outputSizeText.value })
  }
  return rows.filter((r) => r.value && r.value !== '—')
})

const infoRows = computed((): InfoRow[] => {
  const p = props.payload
  const fmt = (p?.format || galleryExtFromUrl(p?.fullSrc ?? '')).toUpperCase() || '—'
  const kind = p?.mediaKind || galleryMediaKindFromUrl(p?.fullSrc ?? '') || '—'
  const original = originalAbsoluteUrl.value || '—'
  const thumbnail = thumbnailAbsoluteUrl.value || '—'
  const rows: InfoRow[] = [
    { label: 'ID', value: p?.itemId || '—' },
    { label: '分类', value: p?.categoryName?.trim() || p?.folderKey || '—' },
    { label: '类型', value: kind },
    { label: '格式', value: fmt, kind: 'tag' },
    {
      label: isImage.value ? '原版大小' : '大小',
      value: formatFileSize(p?.fileSize),
      kind: 'tag',
    },
  ]

  if (isImage.value && naturalSizeText.value !== '—') {
    rows.push({ label: '原图尺寸', value: naturalSizeText.value })
  }
  if (isImage.value) {
    const outputVal =
      outputSizeText.value ||
      (cropSizeText.value !== '—' ? cropSizeText.value : '')
    if (outputVal) {
      rows.push({ label: '输出尺寸', value: outputVal })
    }
  }

  rows.push(
    { label: '上传时间', value: formatDate(p?.uploadedAt) },
    { label: '更新时间', value: formatDate(p?.updatedAt) },
    { label: '原版链接', value: original, copyable: original !== '—' },
    {
      label: '短链接',
      value:
        shareableShortLinkPreview.value ||
        shortLinkPreview.value ||
        (p?.linkName ? buildShortLinkPreview(linkNameStem(p.linkName)) : '—'),
      copyable: Boolean(shareableShortLinkPreview.value || shortLinkPreview.value || p?.linkName),
      copyKind: 'shareableShortLink',
    },
  )

  if (supportsThumbnail.value) {
    rows.splice(rows.length - 1, 0, {
      label: '缩略图链接',
      value: thumbnail,
      copyable: thumbnail !== '—',
    })
  }

  return rows
})

const exifRows = computed(() => {
  const img = editor.sourceImg.value
  return exifToDisplayRows(exifData.value, img ? { w: img.naturalWidth, h: img.naturalHeight } : undefined)
})

function emitClose() {
  if (saving.value) return
  emit('close')
}

function syncLinkStemFromPayload() {
  linkStemModel.value = resolveLinkStem({
    linkName: props.payload?.linkName,
    shortUrl: props.payload?.shortUrl,
  })
}

function resetForm() {
  activeTab.value = 'basic'
  syncLinkStemFromPayload()
  titleModel.value = props.payload?.title?.trim() ?? ''
  selectedTags.value = [...(props.payload?.tags ?? [])]
  newTagInput.value = ''
  transferTargetFolderKey.value = ''
  customPresets.value = loadCustomSizePresets()
  customPresetLabel.value = ''
  customPresetWidth.value = ''
  customPresetHeight.value = ''
  customWatermarkPresets.value = loadCustomWatermarkPresets()
  customWatermarkLabel.value = ''
  customExportQualityPresets.value = loadCustomExportQualityPresets()
  customExportLabel.value = ''
  replaceFile.value = null
  replaceFileName.value = ''
  errorText.value = ''
  exifData.value = {}
  markdownContent.value = ''
  markdownOriginalContent.value = ''
  markdownLoading.value = false
  markdownError.value = ''
  markdownEditing.value = false
  clearRulerGuides()
  editor.resetTransformState()
  clearExportEstimate()
}

function fullLinkName(): string {
  return composeLinkName(linkStemModel.value, linkNameExt.value)
}

async function copyInfoValue(raw: string) {
  const text = raw.trim()
  if (!text || text === '—') return
  const ok = await copyTextToClipboard(text)
  messageStore.show(ok ? '已复制到剪贴板' : '复制失败', ok ? 'success' : 'error')
}

async function copyShareableShortLink() {
  const url = shareableShortLinkPreview.value.trim()
  if (!url) return
  const ok = await copyTextToClipboard(url)
  if (!ok) {
    messageStore.show('复制失败', 'error')
    return
  }
  const fk = props.payload?.folderKey?.trim() ?? ''
  const needsKey = folderRequiresViewKey.value
  const hasGrant = needsKey && fk ? Boolean(getGalleryViewGrant(fk)) : true
  if (needsKey && !hasGrant) {
    messageStore.show('链接已复制；加密相册需在链接后加 ?k=查看密码', 'success')
    return
  }
  messageStore.show('链接已复制', 'success')
}

function markEditHistory() {
  editor.recordEditHistory()
}

function undoEditStep() {
  if (!editor.undoEdit()) return
  repaint()
}

function redoEditStep() {
  if (!editor.redoEdit()) return
  repaint()
}

function syncSizeFromCropEdit() {
  editor.recordEditHistory()
  editor.syncSizeFromCrop()
  repaint()
}

function clearOutputSizeEdit() {
  editor.recordEditHistory()
  clearOutputSize()
  repaint()
}

function resetTransformEdit() {
  editor.recordEditHistory()
  resetTransform()
}

function toggleFlipHEdit() {
  editor.toggleFlipHorizontal(repaint)
}

function toggleFlipVEdit() {
  editor.toggleFlipVertical(repaint)
}

function resetZoomView() {
  editor.resetZoom()
  repaint()
}

function onStagePointerCapture(event: PointerEvent) {
  editor.onStagePointerCaptureDown(event, stageRef.value)
}

function onStagePointerCaptureMove(event: PointerEvent) {
  editor.onStagePointerCaptureMove(event, stageRef.value)
}

function onStagePointerCaptureEnd(event: PointerEvent) {
  editor.onStagePointerCaptureEnd(event, stageRef.value)
}

function onStagePointerDown(event: PointerEvent) {
  if (event.button === 2) {
    editor.onStagePanPointerDown(event)
    return
  }
  if (event.pointerType !== 'mouse' && event.button === 0) {
    const target = event.target
    if (target instanceof HTMLElement && target.closest('.item-edit__crop')) return
    editor.onStagePanPointerDown(event)
  }
}

function onCropOverlayPointerDown(event: PointerEvent) {
  if (isPinching.value) return
  if (event.button === 2) {
    editor.onStagePanPointerDown(event)
    return
  }
  if (event.button === 0) {
    editor.onCropPointerDown(event)
  }
}

function toggleTag(tag: string) {
  const idx = selectedTags.value.indexOf(tag)
  if (idx >= 0) {
    selectedTags.value = selectedTags.value.filter((t) => t !== tag)
  } else {
    selectedTags.value = [...selectedTags.value, tag]
  }
}

function addNewTag() {
  const tag = newTagInput.value.trim()
  if (!tag) return
  if (!selectedTags.value.includes(tag)) {
    selectedTags.value = [...selectedTags.value, tag]
  }
  newTagInput.value = ''
}

function onTransfer() {
  const target = transferTargetFolderKey.value.trim()
  if (!target || transferSubmitting.value) return
  emit('transfer', target)
}

function onMarkdownCancel() {
  markdownContent.value = markdownOriginalContent.value
  markdownEditing.value = false
}

async function onMarkdownSave() {
  await onSave({ includeTextContent: true })
}

function openReplacePicker() {
  replaceInputRef.value?.click()
}

function onReplaceSelected(event: Event) {
  const input = event.target
  if (!(input instanceof HTMLInputElement)) return
  const file = input.files?.[0] ?? null
  replaceFile.value = file
  replaceFileName.value = file?.name ?? ''
  input.value = ''
  if (file && galleryMediaKindFromUrl(file.name) === 'image') {
    void editor.loadSourceImage(URL.createObjectURL(file)).then(() => repaint())
  }
}

function clearOutputSize() {
  outWidthText.value = ''
  outHeightText.value = ''
}

function rotateImageCCW() {
  editor.rotateCCW(repaint)
}

function resetTransform() {
  flipH.value = false
  flipV.value = false
  rotationQuarter.value = 0
  repaint()
}

function applyPreset(preset: SizePreset) {
  editor.applyPreset(preset)
  repaint()
}

function applyWatermarkPreset(preset: WatermarkPreset) {
  editor.applyWatermarkPreset(preset)
  repaint()
}

function clearWatermarkView() {
  editor.clearWatermark()
  repaint()
}

function fillCustomPresetFromCurrent() {
  editor.syncSizeFromCrop()
  customPresetWidth.value = outWidthText.value
  customPresetHeight.value = outHeightText.value
}

function saveCustomPreset() {
  const width = Number.parseInt(customPresetWidth.value.trim(), 10)
  const height = Number.parseInt(customPresetHeight.value.trim(), 10)
  if (!Number.isFinite(width) || !Number.isFinite(height) || width <= 0 || height <= 0) {
    errorText.value = '请输入有效的预设宽高'
    return
  }
  const label = customPresetLabel.value.trim() || '自定义预设'
  customPresets.value = addCustomSizePreset({
    label,
    width,
    height,
    desc: '自定义预设',
  })
  customPresetLabel.value = ''
  errorText.value = ''
}

function removeCustomPreset(id: string) {
  customPresets.value = removeCustomSizePreset(id)
}

function saveCustomWatermarkPreset() {
  const label = customWatermarkLabel.value.trim() || watermarkText.value.trim() || '自定义水印'
  if (!watermarkText.value.trim()) {
    errorText.value = '请先填写水印文字'
    return
  }
  customWatermarkPresets.value = addCustomWatermarkPreset(editor.snapshotWatermarkPreset(label))
  customWatermarkLabel.value = ''
  errorText.value = ''
  messageStore.show('已保存水印预设', 'success')
}

function removeCustomWatermark(id: string) {
  customWatermarkPresets.value = removeCustomWatermarkPreset(id)
}

function clearExportEstimate() {
  exportEstimateSeq += 1
  if (exportEstimateTimer != null) {
    clearTimeout(exportEstimateTimer)
    exportEstimateTimer = null
  }
  exportEstimateLoading.value = false
  exportEstimateBytes.value = null
}

function scheduleExportEstimate() {
  if (exportEstimateTimer != null) clearTimeout(exportEstimateTimer)
  exportEstimateTimer = setTimeout(() => {
    exportEstimateTimer = null
    void refreshExportEstimate()
  }, 400)
}

async function refreshExportEstimate() {
  if (!props.open || !isImage.value || activeTab.value !== 'export' || !cropReady.value) {
    clearExportEstimate()
    return
  }
  const seq = ++exportEstimateSeq
  exportEstimateLoading.value = true
  try {
    const size = await editor.estimateEditedBlobSize()
    if (seq !== exportEstimateSeq) return
    exportEstimateBytes.value = size
  } catch {
    if (seq !== exportEstimateSeq) return
    exportEstimateBytes.value = null
  } finally {
    if (seq === exportEstimateSeq) exportEstimateLoading.value = false
  }
}

function applyExportQualityPreset(preset: ExportQualityPreset) {
  exportQualityPercent.value = preset.percent
  if (preset.encoding) {
    exportEncoding.value = preset.encoding
  } else if (preset.percent < 100) {
    exportEncoding.value = 'jpeg'
  }
  errorText.value = ''
}

function saveCustomExportQualityPreset() {
  const label = customExportLabel.value.trim() || `${exportQualityPercent.value}%`
  const next: ExportQualityPreset = {
    id: `custom-export-${Date.now()}`,
    label,
    percent: exportQualityPercent.value,
    encoding: exportQualityPercent.value >= 100 ? exportEncoding.value : 'jpeg',
    custom: true,
  }
  customExportQualityPresets.value = addCustomExportQualityPreset(next)
  customExportLabel.value = ''
  errorText.value = ''
  messageStore.show('已保存导出质量预设', 'success')
}

function removeCustomExportQuality(id: string) {
  customExportQualityPresets.value = removeCustomExportQualityPreset(id)
}

async function onExportDownload() {
  const payload = props.payload
  if (!payload || !isImage.value) return
  if (!cropReady.value) {
    errorText.value = '请等待图片加载完成'
    return
  }
  saving.value = true
  errorText.value = ''
  try {
    const name = fullLinkName() || payload.linkName || 'export.jpg'
    const file = await editor.buildEditedFile(name)
    if (!file) throw new Error('无法生成导出文件')
    const url = URL.createObjectURL(file)
    const a = document.createElement('a')
    a.href = url
    a.download = file.name
    document.body.appendChild(a)
    a.click()
    a.remove()
    URL.revokeObjectURL(url)
    messageStore.show('已开始下载', 'success')
  } catch (e) {
    errorText.value = e instanceof Error ? e.message : '下载失败'
  } finally {
    saving.value = false
  }
}

function onOriginalDownload() {
  const payload = props.payload
  if (!payload) return
  const url = (payload.originalUrl?.trim() || resolveEditorOriginalUrl(payload)).trim()
  if (!url) return
  const a = document.createElement('a')
  a.href = url
  a.download = payload.linkName || linkNameFromResourceUrl(payload.shortUrl || payload.fullSrc)
  a.target = '_blank'
  a.rel = 'noopener'
  document.body.appendChild(a)
  a.click()
  a.remove()
  messageStore.show('已开始下载', 'success')
}

function onStageWheel(event: WheelEvent) {
  editor.onStageWheel(event, stageRef.value)
  repaint()
}

function repaint() {
  editor.paintStage(canvasRef.value, stageRef.value)
}

async function loadPreview() {
  const payload = props.payload
  if (payload && isEditableText.value) {
    await loadMarkdownPreview(payload)
    return
  }
  if (!payload || !isImage.value) return
  const src = resolveEditorOriginalUrl(payload)
  if (!src || src.includes('/thumb/')) {
    errorText.value = '无法获取原图地址（original/），请刷新后重试'
    return
  }
  const abs = toAbsoluteResourceUrl(src)
  await editor.loadSourceImage(abs)
  editor.clearEditHistory()
  await nextTick()
  repaint()
  void readImageExif(abs).then((d) => {
    exifData.value = d
  })
}

async function loadMarkdownPreview(payload: GalleryItemEditPayload) {
  markdownLoading.value = true
  markdownError.value = ''
  try {
    const url = toAbsoluteResourceUrl(payload.fullSrc.trim())
    const response = await fetch(url)
    if (!response.ok) throw new Error(`加载 Markdown 失败（${response.status}）`)
    const content = await response.text()
    markdownContent.value = content
    markdownOriginalContent.value = content
  } catch (e) {
    markdownError.value = e instanceof Error ? e.message : '加载 Markdown 失败'
  } finally {
    markdownLoading.value = false
  }
}

async function onRevertEdited() {
  const payload = props.payload
  if (!payload) return
  saving.value = true
  errorText.value = ''
  try {
    await revertCategoryItemEdit(payload.folderKey, payload.itemId)
    emit('saved')
    emit('close')
  } catch (e) {
    errorText.value = e instanceof Error ? e.message : '恢复失败'
  } finally {
    saving.value = false
  }
}

async function onSave(options: { includeTextContent?: boolean } = {}) {
  const payload = props.payload
  if (!payload) return
  saving.value = true
  errorText.value = ''
  try {
    const tags = [...selectedTags.value]
    const linkName = fullLinkName()
    const fields = {
      linkName,
      title: titleModel.value.trim(),
      tags: tags.join(','),
    }

    if (replaceFile.value) {
      await patchCategoryItemWithFile(payload.folderKey, payload.itemId, replaceFile.value, fields, {
        saveMode: 'replace',
      })
    } else if (
      options.includeTextContent &&
      isEditableText.value &&
      markdownContent.value !== markdownOriginalContent.value
    ) {
      const filename = fullLinkName() || payload.linkName || (isMarkdown.value ? 'document.md' : 'document.txt')
      const fileType = isMarkdown.value ? 'text/markdown;charset=utf-8' : 'text/plain;charset=utf-8'
      const file = new File([markdownContent.value], filename, { type: fileType })
      await patchCategoryItemWithFile(payload.folderKey, payload.itemId, file, fields, {
        saveMode: 'replace',
      })
    } else if (isImage.value && editor.hasEdits()) {
      const file = await editor.buildEditedFile(fullLinkName() || payload.linkName || 'edited.jpg')
      if (!file) throw new Error('无法生成编辑文件')
      await patchCategoryItemWithFile(payload.folderKey, payload.itemId, file, fields, { saveMode: 'edit' })
    } else {
      await patchCategoryItem(payload.folderKey, payload.itemId, {
        linkName: fields.linkName || undefined,
        title: fields.title,
        tags,
      })
    }
    emit('saved')
    emit('close')
  } catch (e) {
    errorText.value = e instanceof Error ? e.message : '保存失败'
  } finally {
    saving.value = false
  }
}

watch(
  () => [props.payload?.linkName, props.payload?.shortUrl] as const,
  () => {
    if (!props.open) return
    syncLinkStemFromPayload()
  },
)

watch(
  () => [props.open, props.payload?.itemId] as const,
  async ([open]) => {
    if (!open || !props.payload) return
    resetForm()
    await loadPreview()
  },
)

watch(
  () => activeTab.value,
  (tab, prev) => {
    if (!props.open || !isImage.value) return
    if (prev === 'crop') {
      editor.resetZoom()
    }
    if (tab === 'export') {
      scheduleExportEstimate()
    } else if (prev === 'export') {
      clearExportEstimate()
    }
    nextTick(() => repaint())
  },
)

watch(
  () => props.open,
  (open) => {
    if (!open) clearExportEstimate()
  },
)

watch(exportQualityPercent, (pct) => {
  if (!props.open || !isImage.value) return
  if (pct < 100) exportEncoding.value = 'jpeg'
})

watch(
  [
    exportQualityPercent,
    exportEncoding,
    outWidthText,
    outHeightText,
    flipH,
    flipV,
    rotationQuarter,
    watermarkText,
    watermarkOpacity,
    watermarkRotation,
    watermarkPosition,
    watermarkCustomX,
    watermarkCustomY,
    watermarkFontSize,
    watermarkCompactness,
    cropReady,
  ],
  () => {
    if (!props.open || activeTab.value !== 'export') return
    scheduleExportEstimate()
  },
)

watch(
  crop,
  () => {
    if (!props.open || activeTab.value !== 'export') return
    scheduleExportEstimate()
  },
  { deep: true },
)

let outputSizeDebounce: ReturnType<typeof setTimeout> | null = null

watch([outWidthText, outHeightText], (value, oldValue) => {
  if (!props.open || !isImage.value) return
  const [nw, nh] = value
  const [ow, oh] = oldValue ?? ['', '']
  let edited: 'w' | 'h' | null = null
  if (nw !== ow) edited = 'w'
  else if (nh !== oh) edited = 'h'

  const scheduleCropSync = () => {
    if (outputSizeDebounce) clearTimeout(outputSizeDebounce)
    outputSizeDebounce = setTimeout(() => {
      editor.syncCropFromSize()
      repaint()
    }, 280)
  }

  if (edited && editor.consumeAutoFilledOutputField(edited)) {
    scheduleCropSync()
    return
  }

  if (lockAspect.value && edited) {
    editor.syncPairedOutputSize(edited)
  }

  scheduleCropSync()
})

watch(lockAspect, (locked) => {
  if (!props.open || !isImage.value || !locked) return
  editor.syncPairedOutputSize('w')
  if (outputSizeDebounce) clearTimeout(outputSizeDebounce)
  outputSizeDebounce = setTimeout(() => {
    editor.syncCropFromSize()
    repaint()
  }, 280)
})

watch(
  [
    watermarkText,
    watermarkOpacity,
    watermarkRotation,
    watermarkPosition,
    watermarkCustomX,
    watermarkCustomY,
    watermarkFontSize,
    watermarkCompactness,
    zoom,
  ],
  () => {
    if (!props.open || !isImage.value) return
    repaint()
  },
)

function bindStageResizeObserver() {
  if (typeof ResizeObserver === 'undefined') return
  if (!stageResizeObserver) {
    stageResizeObserver = new ResizeObserver(() => {
      if (props.open && isImage.value) repaint()
    })
  }
  stageResizeObserver.disconnect()
  const stage = stageRef.value
  if (stage) stageResizeObserver.observe(stage)
}

onMounted(() => {
  syncStackLayout()
  stackLayoutMq?.addEventListener('change', onStackLayoutChange)
  bindStageResizeObserver()
})

function onStackLayoutChange() {
  syncStackLayout()
  if (props.open && isImage.value) {
    nextTick(() => repaint())
  }
}

watch(
  () => props.open,
  async (open) => {
    if (!open) {
      editor.clearStageInteraction()
      return
    }
    await nextTick()
    bindStageResizeObserver()
    if (isImage.value) repaint()
  },
)

watch(isStackLayout, () => {
  if (!props.open || !isImage.value) return
  nextTick(() => repaint())
})

onBeforeUnmount(() => {
  if (outputSizeDebounce) clearTimeout(outputSizeDebounce)
  clearExportEstimate()
  stackLayoutMq?.removeEventListener('change', onStackLayoutChange)
  clearRulerGuides()
  editor.onCropPointerUp()
  editor.onStagePanPointerUp()
  stageResizeObserver?.disconnect()
})
</script>

<style scoped lang="scss">
$item-edit-accent: #000;
$item-edit-text: #1d1d1f;
$item-edit-muted: #888;
$item-edit-border: #e5e5e5;
$item-edit-surface: #eeeeee;
$item-edit-radius: 6px;
$item-edit-guide-green: rgba(72, 199, 104, 0.95);
$item-edit-crop-accent: #5fd68a;
$item-edit-crop-accent-soft: rgba(95, 214, 138, 0.45);
$item-edit-crop-line: rgba(255, 255, 255, 0.94);
$item-edit-crop-shade: rgba(5, 7, 11, 0.64);

.item-edit {
  display: grid;
  grid-template-columns: 72px minmax(300px, 380px) minmax(0, 1fr);
  height: 100%;
  min-height: 0;
  background: $item-edit-surface;
}

.item-edit.is-basic-tab .item-edit__preview-head {
  display: none;
}

.item-edit.is-basic-tab .item-edit__preview--markdown .item-edit__preview-head {
  display: flex;
}

.item-edit__nav-rail {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding-right: 8px;
  background: #fff;
  border-right: 1px solid $item-edit-border;
}

.item-edit__nav-btn {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 5px;
  padding: 10px 4px;
  border: none;
  border-radius: $item-edit-radius;
  background: transparent;
  cursor: pointer;
  color: $item-edit-muted;
  transition: background 0.18s ease, color 0.18s ease;

  &:hover {
    background: $item-edit-surface;
    color: $item-edit-text;
  }

  &.is-active {
    background: $item-edit-surface;
    color: $item-edit-accent;

    /* &::before {
      content: '';
      position: absolute;
      left: 0;
      top: 10px;
      bottom: 10px;
      width: 2px;
      border-radius: 1px;
      background: $item-edit-accent;
    } */
  }
}

.item-edit__nav-icon {
  display: block;
  width: 20px;
  height: 20px;
  flex-shrink: 0;
  object-fit: contain;
  opacity: 0.55;
  transition: opacity 0.18s ease, filter 0.18s ease;
}

.item-edit__nav-btn:hover .item-edit__nav-icon,
.item-edit__nav-btn.is-active .item-edit__nav-icon {
  opacity: 1;
  filter: brightness(0);
}

.item-edit__nav-text {
  font-size: 11px;
  font-weight: 600;
  line-height: 1.2;
}

.item-edit__content {
  display: flex;
  flex-direction: column;
  min-height: 0;
  background: #fff;
  /* border-right: 1px solid $item-edit-border; */
}

.item-edit__content-head {
  padding: 12px 20px 0;
}

.item-edit__content-title {
  margin: 0;
  font-size: 16px;
  font-weight: 700;
  color: $item-edit-accent;
}

.item-edit__content-desc {
  margin: 4px 0 0;
  font-size: 12px;
  color: $item-edit-muted;
  line-height: 1.45;
}

.item-edit__content-body {
  flex: 1;
  min-height: 0;
  overflow: auto;
  padding: 14px 20px 12px;
}

.item-edit__footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 20px 30px;
  border-top: 1px solid #f0f0f0;
  background: #fff;

  position: sticky;
  bottom: 0;
}

.item-edit__footer-main {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-left: auto;
}

.item-edit__section {
  margin-bottom: 16px;
}

.item-edit__divider {
  height: 1px;
  margin: 4px 0 16px;
  background: #f0f0f0;
}

.item-edit__sub-title {
  margin: 0 0 10px;
  font-size: 12px;
  font-weight: 700;
  color: $item-edit-text;
}

.item-edit__field-label {
  display: block;
  margin-bottom: 8px;
  font-size: 12px;
  font-weight: 600;
  color: $item-edit-text;
}

.info-grid {
  display: grid;
  gap: 10px;
}

.info-grid--compact {
  padding-top: 4px;
}

.info-row {
  display: grid;
  grid-template-columns: 72px 1fr;
  gap: 8px;
  align-items: start;
  font-size: 12px;
}

.info-row__label {
  color: $item-edit-muted;
}

.info-row__value {
  color: $item-edit-text;
  word-break: break-all;
  line-height: 1.45;
}

.info-row__value-wrap {
  min-width: 0;
}

.info-tag {
  display: inline-flex;
  align-items: center;
  padding: 4px 8px;
  font-size: 12px;
  color: #303030;
  background: #eee;
  border-radius: 4px;
  line-height: 1.5;
}

.info-link {
  display: block;
  width: 100%;
  padding: 0;
  border: 0;
  background: none;
  color: $item-edit-text;
  font: inherit;
  text-align: left;
  word-break: break-all;
  line-height: 1.45;
  cursor: pointer;
  transition: opacity 0.15s ease;

  &:hover {
    opacity: 0.65;
  }
}

.info-note {
  margin-top: 4px;
  padding: 10px 12px;
  border-radius: $item-edit-radius;
  background: #fafafa;
  border: 1px solid #ececec;
  color: #666;
  font-size: 12px;
  line-height: 1.5;
}

.item-edit__hint {
  margin: 6px 0 0;
  font-size: 11px;
  color: $item-edit-muted;
  line-height: 1.45;

  code {
    font-family: ui-monospace, monospace;
    font-size: 11px;
    color: #555;
  }

  &--tight {
    margin-top: 4px;
  }
}

.item-edit__export-estimate {
  margin-bottom: 14px;
}

.item-edit__export-estimate-value {
  display: block;
  margin-top: 6px;
  font-size: 18px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
  color: $item-edit-text;

  &.is-loading {
    font-size: 13px;
    font-weight: 500;
    color: $item-edit-muted;
  }
}

.item-edit__link-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.item-edit__link-ext {
  flex-shrink: 0;
  padding: 10px 12px;
  height: 36px;
  display: inline-flex;
  align-items: center;
  border-radius: $item-edit-radius;
  background: $item-edit-surface;
  color: #666;
  font-size: 12px;
  font-family: ui-monospace, monospace;
}

.item-edit__copy-chip {
  display: block;
  width: 100%;
  margin-top: 8px;
  padding: 8px 10px;
  border: 1px solid #ececec;
  border-radius: $item-edit-radius;
  background: $item-edit-surface;
  color: $item-edit-text;
  font-size: 11px;
  font-family: ui-monospace, monospace;
  text-align: left;
  word-break: break-all;
  cursor: pointer;
  transition: background 0.15s ease, border-color 0.15s ease;

  &:hover {
    background: #ececf0;
    border-color: #ddd;
  }
}

.item-edit__layer {
  position: absolute;
  transform-origin: 0 0;
  overflow: visible;
}

.item-edit__layer-inner {
  position: relative;
  width: 100%;
  height: 100%;
  transform-origin: center center;

  &.is-transform-anim {
    transition: transform 0.28s cubic-bezier(0.4, 0, 0.2, 1);
    pointer-events: none;
  }
}

@media (prefers-reduced-motion: reduce) {
  .item-edit__layer-inner.is-transform-anim {
    transition: none;
  }
}

.tag-picker {
  display: grid;
  gap: 10px;
}

.tag-picker__chips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.tag-picker__chip {
  padding: 4px 8px;
  border: 1px solid transparent;
  border-radius: 4px;
  background: #eee;
  color: #303030;
  font-size: 12px;
  cursor: pointer;
  transition: background 0.15s ease, border-color 0.15s ease, color 0.15s ease;

  &:hover {
    background: #e5e5e5;
  }

  &.is-selected {
    border-color: $item-edit-accent;
    background: $item-edit-text;
    color: #fff;
  }
}

.tag-picker__add {
  display: flex;
  gap: 8px;
  align-items: center;
}

.item-edit__field {
  display: block;
  margin-bottom: 12px;

  > span {
    display: block;
    margin-bottom: 6px;
    font-size: 12px;
    font-weight: 600;
    color: $item-edit-text;

    em {
      font-style: normal;
      font-weight: 400;
      color: $item-edit-muted;
    }
  }
}

.item-edit__field--check {
  margin-top: 2px;
}

.item-edit__field--slider {
  margin-bottom: 10px;
}

.item-edit__size-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 24px minmax(0, 1fr);
  column-gap: 8px;
  align-items: end;
  margin-bottom: 12px;
}

.item-edit__size-col {
  display: grid;
  gap: 6px;
  min-width: 0;
}

.item-edit__size-label {
  font-size: 12px;
  font-weight: 600;
  color: $item-edit-text;
}

.item-edit__size-sep {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 40px;
  color: $item-edit-muted;
  font-size: 13px;
  line-height: 1;
  user-select: none;
}

.item-edit__panel-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  margin-top: 4px;
}

.item-edit__toggle-group {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
  margin-bottom: 12px;

  &--triple {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}

.item-edit__toggle-btn {
  padding: 10px 12px;
  border: 1px solid #ccc;
  border-radius: $item-edit-radius;
  background: #fff;
  color: $item-edit-text;
  font-size: 12px;
  cursor: pointer;
  transition:
    background 0.15s ease,
    border-color 0.15s ease,
    color 0.15s ease;

  &:hover {
    background: $item-edit-surface;
  }

  &.is-active {
    border-color: $item-edit-accent;
    background: $item-edit-accent;
    color: #fff;
  }
}

.item-edit__size-info {
  margin-bottom: 16px;
}

.item-edit__preset-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
}

.item-edit__preset-card {
  position: relative;
  padding: 10px 12px;
  border: 1px solid #ececec;
  border-radius: $item-edit-radius;
  background: #fafafa;
  text-align: left;
  cursor: pointer;
  transition:
    border-color 0.15s ease,
    background 0.15s ease,
    box-shadow 0.15s ease;

  &:hover {
    border-color: #ccc;
    background: #fff;
    box-shadow: 0 1px 4px rgba(0, 0, 0, 0.04);
  }

  &.is-custom {
    border-color: #ddd;
    background: #fff;
  }
}

.item-edit__preset-name {
  display: block;
  font-size: 12px;
  font-weight: 600;
  color: $item-edit-text;
}

.item-edit__preset-badge {
  margin-left: 4px;
  padding: 1px 5px;
  border-radius: 999px;
  background: #eee;
  color: #666;
  font-size: 10px;
  font-weight: 600;
}

.item-edit__preset-size {
  display: block;
  margin-top: 4px;
  font-size: 11px;
  color: $item-edit-text;
  font-family: ui-monospace, monospace;
}

.item-edit__preset-desc {
  display: block;
  margin-top: 2px;
  font-size: 11px;
  color: $item-edit-muted;
}

.item-edit__preset-remove {
  display: inline-block;
  margin-top: 8px;
  font-size: 11px;
  color: $item-edit-muted;
  text-decoration: underline;
  cursor: pointer;

  &:hover {
    color: $item-edit-text;
  }
}

.item-edit__chip-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.item-edit__chip-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 6px 10px;
  border: 1px solid #ececec;
  border-radius: 999px;
  background: #fafafa;
  color: $item-edit-text;
  font-size: 12px;
  cursor: pointer;
  transition: background 0.15s ease, border-color 0.15s ease;

  &:hover {
    background: $item-edit-surface;
    border-color: #ccc;
  }

  &.is-active {
    border-color: $item-edit-accent;
    background: $item-edit-surface;
    font-weight: 600;
  }

  &.is-custom {
    padding-right: 6px;
  }
}

.item-edit__chip-meta {
  font-size: 10px;
  color: $item-edit-muted;
}

.item-edit__chip-remove {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  border-radius: 50%;
  font-size: 14px;
  line-height: 1;
  color: $item-edit-muted;

  &:hover {
    background: #ececec;
    color: $item-edit-text;
  }
}

.item-edit__panel-actions--export {
  margin-top: 8px;
}

.item-edit__replace {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.item-edit__transfer-row {
  display: flex;
  align-items: center;
  gap: 8px;

  > :first-child {
    flex: 1;
    min-width: 0;
  }
}

.item-edit__replace-name {
  flex: 1;
  min-width: 100px;
  padding: 10px 12px;
  border: 1px solid #ececec;
  border-radius: $item-edit-radius;
  background: $item-edit-surface;
  font-size: 12px;
  color: #666;
  word-break: break-all;
}

.item-edit__file-input {
  display: none;
}

.item-edit__error {
  margin-top: 10px;
  padding: 8px 10px;
  border-radius: $item-edit-radius;
  background: #fef3f2;
  font-size: 12px;
  color: #b42318;
}

.item-edit__preview {
  display: flex;
  flex-direction: column;
  min-height: 0;
  background: #323232;

  &--media {
    background: $item-edit-surface;
  }
}

.item-edit__preview-head--media {
  background: #fff;
}

.item-edit__preview--markdown {
  background: #fff;
}

.item-edit__preview--markdown .item-edit__preview-head--media {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.item-edit__markdown-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.item-edit__markdown-viewport {
  flex: 1;
  min-height: 0;
  overflow: hidden;
  background: #fff;
}

.item-edit__markdown-state {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  padding: 24px;
  color: $item-edit-muted;
  font-size: 13px;
  text-align: center;
}

.item-edit__markdown-state--error {
  color: #b42318;
}

.item-edit__text-document,
.item-edit__text-document-editor {
  width: 100%;
  height: 100%;
  min-height: 0;
  margin: 0;
  padding: 32px 42px 56px;
  overflow: auto;
  border: 0;
  outline: 0;
  background: #fff;
  color: #252525;
  font: 14px/1.75 ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

.item-edit__text-document-editor {
  resize: none;
}

.item-edit__media-viewport {
  flex: 1;
  min-height: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  /* padding: 16px; */
  background: $item-edit-surface;
}

.item-edit__media-stage {
  width: 100%;
  height: 100%;
  min-height: 0;
  display: flex;
  align-items: center;
  justify-content: center;
}

.item-edit__media-stage--video {
  position: relative;
  max-width: 100%;
  max-height: 100%;
  background: #323232;
  /* border-radius: $item-edit-radius; */
  overflow: hidden;
}

.item-edit__media-video {
  display: block;
  width: 100%;
  max-height: 100%;
  object-fit: contain;
  background: #323232;
}

.item-edit__media-stage--file {
  max-width: 360px;
}

.item-edit__viewport {
  display: grid;
  grid-template-columns: 24px minmax(0, 1fr);
  grid-template-rows: 24px minmax(0, 1fr);
  flex: 1;
  min-height: 0;
  background: #323232;
}

.item-edit__ruler-corner {
  grid-column: 1;
  grid-row: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #2a2a2a;
  border-right: 1px solid #1a1a1a;
  border-bottom: 1px solid #1a1a1a;
  z-index: 4;
}

.item-edit__ruler-unit {
  font-size: 8px;
  font-weight: 600;
  color: #9a9a9a;
  letter-spacing: 0.02em;
}

.item-edit__ruler-band {
  overflow: hidden;
  background: #3d3d3d;
  z-index: 3;
  pointer-events: auto;
  touch-action: none;
}

.item-edit__ruler-band--top {
  grid-column: 2;
  grid-row: 1;
  border-bottom: 1px solid #1a1a1a;
  cursor: row-resize;
}

.item-edit__ruler-band--left {
  grid-column: 1;
  grid-row: 2;
  border-right: 1px solid #1a1a1a;
  cursor: col-resize;
}

.item-edit__ruler-track {
  position: relative;
  height: 100%;
  min-width: 0;
}

.item-edit__ruler-track--v {
  width: 100%;
}

.item-edit__ruler-tick {
  position: absolute;
  top: 0;
  height: 100%;
  display: flex;
  align-items: flex-end;
  transform: translateX(-0.5px);

  em {
    position: absolute;
    top: 3px;
    left: 4px;
    font-style: normal;
    font-size: 10px;
    line-height: 1;
    color: #dcdcdc;
    font-family: ui-monospace, 'Segoe UI', sans-serif;
    white-space: nowrap;
    user-select: none;
    pointer-events: none;
  }
}

.item-edit__ruler-tick--v {
  left: 0;
  width: 100%;
  height: auto;
  align-items: stretch;
  justify-content: flex-end;
  transform: translateY(-0.5px);

  em {
    top: auto;
    left: 3px;
    bottom: 2px;
    writing-mode: vertical-rl;
    text-orientation: mixed;
    transform: rotate(180deg);
  }
}

.item-edit__ruler-tick-line {
  display: block;
  flex-shrink: 0;
  background: #8a8a8a;
}

.item-edit__ruler-band--top .item-edit__ruler-tick-line {
  width: 1px;
  height: 5px;
}

.item-edit__ruler-band--top .item-edit__ruler-tick.is-major .item-edit__ruler-tick-line {
  height: 10px;
  background: #b0b0b0;
}

.item-edit__ruler-band--left .item-edit__ruler-tick-line {
  width: 5px;
  height: 1px;
}

.item-edit__ruler-band--left .item-edit__ruler-tick.is-major .item-edit__ruler-tick-line {
  width: 10px;
  background: #b0b0b0;
}

.item-edit__guides {
  position: absolute;
  inset: 0;
  z-index: 3;
  pointer-events: none;
  overflow: visible;
}

.item-edit__guide {
  position: absolute;
  pointer-events: auto;
  touch-action: none;

  &--v {
    top: 0;
    bottom: 0;
    width: 0;
    margin-left: -1px;
    border-left: 1px solid $item-edit-guide-green;
    box-shadow: 0 0 0 1px rgba(0, 0, 0, 0.28);
    cursor: col-resize;
  }

  &--h {
    left: 0;
    right: 0;
    height: 0;
    margin-top: -1px;
    border-top: 1px solid $item-edit-guide-green;
    box-shadow: 0 0 0 1px rgba(0, 0, 0, 0.28);
    cursor: row-resize;
  }

  &::after {
    content: '';
    position: absolute;
    pointer-events: auto;
  }

  &--v::after {
    left: -5px;
    top: 0;
    bottom: 0;
    width: 11px;
  }

  &--h::after {
    top: -5px;
    left: 0;
    right: 0;
    height: 11px;
  }
}

.item-edit__stage {
  grid-column: 2;
  grid-row: 2;
  position: relative;
  min-height: 280px;
  overflow: hidden;
  touch-action: none;
  background-color: #ececef;
  background-image:
    linear-gradient(45deg, #ddd 25%, transparent 25%),
    linear-gradient(-45deg, #ddd 25%, transparent 25%),
    linear-gradient(45deg, transparent 75%, #ddd 75%),
    linear-gradient(-45deg, transparent 75%, #ddd 75%);
  background-size: 16px 16px;
  background-position:
    0 0,
    0 8px,
    8px -8px,
    -8px 0;

  &.is-panning {
    cursor: grabbing;
  }

  &.is-pinching .item-edit__crop {
    pointer-events: none;
  }
}

.item-edit__preview-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
  padding: 10px 14px;
  min-height: 44px;
  background: #fff;
  border-bottom: 1px solid $item-edit-border;
}

.item-edit__preview-meta {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
  min-width: 0;
}

.item-edit__preview-dim {
  display: inline-flex;
  align-items: baseline;
  gap: 6px;
  min-width: 0;
}

.item-edit__preview-dim-label {
  flex-shrink: 0;
  font-size: 11px;
  color: $item-edit-muted;
}

.item-edit__preview-dim-value {
  font-size: 12px;
  font-weight: 500;
  font-family: ui-monospace, 'Consolas', monospace;
  color: $item-edit-text;
  white-space: nowrap;
}

.item-edit__preview-divider {
  flex-shrink: 0;
  width: 1px;
  height: 12px;
  background: $item-edit-border;
}

.item-edit__preview-toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-left: auto;
  flex-shrink: 0;
}

.item-edit__history {
  display: inline-flex;
  overflow: hidden;
  border: 1px solid $item-edit-border;
  border-radius: $item-edit-radius;
}

.item-edit__history-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 28px;
  margin: 0;
  padding: 0;
  border: none;
  border-radius: 0;
  background: transparent;
  cursor: pointer;
  transition: background 0.15s ease, opacity 0.15s ease;

  &:not(:last-child) {
    border-right: 1px solid $item-edit-border;
  }

  &:hover:not(:disabled) {
    background: #f5f5f7;
  }

  &:disabled {
    opacity: 0.35;
    cursor: not-allowed;
  }

  img {
    display: block;
    width: 15px;
    height: 15px;
  }
}

.item-edit__history-btn--redo img {
  transform: scaleX(-1);
}

.item-edit__zoom-badge {
  padding: 2px 8px;
  border: 1px solid $item-edit-border;
  border-radius: 999px;
  background: $item-edit-surface;
  color: $item-edit-muted;
  font-size: 11px;
  font-family: ui-monospace, monospace;
  line-height: 1.4;
}

.item-edit__zoom-reset {
  margin: 0;
  padding: 0;
  border: none;
  background: none;
  color: $item-edit-muted;
  font-size: 11px;
  line-height: 1.4;
  cursor: pointer;
  text-decoration: underline;
  text-underline-offset: 2px;
  transition: color 0.15s ease;

  &:hover {
    color: $item-edit-text;
  }
}

.item-edit__canvas {
  position: relative;
  display: block;
  width: 100%;
  height: 100%;
  box-shadow: 0 8px 28px rgba(0, 0, 0, 0.12);
}

.item-edit__crop {
  position: absolute;
  border: none;
  background: transparent;
  cursor: move;
  touch-action: none;
  z-index: 2;
  overflow: visible;
}

.item-edit__crop-shade {
  position: absolute;
  inset: 0;
  pointer-events: none;
  box-shadow: 0 0 0 9999px $item-edit-crop-shade;
}

.item-edit__crop-grid {
  position: absolute;
  inset: 0;
  pointer-events: none;
  opacity: 0.38;
  background-image:
    linear-gradient(to right, transparent calc(33.333% - 0.5px), rgba(255, 255, 255, 0.28) calc(33.333% - 0.5px), rgba(255, 255, 255, 0.28) calc(33.333% + 0.5px), transparent calc(33.333% + 0.5px)),
    linear-gradient(to right, transparent calc(66.666% - 0.5px), rgba(255, 255, 255, 0.28) calc(66.666% - 0.5px), rgba(255, 255, 255, 0.28) calc(66.666% + 0.5px), transparent calc(66.666% + 0.5px)),
    linear-gradient(to bottom, transparent calc(33.333% - 0.5px), rgba(255, 255, 255, 0.28) calc(33.333% - 0.5px), rgba(255, 255, 255, 0.28) calc(33.333% + 0.5px), transparent calc(33.333% + 0.5px)),
    linear-gradient(to bottom, transparent calc(66.666% - 0.5px), rgba(255, 255, 255, 0.28) calc(66.666% - 0.5px), rgba(255, 255, 255, 0.28) calc(66.666% + 0.5px), transparent calc(66.666% + 0.5px));
}

.item-edit__crop-frame {
  position: absolute;
  inset: 0;
  pointer-events: none;
  border: 1px solid rgba(255, 255, 255, 0.88);
  box-shadow: 0 0 0 0.5px rgba(0, 0, 0, 0.5);
}

$crop-handle-stroke: 1px;
$crop-handle-arm: 12px;
$crop-corner-hit: 10px;

.crop-handle {
  position: absolute;
  z-index: 2;
  touch-action: none;
}

%crop-handle-line {
  background: rgba(255, 255, 255, 0.92);
  box-shadow: 0 0 0 0.5px rgba(0, 0, 0, 0.45);
  transition: background 0.15s ease, box-shadow 0.15s ease;
}

%crop-handle-line-hover {
  background: #f6fff9;
  box-shadow:
    0 0 0 0.5px rgba(0, 0, 0, 0.45),
    0 0 0 1px rgba(95, 214, 138, 0.55);
}

.crop-handle--nw,
.crop-handle--ne,
.crop-handle--sw,
.crop-handle--se {
  width: $crop-handle-arm + $crop-corner-hit;
  height: $crop-handle-arm + $crop-corner-hit;

  &::before,
  &::after {
    content: '';
    position: absolute;
    left: auto;
    top: auto;
    transform: none;
    border-radius: 0;
    @extend %crop-handle-line;
  }

  &::before {
    width: $crop-handle-arm;
    height: $crop-handle-stroke;
  }

  &::after {
    width: $crop-handle-stroke;
    height: $crop-handle-arm;
  }

  &:hover::before,
  &:hover::after {
    @extend %crop-handle-line-hover;
  }
}

/* 直角顶点贴在选框四角，两臂沿边框延伸 */
.crop-handle--nw {
  left: 0;
  top: 0;
  cursor: nwse-resize;

  &::before,
  &::after {
    left: 0;
    top: 0;
  }
}

.crop-handle--ne {
  right: 0;
  top: 0;
  cursor: nesw-resize;

  &::before,
  &::after {
    right: 0;
    top: 0;
    left: auto;
  }
}

.crop-handle--sw {
  left: 0;
  bottom: 0;
  cursor: nesw-resize;

  &::before,
  &::after {
    left: 0;
    bottom: 0;
    top: auto;
  }
}

.crop-handle--se {
  right: 0;
  bottom: 0;
  cursor: nwse-resize;

  &::before,
  &::after {
    right: 0;
    bottom: 0;
    top: auto;
    left: auto;
  }
}

.crop-handle--n,
.crop-handle--s {
  left: 50%;
  width: 28px;
  height: 12px;
  margin-left: -14px;
  cursor: ns-resize;

  &::after {
    content: '';
    position: absolute;
    left: 50%;
    top: 50%;
    transform: translate(-50%, -50%);
    width: $crop-handle-arm;
    height: $crop-handle-stroke;
    border-radius: 0;
    @extend %crop-handle-line;
  }

  &:hover::after {
    @extend %crop-handle-line-hover;
  }
}

.crop-handle--e,
.crop-handle--w {
  top: 50%;
  width: 12px;
  height: 28px;
  margin-top: -14px;
  cursor: ew-resize;

  &::after {
    content: '';
    position: absolute;
    left: 50%;
    top: 50%;
    transform: translate(-50%, -50%);
    width: $crop-handle-stroke;
    height: $crop-handle-arm;
    border-radius: 0;
    @extend %crop-handle-line;
  }

  &:hover::after {
    @extend %crop-handle-line-hover;
  }
}

.crop-handle--n {
  top: 0;
  margin-top: -6px;
}

.crop-handle--s {
  bottom: 0;
  margin-bottom: -6px;
}

.crop-handle--e {
  right: 0;
  margin-right: -6px;
}

.crop-handle--w {
  left: 0;
  margin-left: -6px;
}

// 窄屏堆叠布局（与 is-stack-layout / max-width: 960px 同步）
.item-edit.is-stack-layout {
  grid-template-columns: 60px minmax(0, 1fr);
  grid-template-rows: minmax(220px, 0.9fr) minmax(0, 1fr);
  height: 100%;
  min-height: 0;
  overflow: hidden;

  // &:not(.is-crop-tab) {
  //   grid-template-rows: minmax(140px, 0.82fr) minmax(0, 1fr);
  // }

  // &.is-basic-tab {
  //   grid-template-rows: minmax(96px, 0.58fr) minmax(0, 1fr);
  // }

  .item-edit__preview {
    position: relative;
    display: flex;
    flex-direction: column;
    grid-column: 1 / -1;
    grid-row: 1;
    height: 100%;
    min-height: 0;
    max-height: none;
    overflow: hidden;
    isolation: isolate;
  }

  // 尺寸信息改在下方裁切面板展示，预览区只保留浮动工具条
  .item-edit__preview-meta {
    display: none;
  }

  .item-edit__preview-head {
    position: absolute;
    left: 0;
    right: 0;
    bottom: 0;
    z-index: 6;
    flex-direction: row;
    align-items: center;
    justify-content: flex-end;
    gap: 8px;
    margin: 0;
    padding: 6px 10px;
    min-height: 0;
    background: rgba(255, 255, 255, 0.94);
    border-top: 1px solid $item-edit-border;
    border-bottom: none;
    backdrop-filter: blur(8px);
  }

  .item-edit__preview-toolbar {
    margin-left: 0;
    flex-wrap: nowrap;
    gap: 8px;
  }

  .item-edit__preview--media .item-edit__preview-head {
    display: none;
  }

  .item-edit__viewport,
  .item-edit__media-viewport {
    flex: 1;
    min-height: 0;
    overflow: hidden;
    touch-action: none;
  }

  .item-edit__viewport.is-compact-stage {
    display: block;

    .item-edit__stage {
      width: 100%;
      height: 100%;
      min-height: 0;
    }
  }

  .item-edit__stage {
    min-height: 0;
  }

  .item-edit__nav-rail {
    grid-row: 2;
    grid-column: 1;
    min-height: 0;
    overflow-y: auto;
    padding: 10px 6px;
  }

  .item-edit__content {
    grid-row: 2;
    grid-column: 2;
    min-height: 0;
    overflow: hidden;
  }

  .item-edit__content-head {
    padding: 12px 16px 0;
  }

  .item-edit__content-desc {
    display: none;
  }

  .item-edit__content-body {
    padding: 10px 16px 8px;
  }

  .item-edit__footer {
    z-index: 2;
    box-shadow: 0 -6px 16px rgba(0, 0, 0, 0.05);
  }

  .item-edit__media-stage--file {
    max-width: 100%;
    width: 100%;
    height: 100%;
  }

  .item-edit__media-video {
    width: auto;
    max-width: 100%;
    height: auto;
    max-height: 100%;
  }
}

@media (max-width: 640px) {
  .item-edit.is-stack-layout {
    grid-template-columns: 52px minmax(0, 1fr);
    grid-template-rows: minmax(220px, 0.9fr) minmax(0, 1fr);

    // &:not(.is-crop-tab) {
    //   grid-template-rows: minmax(130px, 0.78fr) minmax(0, 1fr);
    // }

    // &.is-basic-tab {
    //   grid-template-rows: minmax(88px, 0.5fr) minmax(0, 1fr);
    // }
  }
}

:deep(.dialog-body) {
  overflow: hidden;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

:deep(.dialog-header) {
  @media (max-width: 640px) {
    padding: 16px 20px 0;
  }
}

:deep(.dialog-title) {
  @media (max-width: 640px) {
    font-size: 17px;
    line-height: 28px;
  }
}
</style>
