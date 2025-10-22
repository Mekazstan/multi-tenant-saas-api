"use client"

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend } from "recharts"
import type { DailyBreakdown } from "@/types/api"
import { format } from "date-fns"

interface UsageBreakdownChartProps {
  data: DailyBreakdown[]
}

export function UsageBreakdownChart({ data }: UsageBreakdownChartProps) {
  const chartData = data.map((item) => ({
    date: format(new Date(item.date), "MMM dd"),
    success: item.success_count,
    errors: item.error_count,
  }))

  return (
    <Card>
      <CardHeader>
        <CardTitle>Daily Breakdown</CardTitle>
      </CardHeader>
      <CardContent>
        <ResponsiveContainer width="100%" height={300}>
          <BarChart data={chartData}>
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
            <Bar dataKey="success" fill="#10b981" name="Success" />
            <Bar dataKey="errors" fill="#ef4444" name="Errors" />
          </BarChart>
        </ResponsiveContainer>
      </CardContent>
    </Card>
  )
}
