import { api } from "./api"
import type { AuthResponse, User } from "@/types/api"

export const authService = {
  async login(email: string, password: string) {
    const response = await api.post<AuthResponse>("/auth/login", {
      email,
      password,
    })

    if (response.success && response.data) {
      localStorage.setItem("token", response.data.token)
    }

    return response
  },

  async register(organizationName: string, email: string, password: string) {
    const response = await api.post<AuthResponse>("/auth/register", {
      organization_name: organizationName,
      email,
      password,
    })

    if (response.success && response.data) {
      localStorage.setItem("token", response.data.token)
    }

    return response
  },

  async requestPasswordReset(email: string) {
    return api.post("/auth/request-password-reset", { email })
  },

  async resetPassword(token: string, newPassword: string) {
    return api.post("/auth/reset-password", {
      token,
      new_password: newPassword,
    })
  },

  async verifyEmail(token: string) {
    return api.post("/auth/verify-email", { token })
  },

  async getCurrentUser() {
    return api.get<{ user: User }>("/auth/me")
  },

  logout() {
    localStorage.removeItem("token")
    window.location.href = "/login"
  },

  isAuthenticated(): boolean {
    return !!localStorage.getItem("token")
  },
}
