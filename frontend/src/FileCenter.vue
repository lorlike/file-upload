<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  clearToken,
  deleteFile,
  downloadFile,
  listFiles,
  login,
  me,
  register,
  renameFile,
  saveSession,
  tokenExists,
  uploadFile,
  type FileItem,
  type User
} from './api'

type AuthMode = 'login' | 'register'

const authMode = ref<AuthMode>('login')
const authLoading = ref(false)
const pageLoading = ref(false)
const uploadLoading = ref(false)
const currentUser = ref<User | null>(null)
const fileItems = ref<FileItem[]>([])
const pickedFile = ref<File | null>(null)
const fileInput = ref<HTMLInputElement | null>(null)

const authForm = reactive({
  username: '',
  password: ''
})

const stats = computed(() => {
  const totalSize = fileItems.value.reduce((sum, item) => sum + item.sizeBytes, 0)
  return {
    fileCount: fileItems.value.length,
    totalSize
  }
})

function formatBytes(bytes: number) {
  if (bytes < 1024) {
    return `${bytes} B`
  }

  const units = ['KB', 'MB', 'GB', 'TB']
  let value = bytes / 1024
  let unit = units[0]
  for (let i = 1; i < units.length && value >= 1024; i += 1) {
    value /= 1024
    unit = units[i]
  }

  return `${value.toFixed(value >= 10 ? 1 : 2)} ${unit}`
}

function formatDate(value: string) {
  return new Date(value).toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}

async function loadSession() {
  if (!tokenExists()) {
    return
  }

  pageLoading.value = true
  try {
    currentUser.value = await me()
    await reloadFiles()
  } catch (error) {
    clearToken()
    currentUser.value = null
    ElMessage.error(error instanceof Error ? error.message : '登录已失效')
  } finally {
    pageLoading.value = false
  }
}

async function reloadFiles() {
  pageLoading.value = true
  try {
    const data = await listFiles()
    fileItems.value = data.items
  } finally {
    pageLoading.value = false
  }
}

async function submitAuth() {
  authLoading.value = true
  try {
    const action = authMode.value === 'login' ? login : register
    const response = await action(authForm.username, authForm.password)
    saveSession(response.token)
    currentUser.value = response.user
    await reloadFiles()
    ElMessage.success(authMode.value === 'login' ? '登录成功' : '注册成功')
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '操作失败')
  } finally {
    authLoading.value = false
  }
}

function logout() {
  clearToken()
  currentUser.value = null
  fileItems.value = []
  pickedFile.value = null
  if (fileInput.value) {
    fileInput.value.value = ''
  }
  ElMessage.success('已退出登录')
}

function onPickFile(event: Event) {
  const target = event.target as HTMLInputElement
  pickedFile.value = target.files?.[0] ?? null
}

async function submitUpload() {
  if (!pickedFile.value) {
    ElMessage.warning('先选择一个文件')
    return
  }

  uploadLoading.value = true
  try {
    await uploadFile(pickedFile.value)
    pickedFile.value = null
    if (fileInput.value) {
      fileInput.value.value = ''
    }
    await reloadFiles()
    ElMessage.success('上传成功')
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '上传失败')
  } finally {
    uploadLoading.value = false
  }
}

async function handleDownload(item: FileItem) {
  try {
    await downloadFile(item)
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '下载失败')
  }
}

async function handleRename(item: FileItem) {
  try {
    const { value } = await ElMessageBox.prompt('输入新的文件名', '重命名文件', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      inputValue: item.originalName,
      inputPlaceholder: '请输入新的文件名',
      inputValidator: (value: string) => value.trim().length > 0,
      inputErrorMessage: '文件名不能为空'
    })

    await renameFile(item.id, value.trim())
    await reloadFiles()
    ElMessage.success('文件名已更新')
  } catch (error) {
    if (error !== 'cancel' && error !== 'close') {
      ElMessage.error(error instanceof Error ? error.message : '重命名失败')
    }
  }
}

async function handleDelete(item: FileItem) {
  try {
    await ElMessageBox.confirm(`确认删除文件 "${item.originalName}" 吗？`, '删除文件', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning'
    })

    await deleteFile(item.id)
    await reloadFiles()
    ElMessage.success('文件已删除')
  } catch (error) {
    if (error !== 'cancel' && error !== 'close') {
      ElMessage.error(error instanceof Error ? error.message : '删除失败')
    }
  }
}

onMounted(() => {
  void loadSession()
})
</script>

<template>
  <div class="app-shell">
    <div class="ambient ambient-a"></div>
    <div class="ambient ambient-b"></div>

    <main class="workspace-shell">
      <section class="panel app-header">
        <div>
          <div class="eyebrow">文件中心</div>
          <h1>文件上传</h1>
        </div>
        <div class="session-pill" :class="{ active: currentUser }">
          <span v-if="currentUser">已登录</span>
          <span v-else>未登录</span>
        </div>
      </section>

      <section class="panel content-panel">
        <template v-if="!currentUser">
          <div class="panel-header">
            <div>
              <h2>登录或注册</h2>
              <p class="subtle">使用你的账户继续。</p>
            </div>
          </div>

          <el-tabs v-model="authMode" class="auth-tabs">
            <el-tab-pane label="登录" name="login" />
            <el-tab-pane label="注册" name="register" />
          </el-tabs>

          <el-form class="auth-form" label-position="top" @submit.prevent="submitAuth">
            <el-form-item label="用户名">
              <el-input v-model="authForm.username" placeholder="输入用户名" autocomplete="username" />
            </el-form-item>
            <el-form-item label="密码">
              <el-input
                v-model="authForm.password"
                type="password"
                placeholder="至少 6 位"
                show-password
                autocomplete="current-password"
              />
            </el-form-item>
            <el-button type="primary" :loading="authLoading" class="primary-button" @click="submitAuth">
              {{ authMode === 'login' ? '登录' : '注册' }}
            </el-button>
          </el-form>
        </template>

        <template v-else>
          <div class="panel-header">
            <div>
              <h2>{{ currentUser.username }}</h2>
              <p class="subtle">管理你上传的文件。</p>
            </div>
            <el-button text @click="logout">退出登录</el-button>
          </div>

          <div class="stats-row">
            <div class="stat-card">
              <span>文件数量</span>
              <strong>{{ stats.fileCount }}</strong>
            </div>
            <div class="stat-card">
              <span>总大小</span>
              <strong>{{ formatBytes(stats.totalSize) }}</strong>
            </div>
          </div>

          <div class="upload-card">
            <div class="upload-info">
              <strong>上传文件</strong>
              <p>选择文件后上传，文件会保存到你的账户下。</p>
            </div>
            <input ref="fileInput" type="file" class="file-input" @change="onPickFile" />
            <div class="upload-actions">
              <span class="picked-name">{{ pickedFile?.name ?? '尚未选择文件' }}</span>
              <el-button type="primary" :loading="uploadLoading" @click="submitUpload">上传</el-button>
            </div>
          </div>

          <el-table :data="fileItems" stripe class="files-table" v-loading="pageLoading">
            <el-table-column prop="originalName" label="文件名" min-width="180" />
            <el-table-column prop="sizeBytes" label="大小" width="120">
              <template #default="{ row }">
                {{ formatBytes(row.sizeBytes) }}
              </template>
            </el-table-column>
            <el-table-column prop="uploadedAt" label="上传时间" min-width="180">
              <template #default="{ row }">
                {{ formatDate(row.uploadedAt) }}
              </template>
            </el-table-column>
            <el-table-column label="操作" width="220" fixed="right">
              <template #default="{ row }">
                <el-button link type="primary" @click="handleDownload(row)">下载</el-button>
                <el-button link type="primary" @click="handleRename(row)">重命名</el-button>
                <el-button link type="danger" @click="handleDelete(row)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>

          <el-empty v-if="!pageLoading && fileItems.length === 0" description="暂无上传文件" />
        </template>
      </section>
    </main>
  </div>
</template>
