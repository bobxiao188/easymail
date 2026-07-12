<template>
  <div class="profile-container">
    <el-card class="glass-card profile-card">
      <template #header>
        <div class="card-header">
          <div class="header-left">
            <div class="header-icon">
              <el-icon :size="18"><User /></el-icon>
            </div>
            <h2 class="easy-gradient-text">{{ t('profile.title') }}</h2>
          </div>
        </div>
      </template>
      
      <div class="profile-content">
        <!-- 头像区域 -->
        <div class="avatar-section">
          <div class="avatar-wrapper" @click="triggerAvatarPick">
            <input
              ref="avatarFileInputRef"
              type="file"
              class="avatar-file-input"
              accept="image/*"
              @change="onAvatarFileChange"
            />
            <div class="avatar-display easy-avatar" :class="{ 'is-image': showAvatarImage }">
              <img v-if="showAvatarImage" :src="userForm.avatar" alt="" class="avatar-img" />
              <template v-else>{{ userForm.nickname?.[0] || 'U' }}</template>
            </div>
            <div class="avatar-overlay">
              <el-icon :size="18"><Upload /></el-icon>
              <span>{{ t('profile.changeAvatar') }}</span>
            </div>
          </div>
        </div>
        
        <!-- 表单区域 -->
        <div class="form-section">
          <el-form :model="userForm" :rules="rules" ref="userFormRef" label-width="120px" class="easy-form">
            <el-form-item :label="t('profile.username')">
              <el-input v-model="userForm.username" disabled class="easy-input" />
            </el-form-item>
            <el-form-item :label="t('profile.nickname')" prop="nickname">
              <el-input v-model="userForm.nickname" class="easy-input" />
            </el-form-item>
            <el-form-item :label="t('profile.email')" prop="email">
              <el-input v-model="userForm.email" type="email" class="easy-input" />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" size="default" class="easy-button" @click="handleUpdate">
                {{ t('common.save') }}
              </el-button>
            </el-form-item>
          </el-form>
        </div>
      </div>
      
      <!-- 分隔线 -->
      <div class="easy-divider"></div>
      
      <!-- 修改密码区域 -->
      <div class="password-section">
        <div class="section-header">
          <div class="header-icon">
            <el-icon :size="18"><Lock /></el-icon>
          </div>
          <div class="section-title">{{ t('profile.changePassword') }}</div>
        </div>
        
        <el-form :model="passwordForm" :rules="passwordRules" ref="passwordFormRef" label-width="160px" class="easy-form">
          <el-form-item :label="t('profile.oldPassword')" prop="oldPassword">
            <el-input v-model="passwordForm.oldPassword" type="password" class="easy-input" show-password />
          </el-form-item>
          <el-form-item :label="t('profile.newPassword')" prop="newPassword">
            <el-input v-model="passwordForm.newPassword" type="password" class="easy-input" show-password />
          </el-form-item>
          <el-form-item :label="t('profile.confirmPassword')" prop="confirmPassword">
            <el-input v-model="passwordForm.confirmPassword" type="password" class="easy-input" show-password />
          </el-form-item>
          <el-form-item label-width="0" class="password-form-actions">
            <div class="password-form-actions-inner">
              <el-button type="primary" size="default" class="easy-button" @click="handleChangePassword">
                <span>{{ t('profile.changePassword') }}</span>
              </el-button>
            </div>
          </el-form-item>
        </el-form>
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { authApi } from '../api/auth'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '../stores'

const { t } = useI18n()
const authStore = useAuthStore()
const userFormRef = ref(null)
const passwordFormRef = ref(null)
const avatarFileInputRef = ref(null)

/** 是否为可展示的头像地址（后端存的是完整 Data URL 或 http(s)） */
const showAvatarImage = computed(() => {
  const a = userForm.avatar
  if (!a || typeof a !== 'string') return false
  return a.startsWith('data:image/') || a.startsWith('http://') || a.startsWith('https://')
})

const userForm = reactive({
  username: '',
  nickname: '',
  email: '',
  avatar: ''
})

const passwordForm = reactive({
  oldPassword: '',
  newPassword: '',
  confirmPassword: ''
})

const rules = {
  nickname: [
    { required: true, message: t('common.required', { field: t('profile.nickname') }), trigger: 'blur' }
  ],
  email: [
    { required: true, message: t('common.required', { field: t('profile.email') }), trigger: 'blur' },
    { type: 'email', message: t('profile.emailFormatError'), trigger: 'blur' }
  ]
}

const passwordRules = {
  oldPassword: [
    { required: true, message: t('common.required', { field: t('profile.oldPassword') }), trigger: 'blur' }
  ],
  newPassword: [
    { required: true, message: t('common.required', { field: t('profile.newPassword') }), trigger: 'blur' },
    { min: 6, message: t('profile.passwordMinLength'), trigger: 'blur' }
  ],
  confirmPassword: [
    { required: true, message: t('common.required', { field: t('profile.confirmPassword') }), trigger: 'blur' },
    {
      validator: (rule, value, callback) => {
        if (value !== passwordForm.newPassword) {
          callback(new Error(t('profile.passwordMismatch')))
        } else {
          callback()
        }
      },
      trigger: 'blur'
    }
  ]
}

function triggerAvatarPick () {
  avatarFileInputRef.value?.click()
}

/** 将所选图片读成完整 Data URL（与 FileReader 对 blob 的结果一致），写入 avatar 供提交入库 */
function onAvatarFileChange (e) {
  const input = e.target
  const file = input.files && input.files[0]
  input.value = ''
  if (!file) return
  if (!file.type.startsWith('image/')) {
    ElMessage.warning(t('profile.pleaseSelectImage'))
    return
  }
  const maxBytes = 8 * 1024 * 1024
  if (file.size > maxBytes) {
    ElMessage.warning(t('profile.imageTooLarge'))
    return
  }
  const reader = new FileReader()
  reader.onload = () => {
    if (typeof reader.result === 'string') {
      userForm.avatar = reader.result
    }
  }
  reader.onerror = () => {
    ElMessage.error(t('profile.imageReadFailed'))
  }
  reader.readAsDataURL(file)
}

onMounted(async () => {
  try {
    const res = await authStore.getProfile()
    if (res?.code === 0) {
      Object.assign(userForm, res.data)
    }
  } catch (error) {
    ElMessage.error(t('profile.getUserInfoFailed'))
  }
})

const handleUpdate = async () => {
  if (!userFormRef.value) return
  
  await userFormRef.value.validate(async (valid) => {
    if (valid) {
      try {
        await authApi.updateProfile(userForm)
        ElMessage.success(t('profile.updateSuccess'))
        await authStore.getProfile()
      } catch (error) {
        ElMessage.error(t('profile.updateFailed'))
      }
    }
  })
}

const handleChangePassword = async () => {
  if (!passwordFormRef.value) return
  
  await passwordFormRef.value.validate(async (valid) => {
    if (valid) {
      try {
        await authApi.changePassword({
          oldPassword: passwordForm.oldPassword,
          newPassword: passwordForm.newPassword
        })
        ElMessage.success(t('profile.passwordUpdateSuccess'))
        Object.keys(passwordForm).forEach(key => {
          passwordForm[key] = ''
        })
      } catch (error) {
        ElMessage.error(t('profile.passwordUpdateFailed'))
      }
    }
  })
}
</script>

<style scoped>
.profile-container {
  max-width: 900px;
  margin: 0 auto;
}

.profile-card {
  border: none;
  box-shadow: var(--shadow-card);
}

.profile-card :deep(.el-card__header) {
  background: transparent;
  border-bottom: 1px solid var(--border-default);
  padding: 25px 30px;
}

.profile-card :deep(.el-card__body) {
  padding: 30px;
}

/* 卡片头部 */
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
  border: 1px solid var(--accent-30);
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.header-icon svg {
  width: 20px;
  height: 20px;
  color: var(--accent);
}

.card-header h2 {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: var(--foreground);
}

/* 内容区域 */
.profile-content {
  display: grid;
  grid-template-columns: 1fr 2fr;
  gap: 40px;
  margin-bottom: 30px;
}

/* 头像区域 */
.avatar-section {
  display: flex;
  flex-direction: column;
  align-items: center;
}

.avatar-wrapper {
  position: relative;
  width: 150px;
  height: 150px;
  cursor: pointer;
  transition: transform 0.3s ease;
}

.avatar-file-input {
  position: absolute;
  width: 0;
  height: 0;
  opacity: 0;
  pointer-events: none;
}

.avatar-wrapper:hover {
  transform: scale(1.05);
}

.avatar-wrapper:hover .avatar-overlay {
  opacity: 1;
}

.avatar-display {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 48px;
  font-weight: 600;
  color: white;
  border-radius: 50%;
  border: 3px solid var(--accent-30);
  box-shadow: 0 0 30px var(--accent-glow);
  overflow: hidden;
}

.avatar-display.is-image {
  font-size: 0;
}

.avatar-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.avatar-overlay {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background: rgba(0, 0, 0, 0.7);
  border-radius: 50%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  opacity: 0;
  transition: opacity 0.3s ease;
  color: white;
  gap: 8px;
}

.avatar-overlay svg {
  width: 24px;
  height: 24px;
}

.avatar-overlay span {
  font-size: 12px;
  font-weight: 500;
}

/* 分隔线 */
.easy-divider {
  height: 1px;
  background: linear-gradient(90deg, transparent, var(--border-default), transparent);
  margin: 30px 0;
}

.section-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 20px;
}

.section-title {
  font-size: 16px;
  font-weight: 600;
}

/* 修改密码提交行：整行居中（Element Plus 内容在 .el-form-item__content 内，外层 flex 无效） */
.password-form-actions :deep(.el-form-item__content) {
  margin-left: 0 !important;
  margin-inline-start: 0 !important;
  max-width: 100%;
}

.password-form-actions-inner {
  display: flex;
  justify-content: center;
  width: 100%;
}

/* 响应式 */
@media (max-width: 768px) {
  .profile-content {
    grid-template-columns: 1fr;
    gap: 30px;
  }
  
  .avatar-section {
    order: -1;
  }
  
  .avatar-wrapper {
    width: 120px;
    height: 120px;
  }
  
  .avatar-display {
    font-size: 40px;
  }
}
</style>