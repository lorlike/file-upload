export type User = {
  id: string
  username: string
  isAdmin: boolean
}

export type AuthResponse = {
  token: string
  expiresAt: string
  user: User
}

export type FileItem = {
  id: string
  originalName: string
  sizeBytes: number
  uploadedAt: string
  downloadUrl: string
}

export type AdminUser = {
  id: string
  username: string
  isAdmin: boolean
  createdAt: string
  fileCount: number
  totalSize: number
}

export type AdminUserFiles = {
  user: User
  items: FileItem[]
}

const API_BASE = import.meta.env.VITE_API_BASE_URL ?? '/api'

function authToken() {
  return localStorage.getItem('file-upload-token')
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers ?? {})
  const token = authToken()
  if (token) {
    headers.set('Authorization', `Bearer ${token}`)
  }

  if (init.body && !(init.body instanceof FormData)) {
    headers.set('Content-Type', 'application/json')
  }

  const response = await fetch(`${API_BASE}${path}`, {
    ...init,
    headers
  })

  const contentType = response.headers.get('content-type') ?? ''
  const payload = contentType.includes('application/json')
    ? await response.json()
    : await response.text()

  if (!response.ok) {
    const message = typeof payload === 'string' ? payload : payload?.message ?? '请求失败'
    throw new Error(message)
  }

  return payload as T
}

export async function register(username: string, password: string) {
  return request<AuthResponse>('/auth/register', {
    method: 'POST',
    body: JSON.stringify({ username, password })
  })
}

export async function login(username: string, password: string) {
  return request<AuthResponse>('/auth/login', {
    method: 'POST',
    body: JSON.stringify({ username, password })
  })
}

export async function me() {
  return request<User>('/me')
}

export async function listFiles() {
  return request<{ items: FileItem[] }>('/files')
}

export async function uploadFile(file: File) {
  const formData = new FormData()
  formData.append('file', file)

  return request<FileItem>('/files', {
    method: 'POST',
    body: formData
  })
}

export async function renameFile(fileId: string, originalName: string) {
  return request<FileItem>(`/files/${fileId}`, {
    method: 'PATCH',
    body: JSON.stringify({ originalName })
  })
}

export async function deleteFile(fileId: string) {
  return request<void>(`/files/${fileId}`, {
    method: 'DELETE'
  })
}

export async function listAdminUsers() {
  return request<{ items: AdminUser[] }>('/admin/users')
}

export async function listAdminUserFiles(userId: string) {
  return request<AdminUserFiles>(`/admin/users/${userId}/files`)
}

export async function deleteAdminUser(userId: string) {
  return request<void>(`/admin/users/${userId}`, {
    method: 'DELETE'
  })
}

export async function deleteAdminFile(fileId: string) {
  return request<void>(`/admin/files/${fileId}`, {
    method: 'DELETE'
  })
}

export async function downloadFile(file: FileItem) {
  const token = authToken()
  const response = await fetch(`${API_BASE}${file.downloadUrl}`, {
    headers: token ? { Authorization: `Bearer ${token}` } : undefined
  })

  if (!response.ok) {
    throw new Error('下载失败')
  }

  const blob = await response.blob()
  const url = window.URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = file.originalName
  anchor.click()
  window.setTimeout(() => window.URL.revokeObjectURL(url), 0)
}

export function clearToken() {
  localStorage.removeItem('file-upload-token')
}

export function saveSession(token: string) {
  localStorage.setItem('file-upload-token', token)
}

export function tokenExists() {
  return Boolean(authToken())
}
