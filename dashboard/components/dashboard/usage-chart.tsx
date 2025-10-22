"use client"

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend } from "recharts"
import type { UsageDataPoint } from "@/types/api"
import { format } from "date-fns"

interface UsageChartProps {
  data: UsageDataPoint[]
}

export function UsageChart({ data }: UsageChartProps) {
  const chartData = data.map((item) => ({
    date: format(new Date(item.date), "MMM dd"),
    success: item.success_count,
    errors: item.error_count,
    total: item.requests,
  }))

  return (
    <Card>
      <CardHeader>
        <CardTitle>API Usage (Last 30 Days)</CardTitle>
      </CardHeader>
      <CardContent>
        <ResponsiveContainer width="100%" height={350}>
          <LineChart data={chartData}>
            <CartesianGrid strokeDasharray="3 3" stroke="#262626" />
            <XAxis dataKey="date" stroke="#a1a1a1" fontSize={12} tickLine={false} />
            <YAxis stroke="#a1a1a1" fontSize={12} tickLine={false} />
            <Tooltip
              contentStyle={{
                backgroundColor: "#0a0a0a",
                border: "1px solid #262626",
                borderRadius: "8px",
              }}
              labelStyle={{ color: "#ededed" }}
            />
            <Legend />
            <Line type="monotone" dataKey="success" stroke="#10b981" strokeWidth={2} dot={false} name="Success" />
            <Line type="monotone" dataKey="errors" stroke="#ef4444" strokeWidth={2} dot={false} name="Errors" />
          </LineChart>
        </ResponsiveContainer>
      </CardContent>
    </Card>
  )
}
