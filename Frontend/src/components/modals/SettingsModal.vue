<template>
  <Dialog
    :open="profileModalOpen"
    :show-actions="false"
    :body-padded="false"
    width="930px"
    height="650px"
    :z-index="2600"
    @close="close"
  >
    <div class="settings-modal">
      <div class="content-box" :class="{ 'is-mobile': isMobileSettings }" @click.stop>
        <div v-show="!isMobileSettings || mobileView === 'menu'" class="sidebar">
          <div class="avatar-box">
            <div class="avatar">
              <img :src="avatarPreview || fallbackAvatar" alt="avatar" />
            </div>
            <div class="info">
              <p class="text">{{ userName }}</p>
              <p class="email">{{ userSubText }}</p>
            </div>
          </div>
          <div class="menu-box">
            <div class="menu-item" :class="{ active: tab === 'me' }" @click="onMenuSelect('me')">
              <img class="menu-item__icon" :src="personIconSrc" alt="" aria-hidden="true" />
              <span class="menu-item__text">个人资料</span>
            </div>
            <div
              class="menu-item"
              :class="{ active: tab === 'settings' }"
              @click="onMenuSelect('settings')"
            >
              <img class="menu-item__icon" :src="privacyIconSrc" alt="" aria-hidden="true" />
              <span class="menu-item__text">隐私设置</span>
            </div>
            <div
              class="menu-item"
              :class="{ active: tab === 'security' }"
              @click="onMenuSelect('security')"
            >
              <img class="menu-item__icon" :src="lockIconSrc" alt="" aria-hidden="true" />
              <span class="menu-item__text">安全</span>
            </div>
            <div
              v-if="canManageAccounts"
              class="menu-item"
              :class="{ active: tab === 'accounts' }"
              @click="onMenuSelect('accounts')"
            >
              <img class="menu-item__icon" :src="userManagementIconSrc" alt="" aria-hidden="true" />
              <span class="menu-item__text">用户管理</span>
            </div>
            <div
              v-if="canManageAccounts"
              class="menu-item"
              :class="{ active: tab === 'storage' }"
              @click="onMenuSelect('storage')"
            >
              <img class="menu-item__icon" :src="storageIconSrc" alt="" aria-hidden="true" />
              <span class="menu-item__text">存储管理</span>
            </div>
            <div
              v-if="canManageAccounts"
              class="menu-item"
              :class="{ active: tab === 'recycle' }"
              @click="onMenuSelect('recycle')"
            >
              <img class="menu-item__icon" :src="recycleIconSrc" alt="" aria-hidden="true" />
              <span class="menu-item__text">回收站</span>
            </div>
          </div>
          <div class="footer-box">
            <div class="footer-item" @click="onLogout">退出登录</div>
          </div>
        </div>
        <div v-show="!isMobileSettings || mobileView === 'content'" class="content">
          <div class="title">
            <button
              v-if="isMobileSettings"
              type="button"
              class="mobile-back-btn"
              aria-label="返回"
              @click="goBackToMenu"
            >
              <img :src="leftIconSrc" alt="" aria-hidden="true" />
            </button>
            <p>{{ mainTitle }}</p>
          </div>
          <div class="panel-box">
            <template v-if="tab === 'me'">
              <div class="setting-item avatar-setting">
                <div class="item-left">
                  <p class="item-title">头像</p>
                  <p class="item-value">支持 PNG、JPG、WEBP，最大 10MB</p>
                  <div class="avatar-upload-row">
                    <input
                      ref="avatarInputRef"
                      type="file"
                      class="avatar-file-input"
                      accept="image/png,image/jpeg,image/webp"
                      @change="onAvatarFileChange"
                    />
                    <button
                      type="button"
                      class="avatar-upload-preview"
                      :disabled="savingAvatar"
                      aria-label="上传头像"
                      @click="triggerAvatarPick"
                    >
                      <img :src="avatarPreview || fallbackAvatar" alt="avatar" />
                      <span
                        class="avatar-upload-overlay"
                        :class="{ 'is-visible': savingAvatar }"
                        aria-hidden="true"
                      >
                        <img
                          v-if="!savingAvatar"
                          class="avatar-upload-edit-icon"
                          :src="editIconSrc"
                          alt=""
                        />
                        <span v-else class="avatar-upload-loading">上传中</span>
                      </span>
                    </button>
                  </div>
                </div>
              </div>
              <div class="setting-item mt-16">
                <div class="item-left">
                  <p class="item-title">用户名</p>
                  <Input
                    v-model="draftDisplayName"
                    type="text"
                    class="form-input"
                    placeholder="请输入账户名称"
                  />
                </div>
                <div class="item-right">
                  <Button
                    size="small"
                    class="item-button"
                    type="info"
                    native-type="button"
                    :disabled="savingProfile || !displayNameChanged"
                    @click="saveDisplayNameIfNeeded"
                  >
                    {{ savingProfile ? '保存中...' : '修改' }}
                  </Button>
                </div>
              </div>

              <div class="setting-item mt-16">
                <div class="item-left">
                  <p class="item-title">邮箱（登录账户）</p>
                  <Input
                    v-model="draftUsername"
                    type="email"
                    class="form-input"
                    placeholder="请输入邮箱"
                  />
                </div>
                <div class="item-right">
                  <Button
                    size="small"
                    class="item-button"
                    type="info"
                    native-type="button"
                    :disabled="savingProfile || !usernameChanged"
                    @click="onEditEmailClick"
                  >
                    {{ savingProfile ? '保存中...' : '修改' }}
                  </Button>
                </div>
              </div>

              <div class="setting-item mt-16" style="margin-bottom: 1rem">
                <div class="item-left">
                  <p class="item-title">密码</p>
                  <p class="item-value">已加密存储</p>
                </div>
                <div class="item-right">
                  <Button
                    size="small"
                    class="item-button"
                    type="info"
                    native-type="button"
                    @click="editPassword = !editPassword"
                  >
                    {{ editPassword ? '取消' : '修改' }}
                  </Button>
                </div>
              </div>
              <div v-if="editPassword">
                <div class="account-edit-pwd-box">
                  <Input
                    v-model="currentPasswordForPassword"
                    type="password"
                    class="form-input"
                    placeholder="请输入当前密码"
                  />
                  <Input
                    v-model="draftNewPassword"
                    type="password"
                    class="form-input"
                    placeholder="新密码至少 6 位"
                  />
                  <Input
                    v-model="draftConfirmPassword"
                    type="password"
                    class="form-input"
                    placeholder="再次输入新密码"
                  />
                </div>
                <div class="form-actions">
                  <Button
                    size="small"
                    class="form-button primary"
                    :disabled="savingPassword"
                    native-type="button"
                    @click="onUpdatePassword"
                  >
                    {{ savingPassword ? '更新中...' : '更新密码' }}
                  </Button>
                </div>
              </div>
            </template>

            <template v-if="tab === 'settings'">
              <div class="privacy-box">
                <div class="privacy-head">
                  <p class="privacy-title">目录访问策略</p>
                  <div class="privacy-head-actions">
                    <Button
                      size="small"
                      class="table-action-btn"
                      type="info"
                      native-type="button"
                      @click="openAddPrivacyDialog"
                    >
                      添加导航
                    </Button>
                    <Button
                      size="small"
                      class="table-action-btn"
                      native-type="button"
                      @click="openAddGalleryDialog"
                    >
                      添加画廊
                    </Button>
                  </div>
                </div>
                <p v-if="!canManageLayout" class="item-value">当前账号无目录管理权限。</p>
                <p v-else-if="!navDraft" class="item-value">正在准备编辑数据...</p>
                <Table
                  v-else
                  v-model:page="privacyTablePage"
                  :columns="privacyTableColumns"
                  :rows="privacyTableRows"
                  row-key="id"
                  tree
                  tree-column-key="name"
                  :max-height="460"
                  paginated
                  :page-size="12"
                >
                  <template #cell-type="{ row }">
                    {{ row.scope === 'major' ? '导航' : '画廊' }}
                  </template>
                  <template #cell-folderKey="{ row }">
                    <span class="is-mono">{{ row.folderKey || '-' }}</span>
                  </template>
                  <template #cell-actions="{ row }">
                    <div class="table-actions">
                      <Button
                        size="small"
                        class="table-action-btn"
                        type="info"
                        native-type="button"
                        @click="onPrivacyRowEdit(asPrivacyTableRow(row))"
                      >
                        编辑
                      </Button>
                      <Popconfirm
                        :title="
                          asPrivacyTableRow(row).scope === 'major'
                            ? '确认删除该导航吗？其下所有画廊的文件将移入回收站。'
                            : '确认删除该画廊吗？目录内文件将移入回收站。'
                        "
                        confirm-text="删除"
                        @confirm="onPrivacyRowDelete(asPrivacyTableRow(row))"
                      >
                        <template #trigger>
                          <Button
                            size="small"
                            class="table-action-btn"
                            native-type="button"
                            :disabled="deletingCategory"
                          >
                            删除
                          </Button>
                        </template>
                      </Popconfirm>
                    </div>
                  </template>
                </Table>
              </div>
            </template>

            <template v-if="tab === 'security'">
              <div class="security-box">
                <div class="security-head">
                  <p class="item-title">通行证（Passkey）</p>
                  <Button
                    size="small"
                    class="table-action-btn"
                    native-type="button"
                    :disabled="bindingPasskey"
                    @click="onBindPasskey"
                  >
                    {{ bindingPasskey ? '绑定中...' : '添加通行证' }}
                  </Button>
                </div>
                <p class="item-value security-desc">
                  用于 iPhone 扫码后通过 Face ID 快速完成登录确认。
                </p>
                <p v-if="passkeysLoading" class="item-value">通行证加载中...</p>
                <p v-else-if="passkeys.length === 0" class="item-value">当前账号尚未绑定通行证。</p>
                <div v-else class="passkey-list">
                  <div v-for="item in passkeys" :key="item.id" class="passkey-row">
                    <div class="passkey-main">
                      <p class="passkey-title">{{ item.label || '已绑定通行证' }}</p>
                      <p class="passkey-sub">ID: {{ item.displayId }}</p>
                    </div>
                    <div class="passkey-meta">
                      <span>创建：{{ formatDateText(item.createdAt) }}</span>
                      <span>最近使用：{{ formatDateText(item.lastUsedAt) }}</span>
                    </div>
                  </div>
                </div>
              </div>
            </template>

            <template v-if="tab === 'storage' && canManageAccounts">
              <div class="storage-box">
                <p class="privacy-title">存储管理</p>
                <p class="item-value storage-desc">
                  单文件大小不设上限；总占用不得超过下方配额。大文件自动分片上传；视频封面由浏览器截取第一帧上传，服务器无需安装额外软件。
                </p>
                <div v-if="storageLoadErr" class="status-message is-error">
                  {{ storageLoadErr }}
                </div>
                <div v-else-if="!storageStatus" class="item-value">加载中...</div>
                <template v-else>
                  <StoragePieChart
                    :used-bytes="storageStatus.usedBytes"
                    :quota-bytes="storageStatus.quotaBytes"
                    :available-bytes="storageStatus.availableBytes"
                    :used-percent="storageStatus.usedPercent"
                  />
                  <p v-if="storageStatus.calculatedAt" class="storage-calculated-at">
                    统计时间：{{ formatDateText(storageStatus.calculatedAt) }}
                  </p>
                  <div class="setting-item mt-16">
                    <div class="item-left">
                      <p class="item-title">存储配额（GB）</p>
                      <Input
                        v-model="storageQuotaGbDraft"
                        type="number"
                        class="form-input storage-quota-input"
                        placeholder="默认 30"
                        min="1"
                        step="1"
                      />
                    </div>
                    <div class="item-right storage-actions">
                      <Button
                        size="small"
                        class="item-button"
                        type="info"
                        native-type="button"
                        :disabled="savingStorageQuota || !storageQuotaChanged"
                        @click="onSaveStorageQuota"
                      >
                        {{ savingStorageQuota ? '保存中...' : '保存配额' }}
                      </Button>
                      <Button
                        size="small"
                        class="table-action-btn"
                        native-type="button"
                        :disabled="recalculatingStorage"
                        @click="onRecalculateStorage"
                      >
                        {{ recalculatingStorage ? '统计中...' : '重新统计' }}
                      </Button>
                    </div>
                  </div>
                </template>
              </div>
            </template>

            <template v-if="tab === 'accounts' && canManageAccounts">
              <div class="privacy-head">
                <p class="privacy-title">用户管理</p>
                <Button
                  size="small"
                  class="table-action-btn"
                  native-type="button"
                  @click="onClickAddAccount"
                >
                  添加
                </Button>
              </div>
              <div v-if="accountsLoadErr" class="status-message is-error">
                {{ accountsLoadErr }}
              </div>
              <div v-else-if="!accountsRows" class="item-value">加载中...</div>
              <Table
                v-else
                v-model:page="accountTablePage"
                :columns="accountTableColumns"
                :rows="accountTableRows"
                row-key="id"
                :max-height="400"
                paginated
                :page-size="10"
              >
                <template #cell-actions="{ row }">
                  <div class="table-actions">
                    <Button
                      size="small"
                      class="table-action-btn"
                      type="info"
                      native-type="button"
                      @click="openAccountEditDialog(asAccountRaw(row))"
                    >
                      编辑
                    </Button>
                    <Popconfirm
                      title="确认删除该用户吗？"
                      confirm-text="删除"
                      @confirm="onDeleteAccount(asAccountRaw(row))"
                    >
                      <template #trigger>
                        <Button size="small" class="table-action-btn" native-type="button"
                          >删除</Button
                        >
                      </template>
                    </Popconfirm>
                  </div>
                </template>
              </Table>
            </template>

            <template v-if="tab === 'recycle' && canManageAccounts">
              <RecycleBinPanel ref="recycleBinRef" />
            </template>
          </div>
        </div>
      </div>
    </div>
  </Dialog>

  <Dialog
    :open="emailPwdDialogVisible"
    title="验证当前密码"
    :show-close="false"
    :close-on-mask="false"
    cancel-text="取消"
    :confirm-text="savingProfile ? '保存中...' : '确认'"
    :confirm-disabled="savingProfile"
    width="380px"
    height="250px"
    :z-index="2700"
    @cancel="cancelEmailPwdDialog"
    @confirm="confirmEmailPwdDialog"
    @close="cancelEmailPwdDialog"
  >
    <div class="inline-dialog">
      <p class="inline-dialog-desc">修改登录邮箱须填写您的当前密码</p>
      <Input
        v-model="emailPwdInput"
        type="password"
        class="form-input"
        placeholder="请输入当前密码"
        @keydown.enter.prevent="confirmEmailPwdDialog"
      />
      <p v-if="emailPwdDialogError" class="inline-dialog-error">{{ emailPwdDialogError }}</p>
    </div>
  </Dialog>

  <Dialog
    :open="encryptedPwdDialogVisible"
    title="设置查看密码"
    :show-close="false"
    :close-on-mask="false"
    cancel-text="取消"
    confirm-text="确定"
    width="380px"
    height="250px"
    @cancel="cancelEncryptedPwdDialog"
    @confirm="confirmEncryptedPwdDialog"
    @close="cancelEncryptedPwdDialog"
  >
    <div class="inline-dialog">
      <p class="inline-dialog-desc">
        请为「{{ encryptedPwdDialogName }}」设置查看密码（至少 4 位）
      </p>
      <Input
        v-model="encryptedPwdInput"
        type="password"
        class="form-input"
        placeholder="请输入查看密码"
        @keydown.enter.prevent="confirmEncryptedPwdDialog"
      />
      <p v-if="encryptedPwdError" class="inline-dialog-error">{{ encryptedPwdError }}</p>
    </div>
  </Dialog>

  <NavAddModal
    :use-store="false"
    :open="privacyEditVisible"
    :title="privacyEditTargetScope === 'major' ? '编辑导航' : '编辑画廊'"
    :confirm-text="privacyEditing ? '保存中...' : '保存'"
    :confirm-disabled="privacyEditing"
    :mode="privacyEditTargetScope === 'major' ? 'primary' : 'gallery'"
    :show-folder-key="privacyEditTargetScope === 'sub'"
    :show-public="false"
    :show-major-select="privacyEditTargetScope === 'sub'"
    :major-options="addGalleryMajorOptions"
    v-model:major-id="privacyEditMajorIdStr"
    v-model:major-name="privacyEditName"
    v-model:gallery-name="privacyEditName"
    v-model:folder-key-value="privacyEditFolderKey"
    width="420px"
    :height="privacyEditTargetScope === 'major' ? '380px' : '520px'"
    @cancel="closePrivacyEditDialog"
    @confirm="savePrivacyEditDialog"
    @close="closePrivacyEditDialog"
  >
    <template #extra>
      <label class="nav-add__label">访问策略</label>
      <Select v-model="privacyEditPolicy" :options="accessPolicySelectOptions" />
      <template v-if="privacyEditIsEncryptedPolicy">
        <label class="nav-add__label">查看密码</label>
        <Input
          :model-value="privacyEditEncryptedPassword"
          type="password"
          class="form-input"
          placeholder="请输入查看密码（至少 4 位）"
          @focus="onPrivacyEditPasswordFocus"
          @update:model-value="onPrivacyEditPasswordModelUpdate"
        />
        <p
          v-if="privacyEditPasswordHint"
          class="inline-dialog-hint"
          :class="{ 'is-warn': privacyEditPasswordNeedsInput }"
        >
          {{ privacyEditPasswordHint }}
        </p>
      </template>
      <p v-if="privacyEditError" class="inline-dialog-error">{{ privacyEditError }}</p>
    </template>
  </NavAddModal>

  <NavAddModal
    :use-store="false"
    :open="addGalleryVisible"
    title="添加画廊"
    :confirm-text="addingGallery ? '创建中...' : '创建'"
    :confirm-disabled="addingGallery"
    mode="gallery"
    :show-major-select="true"
    :major-options="addGalleryMajorOptions"
    v-model:major-id="addGalleryMajorId"
    v-model:gallery-name="addGalleryName"
    v-model:folder-key-value="addGalleryFolderKey"
    v-model:public-value="addGalleryPublic"
    width="420px"
    height="420px"
    @cancel="closeAddGalleryDialog"
    @confirm="submitAddGalleryDialog"
    @close="closeAddGalleryDialog"
  >
    <template #extra>
      <p v-if="addGalleryError" class="inline-dialog-error">{{ addGalleryError }}</p>
    </template>
  </NavAddModal>

  <Dialog
    :open="accountEditVisible"
    title="编辑用户"
    cancel-text="取消"
    :confirm-text="savingAccountId === accountEditId ? '保存中...' : '保存'"
    :confirm-disabled="savingAccountId === accountEditId"
    :z-index="2700"
    width="460px"
    :height="accountEditDialogHeight"
    @cancel="closeAccountEditDialog"
    @confirm="saveAccountEditDialog"
    @close="closeAccountEditDialog"
  >
    <div class="inline-dialog">
      <label class="nav-add__label">用户名</label>
      <Input v-model="accountEditDisplayName" class="form-input" placeholder="请输入用户名" />
      <label class="nav-add__label">邮箱</label>
      <Input v-model="accountEditEmail" type="email" class="form-input" placeholder="请输入邮箱" />
      <div v-if="accountEditEmailChanged && !editAccountPassword" class="account-edit-pwd-box">
        <Input
          v-model="accountEditCurrentPassword"
          type="password"
          class="form-input"
          placeholder="修改登录邮箱须填写您的当前密码"
        />
      </div>
      <label class="nav-add__label">角色</label>
      <Select
        v-model="accountEditRole"
        class="form-input"
        :options="roleSelectOptions"
        :disabled="accountEditIsAdmin"
      />
      <label class="nav-add__label">权限</label>
      <div class="account-perm-list">
        <Checkbox v-model="accountEditManageLayout" :disabled="accountEditIsAdmin"
          >目录管理</Checkbox
        >
        <Checkbox v-model="accountEditManageAccounts" :disabled="accountEditIsAdmin"
          >用户管理</Checkbox
        >
      </div>
      <div class="account-password-head">
        <label class="nav-add__label">修改密码</label>
        <Button
          size="small"
          class="table-action-btn"
          type="info"
          native-type="button"
          @click="toggleAccountEditPassword"
        >
          {{ editAccountPassword ? '取消' : '修改' }}
        </Button>
      </div>
      <div v-if="editAccountPassword" class="account-edit-pwd-box">
        <Input
          v-model="accountEditCurrentPassword"
          type="password"
          class="form-input"
          placeholder="请输入当前密码"
        />
        <Input
          v-model="accountEditNewPassword"
          type="password"
          class="form-input"
          placeholder="新密码至少 6 位"
        />
        <Input
          v-model="accountEditConfirmPassword"
          type="password"
          class="form-input"
          placeholder="再次输入新密码"
        />
      </div>
      <p v-if="accountEditError" class="inline-dialog-error">{{ accountEditError }}</p>
    </div>
  </Dialog>

  <Dialog
    :open="accountAddVisible"
    title="添加用户"
    cancel-text="取消"
    :body-padded="true"
    :confirm-text="addingAccount ? '创建中...' : '创建'"
    :confirm-disabled="addingAccount"
    :z-index="2700"
    width="460px"
    height="560px"
    @cancel="closeAccountAddDialog"
    @confirm="saveAccountAddDialog"
    @close="closeAccountAddDialog"
  >
    <div class="inline-dialog">
      <label class="nav-add__label">用户名</label>
      <Input v-model="accountAddDisplayName" class="form-input" placeholder="请输入用户名" />
      <label class="nav-add__label">邮箱</label>
      <Input v-model="accountAddEmail" type="email" class="form-input" placeholder="请输入邮箱" />
      <label class="nav-add__label">密码</label>
      <div class="account-edit-pwd-box">
        <Input
          v-model="accountAddPassword"
          type="password"
          class="form-input"
          placeholder="新密码至少 6 位"
        />
        <Input
          v-model="accountAddConfirmPassword"
          type="password"
          class="form-input"
          placeholder="再次输入新密码"
        />
      </div>
      <label class="nav-add__label">角色</label>
      <Select v-model="accountAddRole" class="form-input" :options="roleSelectOptions" />
      <label class="nav-add__label">权限</label>
      <div class="account-perm-list">
        <Checkbox v-model="accountAddManageLayout">目录管理</Checkbox>
        <Checkbox v-model="accountAddManageAccounts">用户管理</Checkbox>
      </div>
      <p v-if="accountAddError" class="inline-dialog-error">{{ accountAddError }}</p>
    </div>
  </Dialog>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { storeToRefs } from 'pinia'
import { useAuthStore } from '@/stores/auth'
import { useCategoryNavStore } from '@/stores/categoryNav'
import { useGalleryItemsSyncStore } from '@/stores/galleryItemsSync'
import { useNavAddModalStore } from '@/stores/navAddModal'
import { useMessageStore } from '@/stores/message'
import personIconSrc from '@/assets/icon/person.svg'
import lockIconSrc from '@/assets/icon/lock.svg'
import privacyIconSrc from '@/assets/icon/privacy.svg'
import userManagementIconSrc from '@/assets/icon/user-management.svg'
import storageIconSrc from '@/assets/icon/database.svg'
import recycleIconSrc from '@/assets/icon/delete.svg'
import leftIconSrc from '@/assets/icon/left.svg'
import editIconSrc from '@/assets/icon/edit.svg?url'
import {
  deleteAuthAccount,
  deleteCategoryMajor,
  deleteCategorySub,
  fetchAuthAccounts,
  fetchStorageStatus,
  patchAuthAccount,
  postAuthAccount,
  patchStorageQuota,
  recalculateStorage,
  patchCategoriesFolderKeys,
  patchCategoriesNames,
  patchCategoriesSubMajor,
  patchCategoriesVisibility,
  postCategorySub,
  type CategoryFolderKeyPatch,
  type CategoryNamePatch,
  type CategorySubMajorPatch,
  type CategoryVisibilityPatch,
} from '@/api/adminApi'
import {
  fetchPasskeyList,
  postPasskeyRegisterOptions,
  postPasskeyRegisterVerify,
  type PasskeyItem,
} from '@/api/authApi'
import type {
  ApiAccountPublic,
  ApiCategoriesDoc,
  ApiCategoryGroup,
  ApiSubcategory,
} from '@/api/types'
import {
  applyFolderAccessPolicy,
  FOLDER_ACCESS_POLICY_OPTIONS,
  folderAccessPolicyFrom,
  folderAccessPolicyLabel,
  type FolderAccessPolicy,
} from '@/utils/galleryAccess'
import Button from '@/components/shared-ui/Button.vue'
import Checkbox from '@/components/shared-ui/Checkbox.vue'
import Dialog from '@/components/shared-ui/Dialog.vue'
import Input from '@/components/shared-ui/Input.vue'
import Popconfirm from '@/components/shared-ui/Popconfirm.vue'
import Select from '@/components/shared-ui/Select.vue'
import Table from '@/components/shared-ui/Table.vue'
import StoragePieChart from '@/components/shared-ui/StoragePieChart.vue'
import NavAddModal from '@/components/modals/NavAddModal.vue'
import RecycleBinPanel from '@/components/settings/RecycleBinPanel.vue'
import { formatFileSize } from '@/utils/formatFileSize'
import { parseApiErrorMessage } from '@/utils/apiError'
import { isReservedGalleryFolderKey } from '@/utils/gallerySearchFolder'
import { clearGalleryViewGrant } from '@/utils/galleryViewGrant'
import type { ApiStorageStatus } from '@/api/types'

const auth = useAuthStore()
const categoryNav = useCategoryNavStore()
const galleryItemsSync = useGalleryItemsSyncStore()
const navAddModalStore = useNavAddModalStore()
const appMessage = useMessageStore()
const router = useRouter()
const { profileModalOpen, currentUser } = storeToRefs(auth)

type SettingsTab = 'me' | 'settings' | 'security' | 'accounts' | 'storage' | 'recycle'
type MobileView = 'menu' | 'content'

const settingsMobileMq =
  typeof window !== 'undefined' ? window.matchMedia('(max-width: 640px)') : null
const isMobileSettings = ref(settingsMobileMq?.matches ?? false)
const mobileView = ref<MobileView>('menu')

const tab = ref<SettingsTab>('me')
const localError = ref('')
const successMsg = ref('')
const savingProfile = ref(false)
const savingPassword = ref(false)
const savingAvatar = ref(false)
const savingAccountId = ref<string | null>(null)

const draftUsername = ref('')
const draftDisplayName = ref('')
const draftNewPassword = ref('')
const draftConfirmPassword = ref('')
const currentPasswordForPassword = ref('')
const emailPwdDialogVisible = ref(false)
const emailPwdInput = ref('')
const emailPwdDialogError = ref('')

const editPassword = ref(false)

const avatarInputRef = ref<HTMLInputElement | null>(null)
const avatarLocalPreview = ref('')

const navDraft = ref<ApiCategoriesDoc | null>(null)
const accountsRows = ref<ApiAccountPublic[] | null>(null)
const accountsLoadErr = ref('')
const accountsTouched = ref(false)
const privacyTablePage = ref(1)
const accountTablePage = ref(1)
const openPickerKey = ref<string | null>(null)
const encryptedPwdDialogVisible = ref(false)
const encryptedPwdDialogName = ref('')
const encryptedPwdInput = ref('')
const encryptedPwdError = ref('')
let encryptedPwdResolver: ((value: string | null) => void) | null = null
let successMsgTimer: ReturnType<typeof setTimeout> | null = null
const privacyEditVisible = ref(false)
const privacyEditTargetScope = ref<'major' | 'sub'>('major')
const privacyEditMajorId = ref<number | null>(null)
const privacyEditMajorIdStr = ref('')
const privacyEditSubId = ref<number | null>(null)
const privacyEditName = ref('')
const privacyEditFolderKey = ref('')
const privacyEditPolicy = ref<FolderAccessPolicy>('open')
const privacyEditEncryptedPassword = ref('')
const privacyEditHasStoredPassword = ref(false)
const privacyEditPasswordTouched = ref(false)
const privacyEditError = ref('')
const privacyEditing = ref(false)
const addGalleryVisible = ref(false)
const addGalleryMajorId = ref('')
const addGalleryName = ref('')
const addGalleryFolderKey = ref('')
const addGalleryPublic = ref(true)
const addGalleryError = ref('')
const addingGallery = ref(false)
const accountEditVisible = ref(false)
const accountEditId = ref('')
const accountEditOriginalEmail = ref('')
const accountEditDisplayName = ref('')
const accountEditEmail = ref('')
const accountEditRole = ref('member')
const accountEditManageLayout = ref(false)
const accountEditManageAccounts = ref(false)
const editAccountPassword = ref(false)
const accountEditCurrentPassword = ref('')
const accountEditNewPassword = ref('')
const accountEditConfirmPassword = ref('')
const accountEditIsAdmin = ref(false)
const accountEditError = ref('')
const accountAddVisible = ref(false)
const accountAddDisplayName = ref('')
const accountAddEmail = ref('')
const accountAddPassword = ref('')
const accountAddConfirmPassword = ref('')
const accountAddRole = ref('member')
const accountAddManageLayout = ref(false)
const accountAddManageAccounts = ref(false)
const accountAddError = ref('')
const addingAccount = ref(false)
const passkeys = ref<PasskeyItem[]>([])
const passkeysLoading = ref(false)
const bindingPasskey = ref(false)
const storageStatus = ref<ApiStorageStatus | null>(null)
const storageLoadErr = ref('')
const storageQuotaGbDraft = ref('30')
const savingStorageQuota = ref(false)
const recalculatingStorage = ref(false)
const storageTouched = ref(false)
const recycleBinRef = ref<InstanceType<typeof RecycleBinPanel> | null>(null)
const recycleTouched = ref(false)

const user = computed(() => currentUser.value)
const canManageLayout = computed(() => !!(user.value?.permissions ?? []).includes('manage_layout'))
const canManageAccounts = computed(
  () => !!(user.value?.permissions ?? []).includes('manage_accounts'),
)
const usernameChanged = computed(
  () => draftUsername.value.trim() !== (user.value?.email ?? user.value?.username ?? '').trim(),
)
const displayNameChanged = computed(
  () => draftDisplayName.value.trim() !== (user.value?.displayName ?? '').trim(),
)
const storageQuotaChanged = computed(() => {
  const status = storageStatus.value
  if (!status) return false
  const draft = Number.parseFloat(storageQuotaGbDraft.value.trim())
  const saved = Number.parseFloat(quotaGbFromBytes(status.quotaBytes))
  if (!Number.isFinite(draft) || !Number.isFinite(saved)) {
    return storageQuotaGbDraft.value.trim() !== quotaGbFromBytes(status.quotaBytes)
  }
  return draft !== saved
})
const hasCustomAvatar = computed(() => {
  const avatar = user.value?.avatar?.trim()
  return !!avatar && avatar.startsWith('/api/avatar/')
})
const accountEditEmailChanged = computed(
  () => accountEditEmail.value.trim() !== accountEditOriginalEmail.value.trim(),
)
const accountEditDialogHeight = computed(() => {
  if (editAccountPassword.value) return '620px'
  if (accountEditEmailChanged.value) return '560px'
  return '500px'
})

const userName = computed(
  () => user.value?.displayName?.trim() || user.value?.email || user.value?.username || '未登录',
)
const userSubText = computed(() => {
  const email = (user.value?.email ?? user.value?.username ?? '').trim()
  if (email) return email
  return `ID ${user.value?.id ?? '—'}`
})
const fallbackAvatar = computed(() => auth.resolvedAvatarUrl())
const avatarPreview = computed(() => {
  if (avatarLocalPreview.value) return avatarLocalPreview.value
  return auth.resolvedAvatarUrl()
})

const mainTitle = computed(() => {
  if (tab.value === 'me') return '个人资料'
  if (tab.value === 'settings') return '隐私设置'
  if (tab.value === 'security') return '安全'
  if (tab.value === 'storage') return '存储管理'
  if (tab.value === 'recycle') return '回收站'
  return '用户管理'
})

const sortedMajorsForNav = computed(() => {
  const d = navDraft.value
  if (!d) return []
  return [...(d.categories ?? [])].sort((a, b) => {
    const sa = a.sort ?? Number.POSITIVE_INFINITY
    const sb = b.sort ?? Number.POSITIVE_INFINITY
    if (sa !== sb) return sa - sb
    return a.id - b.id
  })
})

type PrivacyTableRow = {
  id: string
  scope: 'major' | 'sub'
  name: string
  folderKey: string
  policyLabel: string
  majorRef: ApiCategoryGroup
  subRef?: ApiSubcategory
  children?: PrivacyTableRow[]
}

const privacyTableColumns = [
  { key: '__control', title: '', width: 70, type: 'control' as const, align: 'left' as const },
  {
    key: '__index',
    title: '#',
    width: 45,
    type: 'index' as const,
    align: 'left' as const,
  },
  { key: 'name', title: '目录名称', width: 100, ellipsis: true },
  { key: 'type', title: '类型', width: 80, align: 'left' as const },
  { key: 'folderKey', title: '目录键', width: 120, ellipsis: true },
  { key: 'policyLabel', title: '访问策略', width: 120, ellipsis: true },
  { key: 'actions', title: '操作', width: 120, fixed: 'right' as const, align: 'center' as const },
]

const deletingCategory = ref(false)

const privacyTableRows = computed<PrivacyTableRow[]>(() =>
  sortedMajorsForNav.value.map((major) => ({
    id: `major-${major.id}`,
    scope: 'major',
    name: major.name,
    folderKey: '',
    policyLabel: policyLabel(folderAccessPolicyFrom(major)),
    majorRef: major,
    children: sortedSubs(major).map((sub) => ({
      id: `sub-${major.id}-${sub.id}`,
      scope: 'sub',
      name: sub.name,
      folderKey: sub.folderKey,
      policyLabel: policyLabel(folderAccessPolicyFrom(sub)),
      majorRef: major,
      subRef: sub,
    })),
  })),
)

const accountTableColumns = [
  { key: '__control', title: '', width: 40, type: 'control' as const, align: 'left' as const },
  {
    key: '__index',
    title: '#',
    width: 50,
    type: 'index' as const,
    align: 'center' as const,
  },
  { key: 'displayName', title: '用户名', width: 100, ellipsis: true },
  { key: 'email', title: '邮箱', width: 120, ellipsis: true },
  { key: 'role', title: '角色', width: 80, align: 'center' as const },
  { key: 'manageLayout', title: '目录管理', width: 80, align: 'center' as const },
  { key: 'manageAccounts', title: '用户管理', width: 80, align: 'center' as const },
  { key: 'actions', title: '操作', width: 120, fixed: 'right' as const, align: 'center' as const },
]

const accountTableRows = computed(() =>
  (accountsRows.value ?? []).map((row) => ({
    id: row.id,
    displayName: row.displayName ?? '-',
    email: row.email ?? '-',
    role: roleLabelText(currentRoleValue(row)),
    manageLayout: hasPerm(row, 'manage_layout') ? '是' : '否',
    manageAccounts: hasPerm(row, 'manage_accounts') ? '是' : '否',
    raw: row,
  })),
)

function sortedSubs(major: ApiCategoryGroup): ApiSubcategory[] {
  return [...(major.subcategories ?? [])].sort((a, b) => {
    const sa = a.sort ?? Number.POSITIVE_INFINITY
    const sb = b.sort ?? Number.POSITIVE_INFINITY
    if (sa !== sb) return sa - sb
    return a.id - b.id
  })
}

function onPrivacyRowEdit(row: PrivacyTableRow) {
  if (row.scope === 'major') {
    openPrivacyEditMajor(row.majorRef)
    return
  }
  if (row.subRef) openPrivacyEditSub(row.majorRef, row.subRef)
}

function categoryDeleteSuccessMessage(label: string, trashedItems: number): string {
  if (trashedItems > 0) {
    return `${label}已删除，${trashedItems} 个文件已移入回收站，可在回收站恢复或一键重建目录`
  }
  return `${label}已删除`
}

function folderKeysFromMajor(major: ApiCategoryGroup): string[] {
  return (major.subcategories ?? []).map((sub) => sub.folderKey.trim()).filter(Boolean)
}

async function refreshAfterCategoryDelete(deletedFolderKeys: string[], trashedItems: number) {
  for (const fk of deletedFolderKeys) {
    galleryItemsSync.markCategoryItemsChanged(fk)
  }
  await categoryNav.reloadFromServer()
  if (canManageLayout.value) initNavDraft()

  const currentFolderKey = String(router.currentRoute.value.params.folderKey ?? '').trim()
  if (currentFolderKey && deletedFolderKeys.includes(currentFolderKey)) {
    await router.replace({ name: 'home' })
  }

  if (trashedItems > 0 && canManageAccounts.value) {
    recycleTouched.value = true
    await nextTick()
    await recycleBinRef.value?.reload()
  }
}

async function onPrivacyRowDelete(row: PrivacyTableRow) {
  clearMessages()
  deletingCategory.value = true
  try {
    if (row.scope === 'major') {
      const deletedFolderKeys = folderKeysFromMajor(row.majorRef)
      const out = await deleteCategoryMajor(row.majorRef.id)
      successMsg.value = categoryDeleteSuccessMessage('导航', out.trashedItems)
      await refreshAfterCategoryDelete(deletedFolderKeys, out.trashedItems)
      return
    }
    if (row.subRef) {
      const deletedFolderKeys = [row.subRef.folderKey.trim()].filter(Boolean)
      const out = await deleteCategorySub(row.majorRef.id, row.subRef.id)
      successMsg.value = categoryDeleteSuccessMessage('画廊', out.trashedItems)
      await refreshAfterCategoryDelete(deletedFolderKeys, out.trashedItems)
    }
  } catch (e: unknown) {
    localError.value = e instanceof Error ? e.message : '删除失败'
  } finally {
    deletingCategory.value = false
  }
}

function asPrivacyTableRow(row: Record<string, unknown>): PrivacyTableRow {
  return row as PrivacyTableRow
}

function asAccountRaw(row: Record<string, unknown>): ApiAccountPublic {
  return (row as { raw: ApiAccountPublic }).raw
}

function syncSettingsMobileMq() {
  isMobileSettings.value = settingsMobileMq?.matches ?? false
}

function onSettingsMobileMqChange() {
  syncSettingsMobileMq()
  if (isMobileSettings.value && profileModalOpen.value) {
    mobileView.value = 'content'
  }
}

function onMenuSelect(nextTab: SettingsTab) {
  tab.value = nextTab
  if (isMobileSettings.value) {
    mobileView.value = 'content'
  }
}

function goBackToMenu() {
  mobileView.value = 'menu'
}

function close() {
  resolveEncryptedPwdDialog(null)
  openPickerKey.value = null
  auth.closeProfileModal()
}

function clearMessages() {
  localError.value = ''
  successMsg.value = ''
}

function clearSuccessMsgTimer() {
  if (successMsgTimer) {
    clearTimeout(successMsgTimer)
    successMsgTimer = null
  }
}

function pickerKeyOf(scope: 'major' | 'sub' | 'role', id: string | number): string {
  return `${scope}-${String(id)}`
}

function isPickerOpen(key: string): boolean {
  return openPickerKey.value === key
}

function togglePicker(key: string) {
  openPickerKey.value = openPickerKey.value === key ? null : key
}

function closePicker() {
  openPickerKey.value = null
}

function resetEditors() {
  editPassword.value = false
  emailPwdDialogVisible.value = false
  emailPwdInput.value = ''
  emailPwdDialogError.value = ''
  avatarLocalPreview.value = ''
  if (avatarInputRef.value) avatarInputRef.value.value = ''
}

function initNavDraft() {
  if (!categoryNav.doc) return
  navDraft.value = JSON.parse(JSON.stringify(categoryNav.doc)) as ApiCategoriesDoc
}

function folderKeysForVisibilityPatch(doc: ApiCategoriesDoc, patch: CategoryVisibilityPatch): string[] {
  if (patch.scope === 'sub' && patch.subId != null) {
    const major = doc.categories.find((item) => item.id === patch.majorId)
    const sub = major?.subcategories.find((item) => item.id === patch.subId)
    const fk = sub?.folderKey?.trim()
    return fk ? [fk] : []
  }
  if (patch.scope === 'major') {
    const major = doc.categories.find((item) => item.id === patch.majorId)
    return (major?.subcategories ?? [])
      .map((sub) => sub.folderKey?.trim())
      .filter((fk): fk is string => !!fk)
  }
  return []
}

function notifyFolderViewCredentialChanged(doc: ApiCategoriesDoc, patch: CategoryVisibilityPatch) {
  if (!patch.encryptedPassword?.trim()) return
  for (const fk of folderKeysForVisibilityPatch(doc, patch)) {
    clearGalleryViewGrant(fk)
    galleryItemsSync.markCategoryItemsChanged(fk)
  }
}

async function persistNavPatch(patch: CategoryVisibilityPatch) {
  clearMessages()
  if (!navDraft.value || !categoryNav.doc) {
    localError.value = '无分类数据'
    return
  }
  try {
    const out = await patchCategoriesVisibility([patch])
    categoryNav.replaceDoc(out)
    navDraft.value = JSON.parse(JSON.stringify(out)) as ApiCategoriesDoc
    notifyFolderViewCredentialChanged(out, patch)
    successMsg.value = patch.encryptedPassword?.trim()
      ? '查看密码已更新'
      : '目录策略已保存'
  } catch (e: unknown) {
    localError.value = e instanceof Error ? e.message : '保存失败'
    initNavDraft()
  }
}

async function persistNamePatch(patch: CategoryNamePatch) {
  clearMessages()
  if (!navDraft.value || !categoryNav.doc) {
    localError.value = '无分类数据'
    return
  }
  try {
    const out = await patchCategoriesNames([patch])
    categoryNav.replaceDoc(out)
    navDraft.value = JSON.parse(JSON.stringify(out)) as ApiCategoriesDoc
    successMsg.value = '目录名称已保存'
  } catch (e: unknown) {
    localError.value = e instanceof Error ? e.message : '保存失败'
    initNavDraft()
  }
}

async function persistFolderKeyPatch(patch: CategoryFolderKeyPatch) {
  clearMessages()
  if (!navDraft.value || !categoryNav.doc) {
    localError.value = '无分类数据'
    return
  }
  try {
    const out = await patchCategoriesFolderKeys([patch])
    categoryNav.replaceDoc(out)
    navDraft.value = JSON.parse(JSON.stringify(out)) as ApiCategoriesDoc
    successMsg.value = '目录键已保存'
  } catch (e: unknown) {
    localError.value = e instanceof Error ? e.message : '保存失败'
    initNavDraft()
  }
}

async function persistSubMajorPatch(patch: CategorySubMajorPatch) {
  clearMessages()
  if (!navDraft.value || !categoryNav.doc) {
    localError.value = '无分类数据'
    return
  }
  try {
    const out = await patchCategoriesSubMajor([patch])
    categoryNav.replaceDoc(out)
    navDraft.value = JSON.parse(JSON.stringify(out)) as ApiCategoriesDoc
    successMsg.value = '所属导航已更新'
  } catch (e: unknown) {
    localError.value = e instanceof Error ? e.message : '保存失败'
    initNavDraft()
  }
}

function openEncryptedPwdDialog(name: string): Promise<string | null> {
  encryptedPwdDialogVisible.value = true
  encryptedPwdDialogName.value = name
  encryptedPwdInput.value = ''
  encryptedPwdError.value = ''
  return new Promise((resolve) => {
    encryptedPwdResolver = resolve
  })
}

function resolveEncryptedPwdDialog(value: string | null) {
  if (encryptedPwdResolver) {
    encryptedPwdResolver(value)
    encryptedPwdResolver = null
  }
  encryptedPwdDialogVisible.value = false
  encryptedPwdDialogName.value = ''
  encryptedPwdInput.value = ''
  encryptedPwdError.value = ''
}

function cancelEncryptedPwdDialog() {
  resolveEncryptedPwdDialog(null)
}

function confirmEncryptedPwdDialog() {
  const next = encryptedPwdInput.value.trim()
  if (next.length < 4) {
    encryptedPwdError.value = '查看密码至少 4 位'
    return
  }
  resolveEncryptedPwdDialog(next)
}

function hasPerm(row: ApiAccountPublic, p: string): boolean {
  return !!(row.permissions ?? []).includes(p)
}

function isAdminAccount(row: ApiAccountPublic): boolean {
  if (row.id === '0') return true
  return (row.roles ?? []).some((role) => role.trim().toLowerCase() === 'admin')
}

const roleOptions = computed(() => {
  const set = new Set<string>(['admin', 'member'])
  for (const row of accountsRows.value ?? []) {
    for (const role of row.roles ?? []) {
      const text = role.trim()
      if (text) set.add(text)
    }
  }
  return [...set]
})

const roleLabelMap: Record<string, string> = {
  admin: '管理员',
  member: '成员',
}

function roleLabelText(role: string): string {
  return roleLabelMap[role] ?? role
}

function currentRoleValue(row: ApiAccountPublic): string {
  return row.roles?.[0]?.trim() || 'member'
}

function onRoleChange(row: ApiAccountPublic, role: string) {
  if (isAdminAccount(row)) return
  row.roles = [role.trim() || 'member']
}

function pickRole(row: ApiAccountPublic, role: string) {
  onRoleChange(row, role)
  closePicker()
}

const accessPolicySelectOptions = computed(() =>
  FOLDER_ACCESS_POLICY_OPTIONS.map((item) => ({ label: item.label, value: item.value })),
)
const addGalleryMajorOptions = computed(() =>
  sortedMajorsForNav.value.map((major) => ({
    label: major.name,
    value: String(major.id),
  })),
)
const privacyEditIsEncryptedPolicy = computed(
  () =>
    privacyEditPolicy.value === 'encrypted_public' ||
    privacyEditPolicy.value === 'encrypted_hidden',
)

const STORED_VIEW_PASSWORD_MASK = '********'

function privacyEditTargetHasStoredPassword(): boolean {
  const majorId = privacyEditMajorId.value
  if (majorId == null) return false
  const major = sortedMajorsForNav.value.find((item) => item.id === majorId)
  if (!major) return false
  if (privacyEditTargetScope.value === 'major') {
    return !!major.encryptedPasswordHash?.trim()
  }
  const subId = privacyEditSubId.value
  const sub = major.subcategories.find((item) => item.id === subId)
  return !!sub?.encryptedPasswordHash?.trim()
}

function resetPrivacyEditPassword(hasStoredPassword: boolean) {
  privacyEditHasStoredPassword.value = hasStoredPassword
  privacyEditPasswordTouched.value = false
  privacyEditEncryptedPassword.value = hasStoredPassword ? STORED_VIEW_PASSWORD_MASK : ''
}

function resolvePrivacyEditPasswordInput(): string {
  const raw = privacyEditEncryptedPassword.value.trim()
  if (!privacyEditPasswordTouched.value && privacyEditHasStoredPassword.value) return ''
  if (raw === STORED_VIEW_PASSWORD_MASK) return ''
  return raw
}

function onPrivacyEditPasswordFocus() {
  if (!privacyEditPasswordTouched.value && privacyEditHasStoredPassword.value) {
    privacyEditEncryptedPassword.value = ''
    privacyEditPasswordTouched.value = true
  }
}

function onPrivacyEditPasswordModelUpdate(value: string) {
  privacyEditPasswordTouched.value = true
  privacyEditEncryptedPassword.value = value
}

const privacyEditPasswordNeedsInput = computed(() => {
  if (!privacyEditIsEncryptedPolicy.value) return false
  if (privacyEditHasStoredPassword.value && !privacyEditPasswordTouched.value) return false
  return !privacyEditEncryptedPassword.value.trim()
})

const privacyEditPasswordHint = computed(() => {
  if (!privacyEditIsEncryptedPolicy.value) return ''
  if (privacyEditHasStoredPassword.value && !privacyEditPasswordTouched.value) {
    return '已设置查看密码，输入新密码可修改'
  }
  if (privacyEditPasswordNeedsInput.value) {
    return '请设置查看密码（至少 4 位）'
  }
  return ''
})

watch(privacyEditPolicy, (policy, prev) => {
  if (!privacyEditVisible.value) return
  const nowEncrypted = policy === 'encrypted_public' || policy === 'encrypted_hidden'
  const wasEncrypted = prev === 'encrypted_public' || prev === 'encrypted_hidden'
  if (nowEncrypted && !wasEncrypted) {
    resetPrivacyEditPassword(privacyEditTargetHasStoredPassword())
  } else if (!nowEncrypted) {
    resetPrivacyEditPassword(false)
  }
})
const roleSelectOptions = computed(() =>
  roleOptions.value.map((role) => ({ label: roleLabelText(role), value: role })),
)

function policyLabel(value: FolderAccessPolicy): string {
  return folderAccessPolicyLabel(value)
}

function togglePerm(row: ApiAccountPublic, p: string, on: boolean) {
  if (isAdminAccount(row)) return
  const set = new Set(row.permissions ?? [])
  if (on) set.add(p)
  else set.delete(p)
  row.permissions = [...set]
}

async function onMajorPolicyChange(
  major: ApiCategoryGroup,
  policy: FolderAccessPolicy,
  encryptedPassword?: string,
) {
  const before = folderAccessPolicyFrom(major)
  const nextPwd = encryptedPassword?.trim() ?? ''
  if (before === policy && !nextPwd) return
  const patch: CategoryVisibilityPatch = {
    scope: 'major',
    majorId: major.id,
    public: policy === 'open' || policy === 'encrypted_public',
    encrypted: policy === 'encrypted_public' || policy === 'encrypted_hidden',
  }
  if (patch.encrypted) {
    if (nextPwd) {
      patch.encryptedPassword = nextPwd
    } else if (!major.encryptedPasswordHash) {
      const pwd = await openEncryptedPwdDialog(major.name)
      if (!pwd) return
      patch.encryptedPassword = pwd
    }
  }
  if (before !== policy) {
    applyFolderAccessPolicy(major, policy)
  }
  await persistNavPatch(patch)
}

async function pickMajorPolicy(major: ApiCategoryGroup, policy: FolderAccessPolicy) {
  closePicker()
  await onMajorPolicyChange(major, policy)
}

async function onSubPolicyChange(
  sub: ApiSubcategory,
  policy: FolderAccessPolicy,
  encryptedPassword?: string,
) {
  const before = folderAccessPolicyFrom(sub)
  const nextPwd = encryptedPassword?.trim() ?? ''
  if (before === policy && !nextPwd) return
  const major = sortedMajorsForNav.value.find((it) => it.subcategories.some((x) => x.id === sub.id))
  if (!major) {
    localError.value = '无法定位目录所属分类'
    return
  }
  const patch: CategoryVisibilityPatch = {
    scope: 'sub',
    majorId: major.id,
    subId: sub.id,
    public: policy === 'open' || policy === 'encrypted_public',
    encrypted: policy === 'encrypted_public' || policy === 'encrypted_hidden',
  }
  if (patch.encrypted) {
    if (nextPwd) {
      patch.encryptedPassword = nextPwd
    } else if (!sub.encryptedPasswordHash) {
      const pwd = await openEncryptedPwdDialog(sub.name)
      if (!pwd) return
      patch.encryptedPassword = pwd
    }
  }
  if (before !== policy) {
    applyFolderAccessPolicy(sub, policy)
  }
  await persistNavPatch(patch)
}

async function pickSubPolicy(sub: ApiSubcategory, policy: FolderAccessPolicy) {
  closePicker()
  await onSubPolicyChange(sub, policy)
}

async function onSaveMajorName(major: ApiCategoryGroup) {
  const next = major.name.trim()
  if (!next) {
    localError.value = '目录名称不能为空'
    return
  }
  await persistNamePatch({
    scope: 'major',
    majorId: major.id,
    name: next,
  })
}

async function onSaveSubName(major: ApiCategoryGroup, sub: ApiSubcategory) {
  const next = sub.name.trim()
  if (!next) {
    localError.value = '目录名称不能为空'
    return
  }
  await persistNamePatch({
    scope: 'sub',
    majorId: major.id,
    subId: sub.id,
    name: next,
  })
}

async function onSaveSubFolderKey(major: ApiCategoryGroup, sub: ApiSubcategory) {
  const next = sub.folderKey.trim()
  if (!next) {
    localError.value = '目录键不能为空'
    return
  }
  if (!/^[A-Za-z_]+$/.test(next)) {
    localError.value = '目录键仅支持英文和下划线'
    return
  }
  if (isReservedGalleryFolderKey(next)) {
    localError.value = '目录键 search 为系统保留，不可使用'
    return
  }
  await persistFolderKeyPatch({
    majorId: major.id,
    subId: sub.id,
    folderKey: next,
  })
}

async function saveDisplayNameIfNeeded() {
  const next = draftDisplayName.value.trim()
  const prev = (user.value?.displayName ?? '').trim()
  if (next === prev) {
    return
  }
  if (!next) {
    localError.value = '用户名不能为空'
    return
  }
  clearMessages()
  savingProfile.value = true
  try {
    await auth.updateProfile({ displayName: next })
    draftDisplayName.value = auth.currentUser?.displayName ?? ''
    successMsg.value = '用户名已保存'
  } catch (e: unknown) {
    localError.value = e instanceof Error ? e.message : '保存失败'
  } finally {
    savingProfile.value = false
  }
}

function onEditEmailClick() {
  const un = draftUsername.value.trim()
  if (!un) {
    localError.value = '邮箱不能为空'
    return
  }
  if (!usernameChanged.value) {
    return
  }
  emailPwdDialogError.value = ''
  emailPwdInput.value = ''
  emailPwdDialogVisible.value = true
}

function cancelEmailPwdDialog() {
  emailPwdDialogVisible.value = false
  emailPwdInput.value = ''
  emailPwdDialogError.value = ''
}

async function confirmEmailPwdDialog() {
  const pwd = emailPwdInput.value.trim()
  if (!pwd) {
    emailPwdDialogError.value = '请填写当前密码'
    return
  }
  const un = draftUsername.value.trim()
  if (!un) {
    emailPwdDialogError.value = '邮箱不能为空'
    return
  }
  clearMessages()
  savingProfile.value = true
  try {
    await auth.updateProfile({
      email: un,
      currentPassword: pwd,
    })
    cancelEmailPwdDialog()
    await auth.logout()
    close()
  } catch (e: unknown) {
    emailPwdDialogError.value = e instanceof Error ? e.message : '保存失败'
  } finally {
    savingProfile.value = false
  }
}

async function onUpdatePassword() {
  clearMessages()
  if (draftNewPassword.value.trim().length < 6) {
    localError.value = '新密码至少 6 位'
    return
  }
  if (draftNewPassword.value.trim() !== draftConfirmPassword.value.trim()) {
    localError.value = '两次输入的新密码不一致'
    return
  }
  if (!currentPasswordForPassword.value.trim()) {
    localError.value = '请填写当前密码'
    return
  }
  savingPassword.value = true
  try {
    await auth.updateProfile({
      newPassword: draftNewPassword.value.trim(),
      currentPassword: currentPasswordForPassword.value.trim(),
    })
    draftNewPassword.value = ''
    draftConfirmPassword.value = ''
    currentPasswordForPassword.value = ''
    editPassword.value = false
    await auth.logout()
    close()
  } catch (e: unknown) {
    localError.value = e instanceof Error ? e.message : '更新失败'
  } finally {
    savingPassword.value = false
  }
}

function triggerAvatarPick() {
  avatarInputRef.value?.click()
}

const avatarMaxBytes = 10 * 1024 * 1024

async function onAvatarFileChange(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  clearMessages()
  if (!/^image\/(png|jpeg|webp)$/i.test(file.type)) {
    localError.value = '仅支持 PNG、JPG、WEBP 格式头像'
    input.value = ''
    return
  }
  if (file.size > avatarMaxBytes) {
    localError.value = '头像不能超过 10MB'
    input.value = ''
    return
  }
  const previewUrl = URL.createObjectURL(file)
  avatarLocalPreview.value = previewUrl
  savingAvatar.value = true
  try {
    await auth.uploadAvatar(file)
    avatarLocalPreview.value = ''
    successMsg.value = '头像已更新'
  } catch (err: unknown) {
    avatarLocalPreview.value = ''
    localError.value = err instanceof Error ? err.message : '头像上传失败'
  } finally {
    URL.revokeObjectURL(previewUrl)
    input.value = ''
    savingAvatar.value = false
  }
}

async function onRemoveAvatar() {
  clearMessages()
  savingAvatar.value = true
  try {
    await auth.removeAvatar()
    avatarLocalPreview.value = ''
    if (avatarInputRef.value) avatarInputRef.value.value = ''
    successMsg.value = '头像已移除'
  } catch (e: unknown) {
    localError.value = e instanceof Error ? e.message : '头像移除失败'
  } finally {
    savingAvatar.value = false
  }
}

function quotaGbFromBytes(bytes: number): string {
  const gb = bytes / (1024 * 1024 * 1024)
  if (!Number.isFinite(gb) || gb <= 0) return '30'
  const rounded = Math.round(gb * 100) / 100
  return String(rounded)
}

async function loadStorageStatus() {
  storageLoadErr.value = ''
  try {
    const st = await fetchStorageStatus()
    storageStatus.value = st
    storageQuotaGbDraft.value = quotaGbFromBytes(st.quotaBytes)
  } catch (e: unknown) {
    storageLoadErr.value = e instanceof Error ? e.message : '加载失败'
    storageStatus.value = null
  }
}

async function onSaveStorageQuota() {
  clearMessages()
  const gb = Number.parseFloat(storageQuotaGbDraft.value.trim())
  if (!Number.isFinite(gb) || gb <= 0) {
    localError.value = '请输入有效的配额（GB）'
    return
  }
  savingStorageQuota.value = true
  try {
    storageStatus.value = await patchStorageQuota(gb)
    storageQuotaGbDraft.value = quotaGbFromBytes(storageStatus.value.quotaBytes)
    successMsg.value = '存储配额已更新'
  } catch (e: unknown) {
    localError.value = e instanceof Error ? e.message : '保存失败'
  } finally {
    savingStorageQuota.value = false
  }
}

async function onRecalculateStorage() {
  clearMessages()
  recalculatingStorage.value = true
  try {
    storageStatus.value = await recalculateStorage()
    storageQuotaGbDraft.value = quotaGbFromBytes(storageStatus.value.quotaBytes)
    successMsg.value = '存储用量已重新统计'
  } catch (e: unknown) {
    localError.value = e instanceof Error ? e.message : '统计失败'
  } finally {
    recalculatingStorage.value = false
  }
}

async function loadAccountsList() {
  accountsLoadErr.value = ''
  try {
    const list = await fetchAuthAccounts()
    accountsRows.value = list.map((item) => ({
      ...item,
      displayName: item.displayName ?? '',
      email: item.email || item.username || '',
      roles: [...(item.roles ?? [])],
      permissions: [...(item.permissions ?? [])],
    }))
  } catch (e: unknown) {
    accountsLoadErr.value = e instanceof Error ? e.message : '加载失败'
    accountsRows.value = []
  }
}

async function loadPasskeyList() {
  passkeysLoading.value = true
  try {
    passkeys.value = await fetchPasskeyList()
  } catch (e: unknown) {
    localError.value = parseApiErrorMessage(e, '通行证加载失败')
    passkeys.value = []
  } finally {
    passkeysLoading.value = false
  }
}

function toBytes(b64url: string): Uint8Array {
  const pad = '='.repeat((4 - (b64url.length % 4)) % 4)
  const base64 = (b64url + pad).replace(/-/g, '+').replace(/_/g, '/')
  const raw = window.atob(base64)
  const out = new Uint8Array(raw.length)
  for (let i = 0; i < raw.length; i += 1) out[i] = raw.charCodeAt(i)
  return out
}

function fromBytes(buf: ArrayBuffer): string {
  const arr = new Uint8Array(buf)
  let raw = ''
  for (let i = 0; i < arr.length; i += 1) {
    const code = arr[i]
    if (code === undefined) continue
    raw += String.fromCharCode(code)
  }
  return window.btoa(raw).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/g, '')
}

function formatDateText(raw?: string): string {
  const text = raw?.trim() || ''
  if (!text) return '—'
  const d = new Date(text)
  if (Number.isNaN(d.getTime())) return text
  return d.toLocaleString()
}

async function onBindPasskey() {
  if (bindingPasskey.value) return
  if (!window.isSecureContext) {
    localError.value = '当前页面不是安全环境，请使用 HTTPS 域名访问后再绑定通行证'
    return
  }
  if (!window.PublicKeyCredential || !navigator.credentials?.create) {
    localError.value = '当前设备或浏览器不支持通行证，请在 iOS Safari 最新版重试'
    return
  }
  bindingPasskey.value = true
  clearMessages()
  try {
    const options = await postPasskeyRegisterOptions('iPhone 通行证')
    const publicKey: PublicKeyCredentialCreationOptions = {
      challenge: toBytes(options.challenge) as unknown as BufferSource,
      rp: options.rp,
      user: {
        id: toBytes(options.user.id) as unknown as BufferSource,
        name: options.user.name,
        displayName: options.user.displayName,
      },
      pubKeyCredParams: options.pubKeyCredParams,
      timeout: options.timeoutMs ?? 300000,
      attestation: options.attestation ?? 'none',
      authenticatorSelection: options.authenticatorSelection,
    }
    const created = (await navigator.credentials.create({
      publicKey,
    })) as PublicKeyCredential | null
    if (!created) throw new Error('通行证创建已取消')
    const response = created.response as AuthenticatorAttestationResponse
    const responseWithExtras = response as AuthenticatorAttestationResponse & {
      getPublicKey?: () => ArrayBuffer | null
      getPublicKeyAlgorithm?: () => number
      getAuthenticatorData?: () => ArrayBuffer
      getTransports?: () => string[]
    }
    const publicKeyBuffer = responseWithExtras.getPublicKey?.()
    const algorithm = responseWithExtras.getPublicKeyAlgorithm?.()
    const authenticatorDataBuffer = responseWithExtras.getAuthenticatorData?.()
    if (!publicKeyBuffer || typeof algorithm !== 'number' || !authenticatorDataBuffer) {
      throw new Error('当前浏览器返回的通行证信息不完整，请尝试 Safari 最新版本')
    }
    const payload = {
      credentialId: created.id,
      publicKey: fromBytes(publicKeyBuffer),
      algorithm,
      clientDataJSON: fromBytes(response.clientDataJSON),
      authenticatorData: fromBytes(authenticatorDataBuffer),
      transports: responseWithExtras.getTransports?.() ?? [],
      label: 'iPhone 通行证',
    }
    await postPasskeyRegisterVerify(payload)
    successMsg.value = '通行证绑定成功'
    await loadPasskeyList()
  } catch (e: unknown) {
    localError.value = parseApiErrorMessage(e, '通行证绑定失败')
  } finally {
    bindingPasskey.value = false
  }
}

async function onSaveAccount(row: ApiAccountPublic) {
  if (isAdminAccount(row)) return
  const nextDisplayName = row.displayName.trim()
  const nextEmail = row.email.trim()
  if (!nextDisplayName) {
    localError.value = '用户名不能为空'
    return
  }
  if (!nextEmail) {
    localError.value = '邮箱不能为空'
    return
  }
  clearMessages()
  savingAccountId.value = row.id
  try {
    await patchAuthAccount(row.id, {
      displayName: nextDisplayName,
      email: nextEmail,
      roles: [...(row.roles ?? [])],
      permissions: [...(row.permissions ?? [])],
    })
    successMsg.value = `已保存用户 ${row.email || row.username || row.id}`
    await loadAccountsList()
    if (row.id === user.value?.id) {
      await auth.fetchMe()
    }
  } catch (e: unknown) {
    localError.value = e instanceof Error ? e.message : '保存失败'
  } finally {
    savingAccountId.value = null
  }
}

function openPrivacyEditMajor(major: ApiCategoryGroup) {
  privacyEditVisible.value = true
  privacyEditing.value = false
  privacyEditTargetScope.value = 'major'
  privacyEditMajorId.value = major.id
  privacyEditSubId.value = null
  privacyEditName.value = major.name
  privacyEditPolicy.value = folderAccessPolicyFrom(major)
  resetPrivacyEditPassword(!!major.encryptedPasswordHash?.trim())
  privacyEditError.value = ''
}

function openPrivacyEditSub(major: ApiCategoryGroup, sub: ApiSubcategory) {
  privacyEditVisible.value = true
  privacyEditing.value = false
  privacyEditTargetScope.value = 'sub'
  privacyEditMajorId.value = major.id
  privacyEditMajorIdStr.value = String(major.id)
  privacyEditSubId.value = sub.id
  privacyEditName.value = sub.name
  privacyEditFolderKey.value = sub.folderKey
  privacyEditPolicy.value = folderAccessPolicyFrom(sub)
  resetPrivacyEditPassword(!!sub.encryptedPasswordHash?.trim())
  privacyEditError.value = ''
}

function closePrivacyEditDialog() {
  privacyEditVisible.value = false
  privacyEditing.value = false
  resetPrivacyEditPassword(false)
  privacyEditError.value = ''
}

async function savePrivacyEditDialog() {
  if (privacyEditing.value) return
  privacyEditError.value = ''
  privacyEditing.value = true
  try {
    const majorId = privacyEditMajorId.value
    if (majorId == null) return
    const nextName = privacyEditName.value.trim()
    if (!nextName) {
      privacyEditError.value = '名称不能为空'
      return
    }
    if (privacyEditTargetScope.value === 'major') {
      const major = sortedMajorsForNav.value.find((item) => item.id === majorId)
      if (!major) return
      if (major.name.trim() !== nextName) {
        await persistNamePatch({ scope: 'major', majorId, name: nextName })
      }
      const nextPwd = resolvePrivacyEditPasswordInput()
      if (nextPwd && nextPwd.length < 4) {
        privacyEditError.value = '查看密码至少 4 位'
        return
      }
      if (privacyEditIsEncryptedPolicy.value && !privacyEditTargetHasStoredPassword() && !nextPwd) {
        privacyEditError.value = '请填写查看密码（至少 4 位）'
        return
      }
      if (folderAccessPolicyFrom(major) !== privacyEditPolicy.value || !!nextPwd) {
        await onMajorPolicyChange(major, privacyEditPolicy.value, nextPwd)
      }
      closePrivacyEditDialog()
      return
    }

    const subId = privacyEditSubId.value
    if (subId == null) return
    const targetMajorId = Number.parseInt(privacyEditMajorIdStr.value, 10)
    if (!Number.isFinite(targetMajorId)) {
      privacyEditError.value = '请选择所属导航'
      return
    }
    let effectiveMajorId = majorId
    const major = sortedMajorsForNav.value.find((item) => item.id === majorId)
    const sub = major?.subcategories.find((item) => item.id === subId)
    if (!major || !sub) return
    if (targetMajorId !== majorId) {
      await persistSubMajorPatch({ majorId, subId, targetMajorId })
      effectiveMajorId = targetMajorId
    }
    if (sub.name.trim() !== nextName) {
      await persistNamePatch({ scope: 'sub', majorId: effectiveMajorId, subId, name: nextName })
    }

    const nextFolderKey = privacyEditFolderKey.value.trim()
    if (!nextFolderKey) {
      privacyEditError.value = '目录键不能为空'
      return
    }
    if (!/^[A-Za-z_]+$/.test(nextFolderKey)) {
      privacyEditError.value = '目录键仅支持英文和下划线'
      return
    }
    if (sub.folderKey.trim() !== nextFolderKey) {
      await persistFolderKeyPatch({ majorId: effectiveMajorId, subId, folderKey: nextFolderKey })
    }

    const nextPwd = resolvePrivacyEditPasswordInput()
    if (nextPwd && nextPwd.length < 4) {
      privacyEditError.value = '查看密码至少 4 位'
      return
    }
    const subAfterMove =
      sortedMajorsForNav.value
        .find((item) => item.id === effectiveMajorId)
        ?.subcategories.find((item) => item.id === subId) ?? sub
    if (privacyEditIsEncryptedPolicy.value && !subAfterMove.encryptedPasswordHash?.trim() && !nextPwd) {
      privacyEditError.value = '请填写查看密码（至少 4 位）'
      return
    }
    if (folderAccessPolicyFrom(subAfterMove) !== privacyEditPolicy.value || !!nextPwd) {
      await onSubPolicyChange(subAfterMove, privacyEditPolicy.value, nextPwd)
    }
    closePrivacyEditDialog()
  } finally {
    privacyEditing.value = false
  }
}

function openAddPrivacyDialog() {
  navAddModalStore.openPrimary()
}

function openAddGalleryDialog() {
  addGalleryVisible.value = true
  addGalleryMajorId.value = addGalleryMajorOptions.value[0]?.value ?? ''
  addGalleryName.value = ''
  addGalleryFolderKey.value = ''
  addGalleryPublic.value = true
  addGalleryError.value = ''
}

function closeAddGalleryDialog() {
  addGalleryVisible.value = false
}

const galleryFolderKeyPattern = /^[a-z0-9][a-z0-9_]{1,62}$/

async function submitAddGalleryDialog() {
  const majorId = Number.parseInt(addGalleryMajorId.value, 10)
  const name = addGalleryName.value.trim()
  const folderKey = addGalleryFolderKey.value.trim().toLowerCase()
  if (!Number.isFinite(majorId)) {
    addGalleryError.value = '请选择所属导航'
    return
  }
  if (!name) {
    addGalleryError.value = '画廊名称不能为空'
    return
  }
  if (!galleryFolderKeyPattern.test(folderKey)) {
    addGalleryError.value = '目录键格式错误，请使用 2-63 位小写字母/数字/下划线'
    return
  }
  if (isReservedGalleryFolderKey(folderKey)) {
    addGalleryError.value = '目录键 search 为系统保留，请换一个名称'
    return
  }
  addingGallery.value = true
  addGalleryError.value = ''
  try {
    const out = await postCategorySub({
      majorId,
      name,
      folderKey,
      public: addGalleryPublic.value,
    })
    categoryNav.replaceDoc(out)
    initNavDraft()
    successMsg.value = '画廊已添加'
    closeAddGalleryDialog()
  } catch (e: unknown) {
    addGalleryError.value = e instanceof Error ? e.message : '添加失败'
  } finally {
    addingGallery.value = false
  }
}

function resetAccountEditPasswordFields() {
  editAccountPassword.value = false
  accountEditCurrentPassword.value = ''
  accountEditNewPassword.value = ''
  accountEditConfirmPassword.value = ''
}

function toggleAccountEditPassword() {
  editAccountPassword.value = !editAccountPassword.value
  if (!editAccountPassword.value) {
    accountEditNewPassword.value = ''
    accountEditConfirmPassword.value = ''
    if (!accountEditEmailChanged.value) {
      accountEditCurrentPassword.value = ''
    }
  }
}

function openAccountEditDialog(row: ApiAccountPublic) {
  accountEditVisible.value = true
  accountEditId.value = row.id
  accountEditOriginalEmail.value = (row.email ?? row.username ?? '').trim()
  accountEditDisplayName.value = row.displayName ?? ''
  accountEditEmail.value = row.email ?? row.username ?? ''
  accountEditRole.value = currentRoleValue(row)
  if (isAdminAccount(row)) {
    accountEditManageLayout.value = true
    accountEditManageAccounts.value = true
  } else {
    accountEditManageLayout.value = hasPerm(row, 'manage_layout')
    accountEditManageAccounts.value = hasPerm(row, 'manage_accounts')
  }
  accountEditIsAdmin.value = isAdminAccount(row)
  accountEditError.value = ''
  resetAccountEditPasswordFields()
}

function closeAccountEditDialog() {
  accountEditVisible.value = false
  accountEditError.value = ''
  resetAccountEditPasswordFields()
}

async function saveAccountEditDialog() {
  const id = accountEditId.value
  if (!id) return
  const displayName = accountEditDisplayName.value.trim()
  const email = accountEditEmail.value.trim()
  if (!displayName) {
    accountEditError.value = '用户名不能为空'
    return
  }
  if (!email) {
    accountEditError.value = '邮箱不能为空'
    return
  }
  const emailChanged = accountEditEmailChanged.value
  const wantsPasswordChange =
    editAccountPassword.value && accountEditNewPassword.value.trim() !== ''
  if (emailChanged || wantsPasswordChange) {
    if (!accountEditCurrentPassword.value.trim()) {
      accountEditError.value = '修改登录邮箱或密码需要填写当前密码'
      return
    }
  }
  if (wantsPasswordChange) {
    if (accountEditNewPassword.value.trim().length < 6) {
      accountEditError.value = '新密码至少 6 位'
      return
    }
    if (accountEditNewPassword.value.trim() !== accountEditConfirmPassword.value.trim()) {
      accountEditError.value = '两次输入的新密码不一致'
      return
    }
  }
  const permissions: string[] = []
  if (accountEditIsAdmin.value) {
    permissions.push('manage_layout', 'manage_accounts')
  } else {
    if (accountEditManageLayout.value) permissions.push('manage_layout')
    if (accountEditManageAccounts.value) permissions.push('manage_accounts')
  }
  const roles = accountEditIsAdmin.value ? ['admin'] : [accountEditRole.value || 'member']
  const body: {
    displayName: string
    email: string
    roles: string[]
    permissions: string[]
    currentPassword?: string
    newPassword?: string
  } = { displayName, email, roles, permissions }
  if (emailChanged || wantsPasswordChange) {
    body.currentPassword = accountEditCurrentPassword.value.trim()
  }
  if (wantsPasswordChange) {
    body.newPassword = accountEditNewPassword.value.trim()
  }
  clearMessages()
  savingAccountId.value = id
  try {
    await patchAuthAccount(id, body)
    const editingSelf = id === user.value?.id
    const credentialsChanged = emailChanged || wantsPasswordChange
    closeAccountEditDialog()
    if (editingSelf && credentialsChanged) {
      await auth.logout()
      close()
      return
    }
    successMsg.value = credentialsChanged ? '用户信息已保存，该用户需重新登录' : '用户信息已保存'
    await loadAccountsList()
  } catch (e: unknown) {
    accountEditError.value = e instanceof Error ? e.message : '保存失败'
  } finally {
    savingAccountId.value = null
  }
}

function openAccountAddDialog() {
  accountAddVisible.value = true
  accountAddDisplayName.value = ''
  accountAddEmail.value = ''
  accountAddPassword.value = ''
  accountAddConfirmPassword.value = ''
  accountAddRole.value = 'member'
  accountAddManageLayout.value = false
  accountAddManageAccounts.value = false
  accountAddError.value = ''
}

function closeAccountAddDialog() {
  accountAddVisible.value = false
  accountAddError.value = ''
}

async function saveAccountAddDialog() {
  const displayName = accountAddDisplayName.value.trim()
  const email = accountAddEmail.value.trim()
  const password = accountAddPassword.value.trim()
  const confirmPassword = accountAddConfirmPassword.value.trim()
  if (!displayName) {
    accountAddError.value = '用户名不能为空'
    return
  }
  if (!email) {
    accountAddError.value = '邮箱不能为空'
    return
  }
  if (password.length < 6) {
    accountAddError.value = '新密码至少 6 位'
    return
  }
  if (password !== confirmPassword) {
    accountAddError.value = '两次输入的新密码不一致'
    return
  }
  const permissions: string[] = []
  if (accountAddManageLayout.value) permissions.push('manage_layout')
  if (accountAddManageAccounts.value) permissions.push('manage_accounts')
  const roles = [accountAddRole.value || 'member']
  clearMessages()
  addingAccount.value = true
  try {
    await postAuthAccount({ displayName, email, password, roles, permissions })
    successMsg.value = '用户已创建'
    closeAccountAddDialog()
    await loadAccountsList()
  } catch (e: unknown) {
    accountAddError.value = e instanceof Error ? e.message : '创建失败'
  } finally {
    addingAccount.value = false
  }
}

async function onDeleteAccount(row: ApiAccountPublic) {
  if (row.id === user.value?.id) {
    localError.value = '不能删除当前登录账号'
    return
  }
  clearMessages()
  try {
    await deleteAuthAccount(row.id)
    successMsg.value = '用户已删除'
    await loadAccountsList()
  } catch (e: unknown) {
    localError.value = e instanceof Error ? e.message : '删除失败'
  }
}

function onClickAddAccount() {
  openAccountAddDialog()
}

async function onLogout() {
  await auth.logout()
  close()
}

function onDocPointerdown(e: PointerEvent) {
  const t = e.target as HTMLElement | null
  if (!t?.closest('.settings-picker')) {
    closePicker()
  }
}

watch(profileModalOpen, async (visible) => {
  if (!visible) {
    mobileView.value = 'menu'
    navDraft.value = null
    privacyTablePage.value = 1
    accountTablePage.value = 1
    addGalleryVisible.value = false
    accountAddVisible.value = false
    closeAccountEditDialog()
    closePicker()
    resolveEncryptedPwdDialog(null)
    accountsRows.value = null
    passkeys.value = []
    accountsTouched.value = false
    recycleTouched.value = false
    resetEditors()
    return
  }
  clearMessages()
  resetEditors()
  mobileView.value = 'menu'
  tab.value = 'me'
  draftNewPassword.value = ''
  draftConfirmPassword.value = ''
  currentPasswordForPassword.value = ''
  try {
    await categoryNav.fetchFromServer()
    await auth.fetchMe()
    draftUsername.value = auth.currentUser?.email ?? auth.currentUser?.username ?? ''
    draftDisplayName.value = auth.currentUser?.displayName ?? ''
    if (canManageLayout.value) initNavDraft()
    const pendingTab = auth.takePendingProfileTab()
    if (pendingTab === 'nav' && canManageLayout.value) tab.value = 'settings'
    if (pendingTab === 'accounts' && canManageAccounts.value) tab.value = 'accounts'
    if (pendingTab === 'recycle' && canManageAccounts.value) tab.value = 'recycle'
    if (pendingTab && isMobileSettings.value) {
      mobileView.value = 'content'
    }
  } catch (e: unknown) {
    localError.value = e instanceof Error ? e.message : '加载失败'
  }
})

watch(tab, async (nextTab) => {
  if (!profileModalOpen.value) return
  closePicker()
  clearMessages()
  if (nextTab === 'settings') privacyTablePage.value = 1
  if (nextTab === 'security') await loadPasskeyList()
  if (nextTab === 'accounts') accountTablePage.value = 1
  if (nextTab === 'accounts' && canManageAccounts.value && !accountsTouched.value) {
    accountsTouched.value = true
    await loadAccountsList()
  }
  if (nextTab === 'storage' && canManageAccounts.value && !storageTouched.value) {
    storageTouched.value = true
    await loadStorageStatus()
  }
  if (nextTab === 'recycle' && canManageAccounts.value) {
    if (!recycleTouched.value) recycleTouched.value = true
    await nextTick()
    try {
      await recycleBinRef.value?.reload()
    } catch {
      /* RecycleBinPanel 自行展示 loadErr */
    }
  }
  if (nextTab === 'settings' && canManageLayout.value && !navDraft.value) {
    initNavDraft()
  }
})

watch(successMsg, (msg) => {
  clearSuccessMsgTimer()
  if (msg?.trim()) {
    appMessage.show(msg, {
      type: 'success',
      duration: 2400,
    })
  }
  if (!msg) return
  successMsgTimer = setTimeout(() => {
    successMsg.value = ''
    successMsgTimer = null
  }, 2600)
})

watch(localError, (msg) => {
  if (!msg?.trim()) return
  appMessage.show(msg, {
    type: 'error',
    duration: 2800,
  })
})

onMounted(() => {
  syncSettingsMobileMq()
  settingsMobileMq?.addEventListener('change', onSettingsMobileMqChange)
  document.addEventListener('pointerdown', onDocPointerdown)
})

onUnmounted(() => {
  clearSuccessMsgTimer()
  settingsMobileMq?.removeEventListener('change', onSettingsMobileMqChange)
  document.removeEventListener('pointerdown', onDocPointerdown)
})
</script>

<style scoped lang="scss">
.settings-modal {
  width: 100%;
  height: 100%;

  .content-box {
    width: 100%;
    height: 100%;
    background-color: #fff;
    position: relative;
    border-radius: 6px;
    overflow: hidden;
    display: flex;
    box-shadow: 0 28px 56px -24px rgba(0, 0, 0, 0.16);

    &.is-mobile {
      flex-direction: column;

      .sidebar,
      .content {
        width: 100%;
        flex: 1;
        min-height: 0;
      }
    }

    .sidebar {
      width: 220px;
      height: 100%;
      /* background-color: #fafafa; */
      display: flex;
      flex-direction: column;

      .avatar-box {
        display: flex;
        align-items: center;
        padding: 1rem;
        box-sizing: border-box;
        gap: 1rem;

        .avatar {
          width: 40px;
          height: 40px;
          border-radius: 50%;
          overflow: hidden;
          img {
            width: 100%;
            height: 100%;
            object-fit: cover;
          }
        }
        .info {
          font-size: 12px;
          line-height: 20px;

          .text {
            font-weight: bold;
            font-size: 14px;
          }
        }
      }
      .menu-box {
        width: 100%;
        height: 100%;
        margin-top: 1rem;
        .menu-item {
          display: flex;
          align-items: center;
          gap: 8px;
          padding: 6px 24px;
          font-size: 14px;
          color: #666;
          cursor: pointer;
          margin-bottom: 6px;

          &.active,
          &:hover {
            color: #333;
            background-color: #efefee;
          }

          &__icon {
            width: 16px;
            height: 16px;
            flex: 0 0 16px;
            display: block;
            object-fit: contain;
          }

          &__text {
            min-width: 0;
          }
        }
      }

      .footer-box {
        width: 100%;
        padding: 1rem;
        box-sizing: border-box;

        .footer-item {
          font-size: 12px;
          border-radius: 6px;
          border: 1px solid #e5e5e5;
          padding: 6px 1rem;
          cursor: pointer;
          user-select: none;

          &:active {
            background-color: #efefee;
          }
        }
      }
    }

    .content {
      padding: 1rem;
      box-sizing: border-box;
      flex: 1;
      min-width: 0;
      display: flex;
      flex-direction: column;

      .title {
        position: relative;
        display: flex;
        align-items: center;
        gap: 4px;
        font-size: 16px;
        color: #333;
        padding-bottom: 1rem;
        border-bottom: 1px solid #e5e5e5;
        margin-bottom: 1rem;

        .mobile-back-btn {
          display: inline-flex;
          align-items: center;
          justify-content: center;
          width: 26px;
          height: 26px;
          margin: 0;
          padding: 0;
          padding-top: 2px;
          padding-right: 4px;
          margin-right: 6px;
          border: none;
          border-radius: 6px;
          background: transparent;
          cursor: pointer;
          flex-shrink: 0;

          img {
            width: 24px;
            height: 24px;
            display: block;
          }

          &:active {
            background: #efefee;
          }
        }

        .close {
          width: 24px;
          height: 24px;
          border-radius: 50%;
          cursor: pointer;

          img {
            width: 100%;
            height: 100%;
          }
        }
      }

      .panel-box {
        width: 100%;
        height: 100%;
        min-width: 0;
        overflow-y: auto;
        overflow-x: hidden;
        padding-right: 2px;
        padding-bottom: 1rem;

        .mt-16 {
          margin-top: 16px;
        }

        .status-message {
          margin: 0 0 12px;
          font-size: 12px;
          line-height: 18px;
          padding: 8px 10px;
          border-radius: 6px;
        }

        .is-error {
          color: #c53030;
          background: #fff5f5;
          border: 1px solid #fed7d7;
        }

        .is-success {
          color: #276749;
          background: #f0fff4;
          border: 1px solid #c6f6d5;
        }

        .setting-item {
          display: flex;
          justify-content: space-between;
          align-items: center;

          &.avatar-setting {
            align-items: flex-start;
          }

          .avatar-upload-row {
            margin-top: 10px;
          }

          .avatar-upload-preview {
            position: relative;
            display: block;
            width: 72px;
            height: 72px;
            padding: 0;
            border: 0;
            border-radius: 50%;
            overflow: hidden;
            flex: 0 0 72px;
            background: #f3f3f3;
            cursor: pointer;

            &:disabled {
              cursor: wait;
            }

            &:hover:not(:disabled) .avatar-upload-overlay,
            &:focus-visible .avatar-upload-overlay {
              opacity: 1;
            }

            > img {
              width: 100%;
              height: 100%;
              object-fit: cover;
              display: block;
            }
          }

          .avatar-upload-overlay {
            position: absolute;
            inset: 0;
            display: flex;
            align-items: center;
            justify-content: center;
            background: rgba(0, 0, 0, 0.45);
            opacity: 0;
            transition: opacity 0.15s ease;

            &.is-visible {
              opacity: 1;
            }
          }

          .avatar-upload-edit-icon {
            width: 22px;
            height: 22px;
            filter: brightness(0) invert(1);
          }

          .avatar-upload-loading {
            font-size: 11px;
            color: #fff;
            line-height: 1.2;
          }

          .avatar-file-input {
            display: none;
          }

          .item-left {
            .item-title {
              font-size: 14px;
              color: #333;
              font-weight: bold;
              line-height: 24px;
            }
            .item-value {
              font-size: 12px;
              color: #666;
              line-height: 20px;
            }
          }

          .item-right {
            display: flex;
            align-items: center;
            gap: 8px;
            margin-top: 24px;
          }
        }

        .form-box {
          margin-top: 8px;
          border: 1px solid #f0f0f0;
          border-radius: 6px;
          padding: 10px;
          display: flex;
          flex-direction: column;
          gap: 8px;
        }

        .form-actions {
          display: flex;
          justify-content: flex-end;
          margin-top: 10px;
        }

        .privacy-divider {
          border-top: 1px dashed #ececec;
          margin: 14px 0 10px;
        }

        .privacy-title {
          font-size: 14px;
          color: #333;
          font-weight: bold;
          line-height: 24px;
          margin-bottom: 8px;
        }

        .privacy-head {
          display: flex;
          align-items: center;
          justify-content: space-between;
          gap: 10px;
          margin-bottom: 10px;
        }

        .privacy-head-actions {
          display: inline-flex;
          align-items: center;
          gap: 6px;
          flex-wrap: wrap;
        }

        .security-box {
          .security-head {
            display: flex;
            align-items: center;
            justify-content: space-between;
            gap: 10px;
            margin-bottom: 8px;
          }

          .item-value {
            margin: 0 0 12px;
            font-size: 12px;
            color: #666;
            line-height: 20px;
          }
        }

        .storage-box {
          .storage-desc {
            margin: 8px 0 0;
            font-size: 12px;
            color: #666;
            line-height: 20px;
          }

          .storage-calculated-at {
            margin: 12px 0 0;
            font-size: 12px;
            color: #9ca3af;
          }

          .storage-quota-input {
            max-width: 160px;
          }

          .storage-actions {
            display: inline-flex;
            flex-wrap: wrap;
            gap: 6px;
            justify-content: flex-end;
          }
        }

        .passkey-list {
          display: flex;
          flex-direction: column;
          gap: 10px;
        }

        .passkey-row {
          border: 1px solid #f0f0f0;
          border-radius: 6px;
          padding: 10px 12px;
          display: flex;
          align-items: center;
          justify-content: space-between;
          gap: 10px;
          background: #fff;
        }

        .passkey-main {
          min-width: 0;
        }

        .passkey-title {
          margin: 0;
          font-size: 13px;
          color: #333;
          font-weight: 600;
          line-height: 1.3;
        }

        .passkey-sub {
          margin: 4px 0 0;
          font-size: 12px;
          color: #666;
          line-height: 1.4;
          font-family:
            ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New',
            monospace;
        }

        .passkey-meta {
          display: flex;
          flex-direction: column;
          align-items: flex-end;
          gap: 4px;
          font-size: 12px;
          line-height: 1.3;
          color: #9ca3af;
          text-align: right;
          white-space: nowrap;
        }

        .privacy-name {
          font-size: 13px;
          color: #222;
          font-weight: 600;
        }

        .privacy-policy {
          font-size: 12px;
          color: #555;
        }

        .privacy-collapse {
          border: 1px solid #f0f0f0;
          border-radius: 6px;
          margin-bottom: 10px;
          overflow: hidden;
          background: #fff;
        }

        .privacy-collapse-summary {
          list-style: none;
          display: flex;
          justify-content: space-between;
          align-items: center;
          flex-wrap: wrap;
          gap: 12px;
          padding: 10px;
          background: #fafafa;
          cursor: pointer;

          &::-webkit-details-marker {
            display: none;
          }
        }

        .privacy-collapse-left {
          display: flex;
          align-items: center;
          flex-wrap: wrap;
          gap: 8px;
          min-width: 0;
          flex: 1;
        }

        .privacy-collapse-title {
          font-size: 14px;
          font-weight: bold;
          color: #333;
        }

        .privacy-collapse-right {
          min-width: 220px;
          display: flex;
          align-items: center;
          justify-content: flex-end;
          gap: 8px;
          margin-left: auto;
        }

        .privacy-collapse-body {
          border-top: 1px solid #f0f0f0;
          min-width: 0;
        }

        .privacy-table,
        .account-table {
          width: 100%;
          border-collapse: separate;
          border-spacing: 0;
          table-layout: fixed;

          th,
          td {
            border-bottom: 1px solid #f3f3f3;
            padding: 8px 10px;
            font-size: 12px;
            color: #333;
            text-align: left;
            vertical-align: middle;
            word-break: break-all;
            background: #fff;
          }

          th {
            color: #666;
            font-weight: 600;
            background: #fafafa;
          }
        }

        .privacy-table .is-mono {
          font-family:
            ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New',
            monospace;
          color: #666;
        }

        .major-name-input {
          width: 180px;
        }

        .table-select {
          min-width: 110px;
        }

        .table-actions {
          display: flex;
          flex-wrap: wrap;
          gap: 6px;
        }

        .settings-picker {
          width: 100%;
          min-width: 0;
        }

        .masonry-cols-picker {
          position: relative;
          display: inline-flex;
          align-items: center;
          min-height: 32px;
          border-radius: 6px;
          border: 1px solid #e5e5e5;
          background: #fff;
          padding: 0 6px;
          box-sizing: border-box;

          &.is-menu-open {
            z-index: 30;
          }

          &__control {
            position: relative;
            width: 100%;
            min-width: 0;
          }

          &__trigger {
            display: inline-flex;
            align-items: center;
            justify-content: space-between;
            gap: 6px;
            width: 100%;
            min-width: 0;
            height: 30px;
            padding: 0 4px;
            margin: 0;
            border: 0;
            border-radius: 6px;
            background: transparent;
            font-size: 12px;
            color: #333;
            text-align: left;
            cursor: pointer;

            &:disabled {
              cursor: not-allowed;
              opacity: 0.6;
            }

            &:focus-visible {
              outline: none;
            }

            &.is-open .masonry-cols-picker__chevron {
              transform: rotate(180deg);
            }
          }

          &__value {
            flex: 1;
            min-width: 0;
            overflow: hidden;
            text-overflow: ellipsis;
            white-space: nowrap;
          }

          &__chevron {
            flex-shrink: 0;
            width: 10px;
            height: 6px;
            background: #9a9a9a;
            clip-path: polygon(0 0, 100% 0, 50% 100%);
            transition: transform 0.15s ease;
          }

          &__menu {
            position: absolute;
            top: calc(100% + 6px);
            left: -6px;
            right: -6px;
            margin: 0;
            padding: 4px;
            list-style: none;
            border: 1px solid #e9e9e9;
            border-radius: 8px;
            background: #fff;
            box-shadow: 0 12px 24px rgba(0, 0, 0, 0.12);
            box-sizing: border-box;
          }

          &__menu-item {
            margin: 0;
            padding: 0;
          }

          &__option {
            display: block;
            width: 100%;
            margin: 0;
            padding: 7px 8px;
            border: 0;
            border-radius: 6px;
            background: transparent;
            color: #555;
            font-size: 12px;
            text-align: left;
            cursor: pointer;

            &:hover {
              background: #f5f5f5;
              color: #222;
            }

            &[aria-selected='true'] {
              background: #efefee;
              color: #111;
              font-weight: 600;
            }
          }
        }

        .table-wrap {
          width: 100%;
          max-width: 100%;
          border: 1px solid #f0f0f0;
          border-radius: 6px;
          overflow-x: auto;
          scrollbar-gutter: stable both-edges;
        }

        .table-scroll-wrap {
          width: 100%;
          max-width: 100%;
          overflow-x: auto;
          scrollbar-gutter: stable both-edges;
        }

        .privacy-table {
          width: max-content;
          min-width: 100%;
          table-layout: auto;

          th,
          td {
            white-space: nowrap;
          }

          th:first-child,
          td:first-child {
            position: sticky;
            left: 0;
            min-width: 180px;
            z-index: 3;
            background: #fff;
            box-shadow: 1px 0 0 #f0f0f0;
          }

          th:last-child,
          td:last-child {
            position: sticky;
            right: 0;
            min-width: 152px;
            z-index: 3;
            background: #fff;
            box-shadow: -1px 0 0 #f0f0f0;
          }

          th:first-child,
          th:last-child {
            z-index: 4;
            background: #fafafa;
          }
        }

        .account-table {
          width: max-content;
          min-width: 100%;
          table-layout: auto;

          th,
          td {
            white-space: nowrap;
          }

          th:first-child,
          td:first-child {
            position: sticky;
            left: 0;
            min-width: 180px;
            z-index: 3;
            background: #fff;
            box-shadow: 1px 0 0 #f0f0f0;
          }

          th:last-child,
          td:last-child {
            position: sticky;
            right: 0;
            min-width: 152px;
            z-index: 3;
            background: #fff;
            box-shadow: -1px 0 0 #f0f0f0;
          }

          th:first-child,
          th:last-child {
            z-index: 4;
            background: #fafafa;
          }
          .align-center {
            text-align: center;
          }
        }
      }
    }
  }
}

.inline-dialog {
  width: 100%;
  height: auto;
  background: transparent;
  border-radius: 0;
  padding: 0;
  box-sizing: border-box;
  box-shadow: none;
  border: none;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.account-perm-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.account-password-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;

  .nav-add__label {
    margin-bottom: 0;
  }
}

.account-edit-pwd-box {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.inline-dialog-title {
  margin: 0 0 6px;
  font-size: 15px;
  font-weight: 700;
  color: #333;
}

.inline-dialog-desc {
  margin: 0 0 10px;
  font-size: 12px;
  color: #666;
  line-height: 18px;
}

.inline-dialog-error {
  margin: 8px 0 0;
  font-size: 12px;
  color: #c53030;
}

.inline-dialog-hint {
  margin: 6px 0 0;
  font-size: 12px;
  color: #666;
  line-height: 18px;

  &.is-warn {
    color: #c53030;
  }
}

.inline-dialog-actions {
  margin-top: 12px;
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.nav-add__label {
  display: block;
  margin: 0.65rem 0 0.35rem;
  font-size: 12px;
  font-weight: 600;
  color: #000;
}
</style>
