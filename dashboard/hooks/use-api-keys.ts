"use client"

import { useState, useEffect } from "react"
import { api } from "@/lib/api"
import type { ApiKey } from "@/types/api"
import { toast } from "sonner"

export function useApiKeys() {
  const [keys, setKeys] = useState<ApiKey[]>([])
  const [isLoading, setIsLoading] = useState(true)

  const fetchKeys = async () => {
    setIsLoading(true)
    try {
      const response = await api.get<{ api_keys: ApiKey[] }>("/keys")
      if (response.success && response.data) {
        setKeys(response.data.api_keys)
      }
    } catch (error) {
      toast.error("Failed to load API keys")
    } finally {
      setIsLoading(false)
    }
  }

  const createKey = async (name: string): Promise<string | null> => {
    try {
      const response = await api.post<{ api_key: ApiKey & { full_key: string } }>("/keys", { name })
      if (response.success && response.data) {
        await fetchKeys()
        toast.success("API key created successfully")
        return response.data.api_key.full_key || response.data.api_key.key
      } else {
        toast.error(response.error?.message || "Failed to create API key")
        return null
      }
    } catch (error) {
      toast.error("Failed to create API key")
      return null
    }
  }

  const revokeKey = async (id: string): Promise<boolean> => {
    try {
      const response = await api.delete(`/keys/${id}`)
      if (response.success) {
        await fetchKeys()
        toast.success("API key revoked successfully")
        return true
      } else {
        toast.error(response.error?.message || "Failed to revoke API key")
        return false
      }
    } catch (error) {
      toast.error("Failed to revoke API key")
      return false
    }
  }

  useEffect(() => {
    fetchKeys()
  }, [])

  return {
    keys,
    isLoading,
    createKey,
    revokeKey,
    refetch: fetchKeys,
  }
}
