// API Configuration and utilities
const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api/v1"

export interface ApiError {
  code: string
  message: string
  details?: {
    field?: string
    reason?: string
  }
}

export interface ApiResponse<T> {
  success: boolean
  data?: T
  error?: ApiError
}

export class ApiClient {
  private baseUrl: string

  constructor(baseUrl: string = API_BASE_URL) {
    this.baseUrl = baseUrl
  }

  private getAuthHeaders(): HeadersInit {
    const token = typeof window !== "undefined" ? localStorage.getItem("token") : null
    return {
      "Content-Type": "application/json",
      ...(token && { Authorization: `Bearer ${token}` }),
    }
  }

  async request<T>(endpoint: string, options: RequestInit = {}): Promise<ApiResponse<T>> {
    try {
      const response = await fetch(`${this.baseUrl}${endpoint}`, {
        ...options,
        headers: {
          ...this.getAuthHeaders(),
          ...options.headers,
        },
      })

      const data = await response.json()

      if (!response.ok) {
        // Handle specific error codes
        if (response.status === 401) {
          // Unauthorized - clear token and redirect to login
          if (typeof window !== "undefined") {
            localStorage.removeItem("token")
            window.location.href = "/login"
          }
        }
        return data
      }

      return data
    } catch (error) {
      return {
        success: false,
        error: {
          code: "NETWORK_ERROR",
          message: "Failed to connect to the server",
        },
      }
    }
  }

  async get<T>(endpoint: string): Promise<ApiResponse<T>> {
    return this.request<T>(endpoint, { method: "GET" })
  }

  async post<T>(endpoint: string, body?: unknown): Promise<ApiResponse<T>> {
    return this.request<T>(endpoint, {
      method: "POST",
      body: body ? JSON.stringify(body) : undefined,
    })
  }

  async put<T>(endpoint: string, body?: unknown): Promise<ApiResponse<T>> {
    return this.request<T>(endpoint, {
      method: "PUT",
      body: body ? JSON.stringify(body) : undefined,
    })
  }

  async delete<T>(endpoint: string): Promise<ApiResponse<T>> {
    return this.request<T>(endpoint, { method: "DELETE" })
  }
}

export const api = new ApiClient()
