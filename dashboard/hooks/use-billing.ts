"use client"

import { useState, useEffect } from "react"
import { api } from "@/lib/api"
import type { UsageData, BillingHistory } from "@/types/api"
import { toast } from "sonner"

export function useBilling() {
  const [usageData, setUsageData] = useState<UsageData | null>(null)
  const [billingHistory, setBillingHistory] = useState<BillingHistory | null>(null)
  const [isLoading, setIsLoading] = useState(true)

  const fetchUsage = async (startDate?: string, endDate?: string) => {
    setIsLoading(true)
    try {
      const params = new URLSearchParams()
      if (startDate) params.append("start_date", startDate)
      if (endDate) params.append("end_date", endDate)

      const response = await api.get<UsageData>(`/billing/usage?${params.toString()}`)
      if (response.success && response.data) {
        setUsageData(response.data)
      }
    } catch (error) {
      toast.error("Failed to load usage data")
    } finally {
      setIsLoading(false)
    }
  }

  const fetchBillingHistory = async () => {
    try {
      const response = await api.get<BillingHistory>("/billing/history")
      if (response.success && response.data) {
        setBillingHistory(response.data)
      }
    } catch (error) {
      toast.error("Failed to load billing history")
    }
  }

  const initiatePayment = async (billingCycleId: string, provider: "stripe" | "paystack"): Promise<string | null> => {
    try {
      const response = await api.post<{ payment_url: string }>("/billing/initiate-payment", {
        billing_cycle_id: billingCycleId,
        provider,
      })
      if (response.success && response.data) {
        return response.data.payment_url
      } else {
        toast.error(response.error?.message || "Failed to initiate payment")
        return null
      }
    } catch (error) {
      toast.error("Failed to initiate payment")
      return null
    }
  }

  useEffect(() => {
    fetchUsage()
    fetchBillingHistory()
  }, [])

  return {
    usageData,
    billingHistory,
    isLoading,
    fetchUsage,
    fetchBillingHistory,
    initiatePayment,
  }
}
