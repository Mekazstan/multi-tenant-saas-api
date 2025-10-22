"use client"

import { useEffect, useState } from "react"
import { StatCard } from "@/components/dashboard/stat-card"
import { UsageChart } from "@/components/dashboard/usage-chart"
import { CopyButton } from "@/components/shared/copy-button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { Button } from "@/components/ui/button"
import { api } from "@/lib/api"
import type { DashboardStats, UsageGraphData, ApiKey } from "@/types/api"
import { Activity, Key, DollarSign, TrendingUp } from "lucide-react"
import { toast } from "sonner"
import Link from "next/link"

export default function DashboardPage() {
  const [stats, setStats] = useState<DashboardStats | null>(null)
  const [usageData, setUsageData] = useState<UsageGraphData | null>(null)
  const [recentKeys, setRecentKeys] = useState<ApiKey[]>([])
  const [isLoading, setIsLoading] = useState(true)

  useEffect(() => {
    const fetchData = async () => {
      try {
        const [statsRes, usageRes, keysRes] = await Promise.all([
          api.get<DashboardStats>("/dashboard/stats"),
          api.get<UsageGraphData>("/dashboard/usage-graph"),
          api.get<{ api_keys: ApiKey[] }>("/dashboard/api-keys"),
        ])

        if (statsRes.success && statsRes.data) {
          setStats(statsRes.data)
        }

        if (usageRes.success && usageRes.data) {
          setUsageData(usageRes.data)
        }

        if (keysRes.success && keysRes.data) {
          setRecentKeys(keysRes.data.api_keys)
        }
      } catch (error) {
        toast.error("Failed to load dashboard data")
      } finally {
        setIsLoading(false)
      }
    }

    fetchData()
  }, [])

  if (isLoading) {
    return (
      <div className="flex h-full items-center justify-center">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Dashboard</h1>
        <p className="text-muted-foreground">Welcome back! Here's an overview of your API usage.</p>
      </div>

      {/* Stats Grid */}
      <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-4">
        <StatCard
          title="Total Requests"
          value={stats?.stats.total_requests_30d.toLocaleString() || "0"}
          icon={Activity}
          description="Last 30 days"
        />
        <StatCard
          title="Active API Keys"
          value={stats?.stats.active_api_keys || 0}
          icon={Key}
          description={`${stats?.stats.total_api_keys || 0} total keys`}
        />
        <StatCard
          title="Current Month Cost"
          value={`$${stats?.stats.current_month_cost.toFixed(2) || "0.00"}`}
          icon={DollarSign}
          description={`${stats?.stats.current_month_requests.toLocaleString() || 0} requests`}
        />
        <StatCard
          title="Success Rate"
          value={`${stats?.stats.success_rate.toFixed(1) || "0"}%`}
          icon={TrendingUp}
          description="Last 30 days"
        />
      </div>

      {/* Usage Chart */}
      {usageData && <UsageChart data={usageData.data} />}

      {/* Recent API Keys */}
      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle>Recent API Keys</CardTitle>
          <Button asChild size="sm">
            <Link href="/dashboard/api-keys">View All</Link>
          </Button>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Key</TableHead>
                <TableHead>Last Used</TableHead>
                <TableHead className="text-right">Requests (30d)</TableHead>
                <TableHead className="text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {recentKeys.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={5} className="text-center text-muted-foreground">
                    No API keys found. Create your first key to get started.
                  </TableCell>
                </TableRow>
              ) : (
                recentKeys.slice(0, 5).map((key) => (
                  <TableRow key={key.id}>
                    <TableCell className="font-medium">{key.name}</TableCell>
                    <TableCell>
                      <code className="rounded bg-muted px-2 py-1 text-xs">{key.key}</code>
                    </TableCell>
                    <TableCell className="text-muted-foreground">
                      {key.last_used ? new Date(key.last_used).toLocaleDateString() : "Never"}
                    </TableCell>
                    <TableCell className="text-right">{key.requests_30d.toLocaleString()}</TableCell>
                    <TableCell className="text-right">
                      <CopyButton text={key.key} />
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  )
}
