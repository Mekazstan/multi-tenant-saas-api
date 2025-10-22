"use client"

import { useState } from "react"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { Button } from "@/components/ui/button"
import { StatCard } from "@/components/dashboard/stat-card"
import { StatusBadge } from "@/components/shared/status-badge"
import { DateRangePicker } from "@/components/shared/date-range-picker"
import { UsageBreakdownChart } from "@/components/dashboard/usage-breakdown-chart"
import { PaymentModal } from "@/components/dashboard/payment-modal"
import { useBilling } from "@/hooks/use-billing"
import { useAuth } from "@/hooks/use-auth"
import { Activity, DollarSign, TrendingUp, TrendingDown, CreditCard } from "lucide-react"
import { format } from "date-fns"
import type { DateRange } from "react-day-picker"

export default function BillingPage() {
  const { organization } = useAuth()
  const { usageData, billingHistory, isLoading, fetchUsage, initiatePayment } = useBilling()
  const [dateRange, setDateRange] = useState<DateRange | undefined>()
  const [paymentInvoice, setPaymentInvoice] = useState<{ id: string; amount: number } | null>(null)

  const handleDateRangeChange = (range: DateRange | undefined) => {
    setDateRange(range)
    if (range?.from && range?.to) {
      fetchUsage(format(range.from, "yyyy-MM-dd"), format(range.to, "yyyy-MM-dd"))
    } else {
      fetchUsage()
    }
  }

  const handlePayInvoice = async (invoiceId: string, provider: "stripe" | "paystack") => {
    const paymentUrl = await initiatePayment(invoiceId, provider)
    if (paymentUrl) {
      window.location.href = paymentUrl
    }
  }

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
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Billing & Usage</h1>
          <p className="text-muted-foreground">Monitor your API usage and manage billing</p>
        </div>
        <DateRangePicker value={dateRange} onChange={handleDateRangeChange} />
      </div>

      {/* Current Plan */}
      <Card>
        <CardHeader>
          <CardTitle>Current Plan</CardTitle>
          <CardDescription>Your subscription details</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-between">
            <div>
              <p className="text-2xl font-bold capitalize">{organization?.plan || "Free"} Plan</p>
              <p className="text-sm text-muted-foreground mt-1">
                {organization?.plan === "free" && "Limited to 1,000 requests per month"}
                {organization?.plan === "starter" && "Up to 10,000 requests per month"}
                {organization?.plan === "pro" && "Unlimited requests"}
              </p>
            </div>
            {organization?.plan === "free" && <Button>Upgrade Plan</Button>}
          </div>
        </CardContent>
      </Card>

      {/* Usage Summary */}
      <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-4">
        <StatCard
          title="Total Requests"
          value={usageData?.summary.total_requests.toLocaleString() || "0"}
          icon={Activity}
          description={`${usageData?.summary.period.start} - ${usageData?.summary.period.end}`}
        />
        <StatCard
          title="Successful Requests"
          value={usageData?.summary.successful_requests.toLocaleString() || "0"}
          icon={TrendingUp}
          description={`${usageData?.summary.success_rate.toFixed(1)}% success rate`}
        />
        <StatCard
          title="Failed Requests"
          value={usageData?.summary.failed_requests.toLocaleString() || "0"}
          icon={TrendingDown}
          description={`${(100 - (usageData?.summary.success_rate || 0)).toFixed(1)}% error rate`}
        />
        <StatCard
          title="Total Cost"
          value={`$${usageData?.summary.total_cost.toFixed(2) || "0.00"}`}
          icon={DollarSign}
          description="For selected period"
        />
      </div>

      {/* Daily Breakdown Chart */}
      {usageData?.daily_breakdown && <UsageBreakdownChart data={usageData.daily_breakdown} />}

      {/* Usage by Endpoint */}
      <Card>
        <CardHeader>
          <CardTitle>Usage by Endpoint</CardTitle>
          <CardDescription>API requests grouped by endpoint</CardDescription>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Endpoint</TableHead>
                <TableHead className="text-right">Requests</TableHead>
                <TableHead className="text-right">Success</TableHead>
                <TableHead className="text-right">Errors</TableHead>
                <TableHead className="text-right">Cost</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {usageData?.by_endpoint.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={5} className="text-center text-muted-foreground">
                    No usage data available
                  </TableCell>
                </TableRow>
              ) : (
                usageData?.by_endpoint.map((endpoint, index) => (
                  <TableRow key={index}>
                    <TableCell className="font-mono text-sm">{endpoint.endpoint}</TableCell>
                    <TableCell className="text-right">{endpoint.requests.toLocaleString()}</TableCell>
                    <TableCell className="text-right text-success">{endpoint.success_count.toLocaleString()}</TableCell>
                    <TableCell className="text-right text-destructive">
                      {endpoint.error_count.toLocaleString()}
                    </TableCell>
                    <TableCell className="text-right">${endpoint.cost.toFixed(2)}</TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* Usage by API Key */}
      <Card>
        <CardHeader>
          <CardTitle>Usage by API Key</CardTitle>
          <CardDescription>API requests grouped by key</CardDescription>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Key Name</TableHead>
                <TableHead>Key</TableHead>
                <TableHead className="text-right">Requests</TableHead>
                <TableHead className="text-right">Cost</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {usageData?.by_api_key.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={4} className="text-center text-muted-foreground">
                    No usage data available
                  </TableCell>
                </TableRow>
              ) : (
                usageData?.by_api_key.map((key) => (
                  <TableRow key={key.id}>
                    <TableCell className="font-medium">{key.name}</TableCell>
                    <TableCell>
                      <code className="rounded bg-muted px-2 py-1 text-xs">{key.key}</code>
                    </TableCell>
                    <TableCell className="text-right">{key.requests.toLocaleString()}</TableCell>
                    <TableCell className="text-right">${key.cost.toFixed(2)}</TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* Billing History */}
      <Card>
        <CardHeader>
          <CardTitle>Billing History</CardTitle>
          <CardDescription>Your past invoices and payments</CardDescription>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Invoice #</TableHead>
                <TableHead>Period</TableHead>
                <TableHead className="text-right">Requests</TableHead>
                <TableHead className="text-right">Amount</TableHead>
                <TableHead>Status</TableHead>
                <TableHead className="text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {billingHistory?.invoices.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={6} className="text-center text-muted-foreground">
                    No invoices yet
                  </TableCell>
                </TableRow>
              ) : (
                billingHistory?.invoices.map((invoice) => (
                  <TableRow key={invoice.id}>
                    <TableCell className="font-mono text-sm">{invoice.id.slice(0, 8)}</TableCell>
                    <TableCell className="text-muted-foreground">
                      {format(new Date(invoice.period.start), "MMM dd")} -{" "}
                      {format(new Date(invoice.period.end), "MMM dd, yyyy")}
                    </TableCell>
                    <TableCell className="text-right">{invoice.total_requests.toLocaleString()}</TableCell>
                    <TableCell className="text-right">${invoice.total_amount.toFixed(2)}</TableCell>
                    <TableCell>
                      <StatusBadge status={invoice.status} />
                    </TableCell>
                    <TableCell className="text-right">
                      {invoice.status === "pending" && (
                        <Button
                          size="sm"
                          variant="outline"
                          onClick={() => setPaymentInvoice({ id: invoice.id, amount: invoice.total_amount })}
                        >
                          <CreditCard className="mr-2 h-4 w-4" />
                          Pay
                        </Button>
                      )}
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>

          {billingHistory && billingHistory.invoices.length > 0 && (
            <div className="mt-6 flex justify-end space-x-8 border-t border-border pt-4">
              <div className="text-right">
                <p className="text-sm text-muted-foreground">Total Billed</p>
                <p className="text-lg font-semibold">${billingHistory.summary.total_billed.toFixed(2)}</p>
              </div>
              <div className="text-right">
                <p className="text-sm text-muted-foreground">Total Paid</p>
                <p className="text-lg font-semibold text-success">${billingHistory.summary.total_paid.toFixed(2)}</p>
              </div>
              <div className="text-right">
                <p className="text-sm text-muted-foreground">Outstanding</p>
                <p className="text-lg font-semibold text-destructive">
                  ${billingHistory.summary.outstanding.toFixed(2)}
                </p>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Payment Modal */}
      {paymentInvoice && (
        <PaymentModal
          open={!!paymentInvoice}
          onOpenChange={(open) => !open && setPaymentInvoice(null)}
          invoiceId={paymentInvoice.id}
          amount={paymentInvoice.amount}
          onPay={handlePayInvoice}
        />
      )}
    </div>
  )
}
