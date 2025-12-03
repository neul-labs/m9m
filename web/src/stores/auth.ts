import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import apiClient from '@/api/client'

interface User {
  id: string
  email: string
  firstName?: string
  lastName?: string
  role: string
}

interface LoginCredentials {
  email: string
  password: string
}

interface AuthResponse {
  token: string
  user: User
}

export const useAuthStore = defineStore('auth', () => {
  // State
  const user = ref<User | null>(null)
  const token = ref<string | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)

  // Getters
  const isAuthenticated = computed(() => !!token.value)
  const isAdmin = computed(() => user.value?.role === 'admin')
  const fullName = computed(() => {
    if (!user.value) return ''
    if (user.value.firstName && user.value.lastName) {
      return `${user.value.firstName} ${user.value.lastName}`
    }
    return user.value.email
  })

  // Actions
  function initAuth() {
    const savedToken = localStorage.getItem('auth_token')
    const savedUser = localStorage.getItem('auth_user')

    if (savedToken) {
      token.value = savedToken
    }

    if (savedUser) {
      try {
        user.value = JSON.parse(savedUser)
      } catch {
        // Invalid saved user, clear it
        localStorage.removeItem('auth_user')
      }
    }
  }

  async function login(credentials: LoginCredentials) {
    loading.value = true
    error.value = null
    try {
      const response = await apiClient.post<AuthResponse>('/auth/login', credentials)
      const { token: newToken, user: newUser } = response.data

      token.value = newToken
      user.value = newUser

      localStorage.setItem('auth_token', newToken)
      localStorage.setItem('auth_user', JSON.stringify(newUser))

      return newUser
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Login failed'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function register(data: LoginCredentials & { firstName?: string; lastName?: string }) {
    loading.value = true
    error.value = null
    try {
      const response = await apiClient.post<AuthResponse>('/auth/register', data)
      const { token: newToken, user: newUser } = response.data

      token.value = newToken
      user.value = newUser

      localStorage.setItem('auth_token', newToken)
      localStorage.setItem('auth_user', JSON.stringify(newUser))

      return newUser
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Registration failed'
      throw e
    } finally {
      loading.value = false
    }
  }

  function logout() {
    token.value = null
    user.value = null
    localStorage.removeItem('auth_token')
    localStorage.removeItem('auth_user')
  }

  async function fetchCurrentUser() {
    if (!token.value) return null

    loading.value = true
    error.value = null
    try {
      const response = await apiClient.get<User>('/auth/me')
      user.value = response.data
      localStorage.setItem('auth_user', JSON.stringify(response.data))
      return response.data
    } catch (e) {
      // Token might be invalid, clear auth
      logout()
      throw e
    } finally {
      loading.value = false
    }
  }

  async function updateProfile(updates: Partial<User>) {
    loading.value = true
    error.value = null
    try {
      const response = await apiClient.patch<User>('/auth/me', updates)
      user.value = response.data
      localStorage.setItem('auth_user', JSON.stringify(response.data))
      return response.data
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to update profile'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function changePassword(currentPassword: string, newPassword: string) {
    loading.value = true
    error.value = null
    try {
      await apiClient.post('/auth/change-password', { currentPassword, newPassword })
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to change password'
      throw e
    } finally {
      loading.value = false
    }
  }

  return {
    // State
    user,
    token,
    loading,
    error,

    // Getters
    isAuthenticated,
    isAdmin,
    fullName,

    // Actions
    initAuth,
    login,
    register,
    logout,
    fetchCurrentUser,
    updateProfile,
    changePassword,
  }
})
