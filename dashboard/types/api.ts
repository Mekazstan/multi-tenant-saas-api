// TypeScript interfaces for API responses

export interface User {
  id: string
  email: string
  role: "owner" | "admin" | "member"
  email_verified: boolean
  created_at: string
}

export interface Organization {
  id: string
  name: string
  plan: "free" | "starter" | "pro"
  created_at: string
}

export interface AuthResponse {
  token: string
  user: User
  organization: Organization
}

export interface DashboardStats {
  organization: {
    name: string
    plan: string
  }
  stats: {
    total_requests_30d: number
    current_month_requests: number
    current_month_cost: number
    success_rate: number
    active_api_keys: number
    total_api_keys: number
  }
}

export interface UsageDataPoint {
  date: string
  requests: number
  success_count: number
  error_count: number
}

export interface UsageGraphData {
  period: {
    start: string
    end: string
  }
  data: UsageDataPoint[]
}

export interface ApiKey {
  id: string
  name: string
  key: string
  requests_30d: number
  cost_30d: number
  last_used?: string
  status: "active" | "inactive"
  created_at: string
}

export interface TeamMember {
  id: string
  email: string
  role: "owner" | "admin" | "member"
  email_verified: boolean
  created_at: string
}

export interface PendingInvitation {
  id: string
  email: string
  role: "admin" | "member"
  invited_by: string
  expires_at: string
  created_at: string
}

export interface Invoice {
  id: string
  period: {
    start: string
    end: string
  }
  total_requests: number
  total_amount: number
  status: "paid" | "pending" | "overdue"
  created_at: string
  paid_at?: string
  due_date: string
}

export interface UsageSummary {
  total_requests: number
  successful_requests: number
  failed_requests: number
  success_rate: number
  total_cost: number
  period: {
    start: string
    end: string
  }
}

export interface EndpointUsage {
  endpoint: string
  requests: number
  success_count: number
  error_count: number
  cost: number
}

export interface ApiKeyUsage {
  id: string
  name: string
  key: string
  requests: number
  cost: number
}

export interface DailyBreakdown {
  date: string
  requests: number
  success_count: number
  error_count: number
  cost: number
}

export interface UsageData {
  summary: UsageSummary
  by_endpoint: EndpointUsage[]
  by_api_key: ApiKeyUsage[]
  daily_breakdown: DailyBreakdown[]
}

export interface BillingHistory {
  invoices: Invoice[]
  summary: {
    total_billed: number
    total_paid: number
    outstanding: number
  }
}
