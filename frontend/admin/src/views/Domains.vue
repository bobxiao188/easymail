<template>
  <div class="mail-system-container">
    <el-card class="glass-card main-card">
      <el-tabs v-model="activeTab" class="domain-tabs">
        <el-tab-pane :label="t('domains.domains')" name="domains">
          <div class="domain-panel">
            <div class="search-box">
              <div class="search-box-left">
                <el-input v-model="domainSearchQuery" :placeholder="t('domains.searchDomain')" class="easy-input"
                  clearable style="width: 12rem;" />
                <el-checkbox v-model="showDeletedDomains" @change="loadDomains">{{ t('domains.showDeleted') }}</el-checkbox>
                <el-button type="primary" size="default" class="easy-button" @click="loadDomains">{{ t('domains.search')
                }}</el-button>
              </div>
              <div class="search-box-right">
                <el-button type="primary" size="default" @click="openAddDomainDialog">
                  <el-icon :size="20">
                    <Plus />
                  </el-icon>
                  <span>{{ t('domains.addDomain') }}</span>
                </el-button>
              </div>
            </div>

            <div class="domain-list">
              <el-table :data="filteredDomains" class="easy-table"
                highlight-current-row :row-class-name="getDomainRowClass">
                <el-table-column prop="id" :label="t('domains.id')" min-width="180" />
                <el-table-column prop="name" :label="t('domains.domainName')" min-width="200">
                  <template #default="scope">
                    <span class="domain-name-link" @click="handleDomainNameClick(scope.row)"
                      :style="{ cursor: 'pointer', color: '#409EFF' }">
                      {{ scope.row.name }}
                      <el-icon v-if="scope.row.isDeleted"><Warning /></el-icon>
                    </span>
                  </template>
                </el-table-column>
                <el-table-column prop="description" :label="t('domains.description')" min-width="200">
                  <template #default="scope">
                    {{ scope.row.description || t('domains.noDescription') }}
                  </template>
                </el-table-column>
                <el-table-column prop="active" :label="t('domains.status')" width="160">
                  <template #default="scope">
                    <el-switch v-model="scope.row.active" @change="handleDomainStatusChange(scope.row)"
                      :active-value="true" :inactive-value="false" class="easy-switch" />
                  </template>
                </el-table-column>
                <el-table-column prop="createTime" :label="t('domains.createTime')" width="200">
                  <template #default="scope">
                    <el-tooltip :content="formatDateTime(scope.row.createTime)" placement="top">
                      {{ formatDate(scope.row.createTime) }}
                    </el-tooltip>
                  </template>
                </el-table-column>
                <el-table-column :label="t('domains.operation')" width="240">
                  <template #default="scope">
                    <div class="action-buttons">
                      <button type="button" class="easy-icon-button" @click="openDKIMDialog(scope.row)" title="DKIM">
                        <el-icon>
                          <Lock />
                        </el-icon>
                      </button>
                      <button type="button" class="easy-icon-button" @click="openEditDomainDialog(scope.row)">
                        <el-icon>
                          <Edit />
                        </el-icon>
                      </button>
                      <button type="button" class="easy-icon-button is-danger" v-if="!scope.row.isDeleted"
                        @click="handleDeleteDomain(scope.row.id)">
                        <el-icon>
                          <Delete />
                        </el-icon>
                      </button>
                      <button type="button" class="easy-icon-button is-danger" @click="handlePurgeDomain(scope.row)" v-if="scope.row.isDeleted" :title="t('domains.purgeDomain')">
                        <el-icon>
                          <Remove />
                        </el-icon>
                      </button>
                    </div>
                  </template>
                </el-table-column>
              </el-table>

              <div class="pagination-container">
                <el-pagination v-model:current-page="domainCurrentPage" v-model:page-size="domainPageSize"
                  :page-sizes="[10, 20, 50]" layout="total, sizes, prev, pager, next, jumper" :total="totalDomains"
                  @size-change="handleSizeChange" @current-change="handleCurrentChange" class="easy-pagination" />
              </div>
            </div>
          </div>
        </el-tab-pane>

        <el-tab-pane :label="t('domains.accounts')" name="accounts">
          <div class="account-panel">
            <div class="search-bar">
              <div class="search-bar-left">
                <el-select v-model="selectedDomainId" :placeholder="t('domains.selectDomain')" class="easy-select"
                  style="width: 12rem;">
                  <el-option v-for="domain in domains" :key="domain.id" :label="domain.name" :value="domain.id" />
                </el-select>
                <el-select v-model="accountStatusFilter" :placeholder="t('domains.status')" class="easy-select"
                  :disabled="!selectedDomainId" style="width: 8rem;">
                  <el-option :label="t('domains.all')" value="" />
                  <el-option :label="t('domains.enabled')" value="1" />
                  <el-option :label="t('domains.disabled')" value="0" />
                </el-select>
                <el-input v-model="accountSearchQuery" :placeholder="t('domains.searchAccount')" class="easy-input"
                  style="width: 12rem;" :disabled="!selectedDomainId" />
                <el-button type="primary" size="default" class="easy-button" @click="handleSearchAccounts"
                  :disabled="!selectedDomainId">{{ t('domains.search') }}</el-button>

              </div>

              <div class="search-bar-right">
                <el-button type="primary" size="default" class="easy-button" :disabled="!selectedDomainId"
                  @click="openAddAccountDialog">
                  <el-icon>
                    <Plus />
                  </el-icon>
                  <span>{{ t('domains.addAccount') }}</span>
                </el-button>
              </div>
            </div>

            <div class="account-list" v-if="selectedDomainId">
              <el-table :data="filteredAccounts" class="easy-table">
                <el-table-column prop="id" :label="t('domains.id')" min-width="180" />
                <el-table-column :label="t('domains.emailAddress')" min-width="200">
                  <template #default="scope">
                    <div class="email-cell">
                      {{ scope.row.username }}@{{ currentDomainName }}
                    </div>
                  </template>
                </el-table-column>
                <el-table-column prop="passwordExpireTime" :label="t('domains.passwordExpireTime')" min-width="220">
                  <template #default="scope">
                    {{ isZeroTime(scope.row.passwordExpireTime) ? t('domains.neverExpire') :
                      formatDateTime(scope.row.passwordExpireTime) }}
                  </template>
                </el-table-column>
                <el-table-column prop="storageQuota" :label="t('domains.quota')" width="160" />
                <el-table-column :label="t('domains.status')" width="200">
                  <template #default="scope">
                    <template v-if="scope.row.isDeleted">
                      <el-tag type="danger" size="small">{{ t('domains.accountDeleted') }}</el-tag>
                    </template>
                    <el-switch v-else v-model="scope.row.active" @change="handleAccountStatusChange(scope.row)"
                      :active-value="true" :inactive-value="false" class="easy-switch" />
                  </template>
                </el-table-column>
                <el-table-column prop="createTime" :label="t('domains.createTime')" width="200">
                  <template #default="scope">
                    <el-tooltip :content="formatDateTime(scope.row.createTime)" placement="top">
                      {{ formatDate(scope.row.createTime) }}
                    </el-tooltip>
                  </template>
                </el-table-column>
                <el-table-column :label="t('domains.operation')" width="200">
                  <template #default="scope">
                    <div class="action-buttons">
                      <button type="button" class="easy-icon-button" @click="openEditAccountDialog(scope.row)">
                        <el-icon>
                          <Edit />
                        </el-icon>
                      </button>
                      <button type="button" class="easy-icon-button" @click="openChangePasswordDialog(scope.row)">
                        <el-icon>
                          <key />
                        </el-icon>
                      </button>
                      <button type="button" class="easy-icon-button is-danger" v-if="!scope.row.isDeleted"
                        @click="handleDeleteAccount(scope.row.id)">
                        <el-icon>
                          <Delete />
                        </el-icon>
                      </button>
                      <button type="button" class="easy-icon-button is-danger" @click="handlePurgeAccount(scope.row)" v-if="scope.row.isDeleted" :title="t('domains.purgeAccount')">
                        <el-icon>
                          <Remove />
                        </el-icon>
                      </button>
                    </div>
                  </template>
                </el-table-column>
              </el-table>

              <div class="pagination-container">
                <el-pagination v-model:current-page="accountCurrentPage" v-model:page-size="accountPageSize"
                  :page-sizes="[10, 20, 50]" layout="total, sizes, prev, pager, next, jumper" :total="totalAccounts"
                  @size-change="handleAccountSizeChange" @current-change="handleAccountCurrentChange"
                  class="easy-pagination" />
              </div>
            </div>

            <div v-else class="empty-tip">
              {{ t('domains.selectDomain') }}
            </div>
          </div>
        </el-tab-pane>
      </el-tabs>
    </el-card>

    <!-- 新增/编辑域名对话框 -->
    <el-dialog :title="domainDialogTitle" v-model="domainDialogVisible" width="650px" class="easy-dialog"
      destroy-on-close align-center :close-on-click-modal="false" :close-on-press-escape="false">
      <el-form :model="domainForm" :rules="domainRules" ref="domainFormRef" label-width="160px" class="easy-form">
        <el-form-item :label="t('domains.domainName')" prop="name">
          <el-input v-model="domainForm.name" :placeholder="t('domains.domainNamePlaceholder')" class="easy-input" />
        </el-form-item>
        <el-form-item :label="t('domains.description')" prop="description">
          <el-input v-model="domainForm.description" :placeholder="t('domains.descriptionPlaceholder')" type="textarea"
            class="easy-input" />
        </el-form-item>
        <el-form-item :label="t('domains.status')">
          <el-switch v-model="domainForm.active" class="easy-switch" />
        </el-form-item>
        <el-form-item v-if="domainForm.isDeleted" :label="t('domains.isDeleted')">
          <el-tag type="danger">{{ t('domains.domainDeleted') }}</el-tag>
        </el-form-item>
        <el-form-item v-if="domainForm.isDeleted" :label="t('domains.reactivate')">
          <el-switch v-model="domainForm.reactivate" :disabled="!domainForm.isDeleted" class="easy-switch" />
          <span style="margin-left: 8px; font-size: 12px; color: #666;">{{ t('domains.reactivateTip') }}</span>
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button type="danger" size="default" class="easy-button" @click="domainDialogVisible = false">
            <span>{{ t('common.cancel') }}</span>
          </el-button>
          <el-button type="primary" size="default" class="easy-button" @click="handleSaveDomain">
            <span>{{ t('common.save') }}</span>
          </el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 新增/编辑账号对话框 -->
    <el-dialog :title="accountDialogTitle" v-model="accountDialogVisible" width="650px" class="easy-dialog"
      destroy-on-close align-center :close-on-click-modal="false" :close-on-press-escape="false">
      <el-form :model="accountForm" :rules="accountRules" ref="accountFormRef" label-width="160px" class="easy-form">
        <el-form-item :label="t('domains.username')" prop="username">
          <el-input v-model="accountForm.username" :placeholder="t('domains.usernamePlaceholder')" class="easy-input"
            :disabled="!!accountForm.id">
            <template #append>@{{ currentDomainName }}</template>
          </el-input>
        </el-form-item>
        <el-form-item :label="t('domains.password')" prop="password" v-if="!accountForm.id">
          <el-input v-model="accountForm.password" type="password" :placeholder="t('domains.passwordPlaceholder')"
            class="easy-input" show-password />
        </el-form-item>
        <el-form-item :label="t('domains.quota')" prop="storageQuota">
          <el-input-number v-model="accountForm.storageQuota" :min="0" :step="100" class="easy-input-number" />
        </el-form-item>
        <el-form-item :label="t('domains.passwordExpireTime')" prop="passwordExpireTime">
          <el-date-picker v-model="accountForm.passwordExpireTime" type="datetime"
            :placeholder="t('domains.selectPasswordExpireTime')" class="easy-input" />
        </el-form-item>
        <el-form-item :label="t('domains.status')">
          <el-switch v-model="accountForm.active" class="easy-switch" />
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button size="default" class="easy-button" @click="accountDialogVisible = false">
            <span>{{ t('common.cancel') }}</span>
          </el-button>
          <el-button type="primary" size="default" class="easy-button" @click="handleSaveAccount">
            <span>{{ t('common.save') }}</span>
          </el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 修改密码对话框 -->
    <el-dialog :title="t('domains.changePassword')" v-model="passwordDialogVisible" width="650px" class="easy-dialog"
      destroy-on-close align-center :close-on-click-modal="false" :close-on-press-escape="false">
      <el-form :model="passwordForm" :rules="passwordRules" ref="passwordFormRef" label-width="150px" class="easy-form">
        <el-form-item :label="t('domains.newPassword')" prop="newPassword">
          <el-input v-model="passwordForm.newPassword" type="password"
            :placeholder="t('domains.newPasswordPlaceholder')" class="easy-input" show-password />
        </el-form-item>
        <el-form-item :label="t('domains.confirmPassword')" prop="confirmPassword">
          <el-input v-model="passwordForm.confirmPassword" type="password"
            :placeholder="t('domains.confirmPasswordPlaceholder')" class="easy-input" show-password />
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button size="default" class="easy-button" @click="passwordDialogVisible = false">
            <span>{{ t('common.cancel') }}</span>
          </el-button>
          <el-button type="primary" size="default" class="easy-button" @click="handleChangePassword">
            <span>{{ t('common.save') }}</span>
          </el-button>
        </div>
      </template>
    </el-dialog>

    <!-- DKIM 配置对话框 -->
    <el-dialog :title="t('domains.dkimSettings')" v-model="dkimDialogVisible" width="53%" class="easy-dialog"
      destroy-on-close align-center :close-on-click-modal="false" :close-on-press-escape="false">
      <div class="dkim-dialog-content">
        <el-alert type="info" :closable="false" :title="t('domains.dkimDescription')" show-icon class="mb-4 easy-alert" />

        <el-form :model="dkimForm" :rules="dkimRules" ref="dkimFormRef" label-width="140px" class="easy-form">
          <el-form-item :label="t('domains.domainName')">
            <el-input :value="dkimDomainName" disabled class="easy-input" />
          </el-form-item>

          <el-form-item :label="t('domains.dkimEnabled')">
            <el-switch v-model="dkimForm.enabled" class="easy-switch" />
          </el-form-item>

          <div v-if="dkimForm.enabled">
            <el-form-item :label="t('domains.dkimSelector')" prop="selector">
              <div class="selector-input-group">
                <el-input v-model="dkimForm.selector" :placeholder="t('domains.dkimSelectorPlaceholder')"
                  class="easy-input" />
                <el-button type="primary" size="default" class="easy-button" @click="handleGenerateDKIMKey"
                  :loading="generatingKey">
                  {{ t('domains.dkimGenerateKey') }}
                </el-button>
              </div>
            </el-form-item>

            <el-form-item :label="t('domains.dkimPrivateKey')" prop="privateKey">
              <el-input v-model="dkimForm.privateKey" type="textarea" :rows="6"
                :placeholder="t('domains.dkimPrivateKeyPlaceholder')" class="easy-input" />
            </el-form-item>

            <el-form-item v-if="dkimForm.publicKey" :label="t('domains.dkimDNSRecord')">
              <div class="dns-record-box">
                <p class="dns-record-hint">{{ t('domains.dkimDNSRecordHint') }}</p>
                <div class="dns-record-item">
                  <span class="dns-record-label">{{ t('domains.dkimDNSRecord') }}:</span>
                  <code class="dns-record-value">{{ dkimDNSRecordName }}</code>
                </div>
                <div class="dns-record-item">
                  <span class="dns-record-label">Type:</span>
                  <code class="dns-record-value">TXT</code>
                </div>
                <div class="dns-record-item">
                  <span class="dns-record-label">Value:</span>
                  <div class="dns-record-value-block">
                    <code class="dns-record-value">{{ dkimForm.publicKey }}</code>
                    <el-button type="primary" size="small" text @click="handleCopyDNSRecord">
                      <el-icon>
                        <CopyDocument />
                      </el-icon>
                      {{ t('common.copy') }}
                    </el-button>
                  </div>
                </div>
              </div>
            </el-form-item>
          </div>
        </el-form>
      </div>
      <template #footer>
        <div class="dialog-footer">
          <el-button size="default" class="easy-button" @click="dkimDialogVisible = false">
            <span>{{ t('common.cancel') }}</span>
          </el-button>
          <el-button type="primary" size="default" class="easy-button" @click="handleSaveDKIM" :loading="savingDKIM">
            <span>{{ t('common.save') }}</span>
          </el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { domainApi, type Domain } from '../api/domain'
import { accountApi, type Account } from '../api/account'
import { ElMessage, ElMessageBox, type FormInstance } from 'element-plus'
import { Lock, CopyDocument, Remove } from '@element-plus/icons-vue'
import { formatDate, formatDateTime, isZeroTime } from '../utils/times'

const { t } = useI18n()

const activeTab = ref('domains')
const domains = ref<Domain[]>([])
const accounts = ref<Account[]>([])
const selectedDomainId = ref<string | null>(null)

const currentDomainName = computed(() => {
  if (!selectedDomainId.value) return ''
  const domain = domains.value.find(d => d.id === selectedDomainId.value)
  return domain?.name || ''
})

const domainSearchQuery = ref('')
const accountSearchQuery = ref('')
const accountStatusFilter = ref('')
const accountCurrentPage = ref(1)
const accountPageSize = ref(10)
const totalAccounts = ref(0)

const domainCurrentPage = ref(1)
const domainPageSize = ref(10)
const totalDomains = ref(0)

const domainDialogVisible = ref(false)
const accountDialogVisible = ref(false)
const passwordDialogVisible = ref(false)
const domainDialogTitle = ref(t('domains.addDomain'))
const accountDialogTitle = ref(t('domains.addAccount'))
const domainFormRef = ref<FormInstance | null>(null)
const accountFormRef = ref<FormInstance | null>(null)
const passwordFormRef = ref<FormInstance | null>(null)

const domainForm = reactive({
  id: null as string | null,
  name: '',
  description: '',
  active: true,
  isDeleted: false,
  reactivate: false
})

const accountForm = reactive({
  id: null as number | null,
  username: '',
  password: '',
  storageQuota: 1000,
  active: true,
  passwordExpireTime: null as Date | string | null
})

const passwordForm = reactive({
  accountId: null as number | null,
  newPassword: '',
  confirmPassword: ''
})

const dkimDialogVisible = ref(false)
const dkimFormRef = ref<FormInstance | null>(null)
const dkimDomainId = ref<string | null>(null)
const dkimDomainName = ref('')
const generatingKey = ref(false)
const savingDKIM = ref(false)

const dkimForm = reactive({
  enabled: false,
  selector: 'default',
  privateKey: '',
  publicKey: ''
})

const dkimDNSRecordName = computed(() => {
  if (dkimForm.selector && dkimDomainName.value) {
    return `${dkimForm.selector}._domainkey.${dkimDomainName.value}`
  }
  return ''
})

const dkimRules = {
  selector: [
    { required: true, message: t('domains.dkimSelectorRequired'), trigger: 'blur' }
  ],
  privateKey: [
    { required: true, message: t('domains.dkimPrivateKeyRequired'), trigger: 'blur' }
  ]
}

const domainRules = {
  name: [
    { required: true, message: t('domains.domainNameRequired'), trigger: 'blur' }
  ],
}

const accountRules = {
  username: [
    { required: true, message: t('domains.usernameRequired'), trigger: 'blur' }
  ],
  password: [
    { required: true, message: t('domains.passwordRequired'), trigger: 'blur' },
    { min: 8, max: 32, message: t('domains.passwordMinLength'), trigger: 'blur' }
  ],
  passwordExpireTime: [
    { required: false, message: t('domains.passwordExpireTimeRequired'), trigger: 'blur' }
  ],
  storageQuota: [
    { required: true, message: t('domains.storageQuotaRequired'), trigger: 'blur' },
    { type: 'number', message: t('domains.storageQuotaNumberError'), trigger: 'blur' }
  ],
  active: [
    { required: true, message: t('domains.activeRequired'), trigger: 'blur' }
  ]
}

const passwordRules = {
  newPassword: [
    { required: true, message: t('domains.newPasswordRequired'), trigger: 'blur' },
    { min: 6, message: t('domains.newPasswordMinLength'), trigger: 'blur' }
  ],
  confirmPassword: [
    { required: true, message: t('domains.confirmPasswordRequired'), trigger: 'blur' },
    {
      validator: (_rule: any, value: string, callback: (error?: Error) => void) => {
        if (value !== passwordForm.newPassword) {
          callback(new Error(t('domains.passwordMismatch')))
        } else {
          callback()
        }
      },
      trigger: 'blur'
    }
  ]
}

const showDeletedDomains = ref(false)

const filteredDomains = computed(() => {
  if (showDeletedDomains.value) {
    return domains.value
  }
  return domains.value.filter(d => !d.isDeleted)
})

const filteredAccounts = computed(() => {
  return accounts.value
})

const handleDomainNameClick = (row: Domain) => {
  selectedDomainId.value = row.id
  activeTab.value = 'accounts'
}

const getDomainRowClass = ({ row }: { row: Domain }) => {
  return row.id === selectedDomainId.value ? 'selected-row' : ''
}

const handleSearchAccounts = () => {
  loadAccounts()
}

const loadDomains = async () => {
  try {
    const res = await domainApi.list({
      keyword: domainSearchQuery.value,
      page: domainCurrentPage.value,
      pageSize: domainPageSize.value,
      include_deleted: showDeletedDomains.value
    })
    if (res.code === 0) {
      domains.value = res.data
      if (res.meta) {
        totalDomains.value = (res.meta as any).total || 0
      }
    }
  } catch (error) {
    ElMessage.error(t('domains.getDomainListFailed'))
  }
}

const loadAccounts = async () => {
  if (!selectedDomainId.value) return
  try {
    let status: number | undefined = undefined
    if (accountStatusFilter.value === '1') {
      status = 1
    } else if (accountStatusFilter.value === '0') {
      status = 0
    } else {
      status = -1
    }
    const res = await accountApi.list({
      domainId: selectedDomainId.value,
      keyword: accountSearchQuery.value,
      status: status,
      page: accountCurrentPage.value,
      pageSize: accountPageSize.value
    })
    if (res.code === 0) {
      accounts.value = res.data
      if (res.meta) {
        totalAccounts.value = (res.meta as any).total || 0
      }
    }
  } catch (error) {
    ElMessage.error(t('domains.getAccountListFailed'))
  }
}

watch(activeTab, (newTab) => {
  if (newTab === 'accounts' && !selectedDomainId.value && domains.value.length > 0) {
    if(domains.value.length > 0) {
      selectedDomainId.value = domains.value[0].id
    }
  }
})

watch(selectedDomainId, (newVal) => {
  if (newVal) {
    accountCurrentPage.value = 1
    loadAccounts()
  } else {
    accounts.value = []
  }
})

const openAddDomainDialog = () => {
  domainDialogTitle.value = t('domains.addDomain')
  Object.assign(domainForm, {
    id: null,
    name: '',
    description: '',
    active: true
  })
  domainDialogVisible.value = true
}

const openEditDomainDialog = (domain: Domain) => {
  domainDialogTitle.value = t('domains.editDomain')
  Object.assign(domainForm, {
    id: domain.id,
    name: domain.name,
    description: domain.description,
    active: domain.active,
    isDeleted: domain.isDeleted || false,
    reactivate: domain.isDeleted // 如果是已删除状态，默认勾选重新激活
  })
  domainDialogVisible.value = true
}

const handleSaveDomain = async () => {
  if (!domainFormRef.value) return

  try {
    const valid = await domainFormRef.value.validate()
    if (!valid) return

    if (domainForm.id) {
      // 如果勾选了重新激活，则清除删除状态
      const isReactivating = domainForm.reactivate && domainForm.isDeleted
      
      await domainApi.update(domainForm.id, {
        name: domainForm.name,
        description: domainForm.description,
        active: domainForm.active || isReactivating,
        isDeleted: domainForm.isDeleted && !isReactivating
      })
      
      if (isReactivating) {
        ElMessage.success(t('domains.domainReactivated'))
      } else {
        ElMessage.success(t('domains.domainUpdateSuccess'))
      }
    } else {
      await domainApi.create({
        name: domainForm.name,
        description: domainForm.description
      })
      ElMessage.success(t('domains.domainCreateSuccess'))
    }
    domainDialogVisible.value = false
    await loadDomains()
  } catch (error: any) {
    const errorMessage = error.response?.data?.message || error.message || t('domains.operationFailed')
    ElMessage.error(errorMessage)
  }
}

const handleDomainStatusChange = async (domain: Domain) => {
  try {
    await domainApi.toggle(domain.id)
    ElMessage.success(t('domains.statusUpdateSuccess'))
  } catch (error) {
    domain.active = !domain.active
    ElMessage.error(t('domains.statusUpdateFailed'))
  }
}

const handleDeleteDomain = async (id: string) => {
  try {
    await ElMessageBox.confirm(
      t('domains.deleteDomainConfirm'),
      t('domains.deleteConfirm'),
      {
        confirmButtonText: t('common.confirm'),
        cancelButtonText: t('common.cancel'),
        type: 'warning'
      }
    )

    await domainApi.delete(id)
    ElMessage.success(t('domains.domainDeleteSuccess'))
    if (selectedDomainId.value === id) {
      selectedDomainId.value = null
      accounts.value = []
    }
    await loadDomains()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error(t('domains.deleteFailed'))
    }
  }
}

const handlePurgeDomain = async (domain: Domain) => {
  try {
    await ElMessageBox.confirm(
      t('domains.purgeDomainConfirm'),
      t('domains.purgeDomain'),
      {
        confirmButtonText: t('common.confirm'),
        cancelButtonText: t('common.cancel'),
        type: 'warning'
      }
    )

    await domainApi.purge(domain.id)
    ElMessage.success(t('domains.purgeDomainSuccess'))
    if (selectedDomainId.value === domain.id) {
      selectedDomainId.value = null
      accounts.value = []
    }
    await loadDomains()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error(t('domains.purgeDomainFailed'))
    }
  }
}

const openAddAccountDialog = () => {
  if (!selectedDomainId.value) return
  accountDialogTitle.value = t('domains.addAccount')
  Object.assign(accountForm, {
    id: null,
    username: '',
    password: '',
    storageQuota: 1000,
    passwordExpireTime: null,
    active: true
  })
  accountDialogVisible.value = true
}

const openEditAccountDialog = (account: Account) => {
  accountDialogTitle.value = t('domains.editAccount')
  Object.assign(accountForm, {
    id: account.id,
    username: account.username,
    password: '',
    storageQuota: account.storageQuota,
    passwordExpireTime: account.passwordExpireTime,
    active: account.active
  })
  accountDialogVisible.value = true
}

const openChangePasswordDialog = (account: Account) => {
  Object.assign(passwordForm, {
    accountId: account.id,
    newPassword: '',
    confirmPassword: ''
  })
  passwordDialogVisible.value = true
}

const handleSaveAccount = async () => {
  if (!accountFormRef.value) return

  try {
    const valid = await accountFormRef.value.validate()
    if (!valid) return

    if (accountForm.id) {
      await accountApi.update(accountForm.id, {
        username: accountForm.username,
        domainId: selectedDomainId.value!,
        active: accountForm.active,
        storageQuota: accountForm.storageQuota,
        passwordExpireTime: accountForm.passwordExpireTime
      })
      ElMessage.success(t('domains.accountUpdateSuccess'))
      accountDialogVisible.value = false
      await loadAccounts()
    } else {
      try {
        await accountApi.create({
          username: accountForm.username,
          domainId: selectedDomainId.value!,
          password: accountForm.password,
          storageQuota: accountForm.storageQuota,
          passwordExpireTime: accountForm.passwordExpireTime
        })
        ElMessage.success(t('domains.accountCreateSuccess'))
        accountDialogVisible.value = false
        await loadAccounts()
      } catch (error: any) {
        const errorMessage = error.response?.data?.message || error.message || t('domains.accountCreateFailed')
        ElMessage.error(errorMessage)
      }
    }

  } catch (error: any) {
    ElMessage.error(error.message || t('domains.operationFailed'))
  }
}

const handleChangePassword = async () => {
  if (!passwordFormRef.value) return

  try {
    const valid = await passwordFormRef.value.validate()
    if (!valid) return

    await accountApi.setPassword(passwordForm.accountId!, passwordForm.newPassword)
    ElMessage.success(t('profile.passwordUpdateSuccess'))
    passwordDialogVisible.value = false
  } catch (error: any) {
    const errorMessage = error.response?.data?.message || error.message || t('profile.passwordUpdateFailed')
    ElMessage.error(errorMessage)
  }
}

const handleAccountStatusChange = async (account: Account) => {
  try {
    await accountApi.update(account.id, { active: account.active })
    ElMessage.success(t('domains.statusUpdateSuccess'))
  } catch (error: any) {
    account.active = !account.active
    const errorMessage = error.response?.data?.message || error.message || t('domains.statusUpdateFailed')
    ElMessage.error(errorMessage)
  }
}

const handleDeleteAccount = async (id: number) => {
  try {
    await ElMessageBox.confirm(
      t('domains.deleteAccountConfirm'),
      t('domains.deleteConfirm'),
      {
        confirmButtonText: t('common.confirm'),
        cancelButtonText: t('common.cancel'),
        type: 'warning'
      }
    )

    await accountApi.delete(id)
    ElMessage.success(t('domains.accountDeleteSuccess'))
    await loadAccounts()
  } catch (error: any) {
    if (error !== 'cancel') {
      const errorMessage = error.response?.data?.message || error.message || t('domains.deleteFailed')
      ElMessage.error(errorMessage)
    }
  }
}

const handlePurgeAccount = async (account: Account) => {
  try {
    await ElMessageBox.confirm(
      t('domains.purgeAccountConfirm'),
      t('domains.purgeAccount'),
      {
        confirmButtonText: t('common.confirm'),
        cancelButtonText: t('common.cancel'),
        type: 'warning'
      }
    )

    await accountApi.purge(account.id)
    ElMessage.success(t('domains.purgeAccountSuccess'))
    await loadAccounts()
  } catch (error: any) {
    if (error !== 'cancel') {
      const errorMessage = error.response?.data?.message || error.message || t('domains.purgeAccountFailed')
      ElMessage.error(errorMessage)
    }
  }
}

const handleSizeChange = (size: number) => {
  domainPageSize.value = size
  domainCurrentPage.value = 1
  loadDomains()
}

const handleCurrentChange = (current: number) => {
  domainCurrentPage.value = current
  loadDomains()
}

const handleAccountSizeChange = (size: number) => {
  accountPageSize.value = size
  accountCurrentPage.value = 1
}

const handleAccountCurrentChange = (current: number) => {
  accountCurrentPage.value = current
}

onMounted(async () => {
  await loadDomains()
})

watch(domainSearchQuery, () => {
  domainCurrentPage.value = 1
  loadDomains()
})

watch(accountSearchQuery, () => {
  accountCurrentPage.value = 1
  loadAccounts()
})

watch(accountStatusFilter, () => {
  accountCurrentPage.value = 1
  loadAccounts()
})

const openDKIMDialog = (domain: Domain) => {
  dkimDomainId.value = domain.id
  dkimDomainName.value = domain.name
  Object.assign(dkimForm, {
    enabled: domain.dkimEnabled || false,
    selector: domain.dkimSelector || 'default',
    privateKey: domain.dkimPrivateKey || '',
    publicKey: ''
  })
  dkimDialogVisible.value = true
}

const handleGenerateDKIMKey = async () => {
  try {
    generatingKey.value = true
    // 使用简单的 RSA 密钥生成，实际项目应调用后端 API
    const { publicKey, privateKey } = await generateRSAKeyPair()
    dkimForm.privateKey = privateKey
    dkimForm.publicKey = publicKey
    ElMessage.success(t('domains.dkimGenerateKey') + ' - ' + t('common.success'))
  } catch (error: any) {
    ElMessage.error(error.message || t('domains.operationFailed'))
  } finally {
    generatingKey.value = false
  }
}

const handleSaveDKIM = async () => {
  if (!dkimFormRef.value || !dkimDomainId.value) return

  try {
    if (dkimForm.enabled) {
      const valid = await dkimFormRef.value.validate()
      if (!valid) return
    }

    savingDKIM.value = true
    await domainApi.updateDKIM(dkimDomainId.value, {
      enabled: dkimForm.enabled,
      selector: dkimForm.enabled ? dkimForm.selector : '',
      privateKey: dkimForm.enabled ? dkimForm.privateKey : ''
    })

    ElMessage.success(t('domains.dkimSaveSuccess'))
    dkimDialogVisible.value = false
    await loadDomains()
  } catch (error: any) {
    const errorMessage = error.response?.data?.message || error.message || t('domains.dkimSaveFailed')
    ElMessage.error(errorMessage)
  } finally {
    savingDKIM.value = false
  }
}

const handleCopyDNSRecord = async () => {
  try {
    const txtValue = `v=DKIM1; k=rsa; p=${dkimForm.publicKey}`
    await navigator.clipboard.writeText(txtValue)
    ElMessage.success(t('domains.dkimCopySuccess'))
  } catch {
    ElMessage.error(t('domains.copyFailed'))
  }
}

// 生成 RSA 密钥对（使用 Web Crypto API）
const generateRSAKeyPair = async (): Promise<{ publicKey: string; privateKey: string }> => {
  if (!window.crypto || !window.crypto.subtle) {
    throw new Error('Web Crypto API not available')
  }

  const keyPair = await window.crypto.subtle.generateKey(
    {
      name: 'RSA-OAEP',
      modulusLength: 2048,
      publicExponent: new Uint8Array([0x01, 0x00, 0x01]),
      hash: 'SHA-256',
    },
    true,
    ['encrypt', 'decrypt']
  )

  // 导出为 PEM 格式
  const publicKeyBuffer = await window.crypto.subtle.exportKey('spki', keyPair.publicKey)
  const privateKeyBuffer = await window.crypto.subtle.exportKey('pkcs8', keyPair.privateKey)

  const publicKeyBase64 = btoa(String.fromCharCode(...new Uint8Array(publicKeyBuffer)))
  const privateKeyBase64 = btoa(String.fromCharCode(...new Uint8Array(privateKeyBuffer)))

  // 对于 DKIM，我们需要 RSA 签名密钥而不是加密密钥
  // 这里简化处理，实际应使用 RSA-PSS 签名
  return {
    publicKey: publicKeyBase64,
    privateKey: `-----BEGIN PRIVATE KEY-----\n${privateKeyBase64.match(/.{1,64}/g)?.join('\n') || privateKeyBase64}\n-----END PRIVATE KEY-----`
  }
}
</script>

<style scoped>
.mail-system-container {
  height: 100%;
  margin: 0 auto;
}

.main-card {
  height: 100%;
  border: none;
  box-shadow: var(--shadow-card);
}

.main-card :deep(.el-card__header) {
  background: transparent;
  border-bottom: 1px solid var(--border-default);
  padding: 20px 25px;
}

.main-card :deep(.el-card__body) {
  padding: 25px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.header-icon {
  width: 40px;
  height: 40px;
  background: var(--accent-10);
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--accent);
}

.header-icon :deep(.el-icon) {
  font-size: 20px;
}

.card-header h2 {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: var(--foreground);
}

.domain-tabs {
  height: 100%;
}

.domain-tabs :deep(.el-tabs__content) {
  height: calc(100% - 40px);
  overflow: auto;
}

.domain-tabs :deep(.el-tab-pane) {
  height: 100%;
}

.domain-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.account-panel {
  flex: 1;
  min-width: 0;
  height: 100%;
  display: flex;
  flex-direction: column;
}

.domain-list {
  flex: 1;
  overflow-y: auto;
  padding: 15px;
}

.domain-list :deep(.selected-row) {
  background-color: var(--accent-10) !important;
}

.domain-list :deep(.selected-row:hover > td) {
  background-color: var(--accent-15) !important;
}

.domain-list :deep(.el-table__row) {
  cursor: pointer;
}

.account-list {
  flex: 1;
  padding: 15px;
  overflow-y: auto;
}

.pagination-container {
  display: flex;
  justify-content: center;
  padding: 15px 0;
}

.dialog-footer {
  display: flex;
  gap: 12px;
  justify-content: flex-end;
}

.search-box {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 15px;
}

.search-box-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.search-box-right {
  display: flex;
  align-items: center;
  gap: 12px;
}

.search-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 15px;
}

.search-bar-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.search-bar-right {
  display: flex;
  align-items: center;
  gap: 12px;
}

.empty-tip {
  text-align: center;
  color: var(--foreground-muted);
  padding: 40px 20px;
  font-size: 14px;
}

.domain-name-link:hover {
  text-decoration: underline;
}

:deep(.el-checkbox) {
  display: flex;
  align-items: center;
}

.dkim-dialog-content {
  max-height: 70vh;
  overflow-y: auto;
}

.mb-4 {
  margin-bottom: 16px;
}

.selector-input-group {
  display: flex;
  gap: 12px;
  width: 100%;
}

.selector-input-group .easy-input {
  flex: 1;
}

.dns-record-box {
  background: var(--surface) !important;
  border: 1px solid var(--border-default) !important;
  border-radius: 4px;
  padding: 12px;
  width: 100%;
}

.dns-record-hint {
  color: var(--foreground-muted, #909399);
  font-size: 12px;
  margin: 0 0 8px 0;
}

.dns-record-item {
  display: flex;
  align-items: flex-start;
  margin-bottom: 8px;
  gap: 8px;
}

.dns-record-item:last-child {
  margin-bottom: 0;
}

.dns-record-label {
  font-weight: 600;
  color: var(--foreground);
  white-space: nowrap;
  min-width: 60px;
}

.dns-record-value {
  background: var(--surface);
  border: 1px solid var(--border-default);
  border-radius: 4px;
  padding: 4px 8px;
  font-family: 'Courier New', monospace;
  font-size: 12px;
  word-break: break-all;
  color: var(--foreground);
}

.dns-record-value-block {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
}

.dns-record-value-block .dns-record-value {
  flex: 1;
}
</style>
