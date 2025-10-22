"use client"

import { create } from "zustand"
import type { User, Organization } from "@/types/api"
import { authService } from "@/lib/auth"

interface AuthState {
  user: User | null
  organization: Organization | null
  isLoading: boolean
  isAuthenticated: boolean
  setUser: (user: User | null) => void
  setOrganization: (organization: Organization | null) => void
  setLoading: (loading: boolean) => void
  login: (email: string, password: string) => Promise<boolean>
  logout: () => void
  fetchUser: () => Promise<void>
}

export const useAuth = create<AuthState>((set) => ({
  user: null,
  organization: null,
  isLoading: true,
  isAuthenticated: false,

  setUser: (user) => set({ user, isAuthenticated: !!user }),
  setOrganization: (organization) => set({ organization }),
  setLoading: (isLoading) => set({ isLoading }),

  login: async (email: string, password: string) => {
    const response = await authService.login(email, password)
    if (response.success && response.data) {
      set({
        user: response.data.user,
        organization: response.data.organization,
        isAuthenticated: true,
      })
      return true
    }
    return false
  },

  logout: () => {
    authService.logout()
    set({ user: null, organization: null, isAuthenticated: false })
  },

  fetchUser: async () => {
    set({ isLoading: true })
    const response = await authService.getCurrentUser()
    if (response.success && response.data) {
      set({
        user: response.data.user,
        isAuthenticated: true,
        isLoading: false,
      })
    } else {
      set({ user: null, isAuthenticated: false, isLoading: false })
    }
  },
}))
