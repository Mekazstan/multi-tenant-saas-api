"use client"

import type React from "react"

import { useState } from "react"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { useAuth } from "@/hooks/use-auth"
import { api } from "@/lib/api"
import { toast } from "sonner"
import { CheckCircle2, Mail, CreditCard, Crown } from "lucide-react"
import { Badge } from "@/components/ui/badge"

export default function SettingsPage() {
  const { user, organization } = useAuth()
  const [isLoading, setIsLoading] = useState(false)
  const [passwordData, setPasswordData] = useState({
    currentPassword: "",
    newPassword: "",
    confirmPassword: "",
  })

  const handleRequestVerification = async () => {
    setIsLoading(true)
    try {
      const response = await api.post("/auth/request-verification")
      if (response.success) {
        toast.success("Verification email sent! Check your inbox.")
      } else {
        toast.error(response.error?.message || "Failed to send verification email")
      }
    } catch (error) {
      toast.error("Failed to send verification email")
    } finally {
      setIsLoading(false)
    }
  }

  const handleChangePassword = async (e: React.FormEvent) => {
    e.preventDefault()

    if (passwordData.newPassword !== passwordData.confirmPassword) {
      toast.error("New passwords do not match")
      return
    }

    if (passwordData.newPassword.length < 8) {
      toast.error("Password must be at least 8 characters")
      return
    }

    setIsLoading(true)
    try {
      const response = await api.post("/auth/change-password", {
        current_password: passwordData.currentPassword,
        new_password: passwordData.newPassword,
      })

      if (response.success) {
        toast.success("Password changed successfully")
        setPasswordData({ currentPassword: "", newPassword: "", confirmPassword: "" })
      } else {
        toast.error(response.error?.message || "Failed to change password")
      }
    } catch (error) {
      toast.error("Failed to change password")
    } finally {
      setIsLoading(false)
    }
  }

  const handleUpgradePlan = async (plan: "starter" | "pro") => {
    setIsLoading(true)
    try {
      const response = await api.post("/billing/upgrade", { plan })
      if (response.success) {
        toast.success(`Successfully upgraded to ${plan} plan!`)
        window.location.reload()
      } else {
        toast.error(response.error?.message || "Failed to upgrade plan")
      }
    } catch (error) {
      toast.error("Failed to upgrade plan")
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Settings</h1>
        <p className="text-muted-foreground">Manage your account and organization settings</p>
      </div>

      {/* Settings Tabs */}
      <Tabs defaultValue="profile" className="space-y-6">
        <TabsList>
          <TabsTrigger value="profile">Profile</TabsTrigger>
          <TabsTrigger value="organization">Organization</TabsTrigger>
          <TabsTrigger value="plan">Plan & Billing</TabsTrigger>
        </TabsList>

        {/* Profile Tab */}
        <TabsContent value="profile" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>Profile Information</CardTitle>
              <CardDescription>Your account details and verification status</CardDescription>
            </CardHeader>
            <CardContent className="space-y-6">
              <div className="space-y-2">
                <Label>Email Address</Label>
                <div className="flex items-center space-x-2">
                  <Input value={user?.email || ""} disabled className="flex-1" />
                  {user?.email_verified ? (
                    <Badge variant="outline" className="bg-success/10 text-success border-success/20">
                      <CheckCircle2 className="mr-1 h-3 w-3" />
                      Verified
                    </Badge>
                  ) : (
                    <Badge variant="outline" className="bg-yellow-500/10 text-yellow-500 border-yellow-500/20">
                      Unverified
                    </Badge>
                  )}
                </div>
              </div>

              {!user?.email_verified && (
                <div className="rounded-lg border border-yellow-500/20 bg-yellow-500/10 p-4">
                  <div className="flex items-start space-x-3">
                    <Mail className="h-5 w-5 text-yellow-500 mt-0.5" />
                    <div className="flex-1">
                      <p className="text-sm font-medium text-yellow-500">Email not verified</p>
                      <p className="text-xs text-muted-foreground mt-1">
                        Please verify your email address to access all features
                      </p>
                      <Button
                        size="sm"
                        variant="outline"
                        className="mt-3 bg-transparent"
                        onClick={handleRequestVerification}
                        disabled={isLoading}
                      >
                        {isLoading ? "Sending..." : "Send Verification Email"}
                      </Button>
                    </div>
                  </div>
                </div>
              )}

              <div className="space-y-2">
                <Label>Role</Label>
                <Input value={user?.role.charAt(0).toUpperCase() + user?.role.slice(1) || ""} disabled />
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Change Password</CardTitle>
              <CardDescription>Update your password to keep your account secure</CardDescription>
            </CardHeader>
            <CardContent>
              <form onSubmit={handleChangePassword} className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="currentPassword">Current Password</Label>
                  <Input
                    id="currentPassword"
                    type="password"
                    value={passwordData.currentPassword}
                    onChange={(e) => setPasswordData({ ...passwordData, currentPassword: e.target.value })}
                    disabled={isLoading}
                    required
                  />
                </div>

                <div className="space-y-2">
                  <Label htmlFor="newPassword">New Password</Label>
                  <Input
                    id="newPassword"
                    type="password"
                    value={passwordData.newPassword}
                    onChange={(e) => setPasswordData({ ...passwordData, newPassword: e.target.value })}
                    disabled={isLoading}
                    required
                  />
                </div>

                <div className="space-y-2">
                  <Label htmlFor="confirmPassword">Confirm New Password</Label>
                  <Input
                    id="confirmPassword"
                    type="password"
                    value={passwordData.confirmPassword}
                    onChange={(e) => setPasswordData({ ...passwordData, confirmPassword: e.target.value })}
                    disabled={isLoading}
                    required
                  />
                </div>

                <Button type="submit" disabled={isLoading}>
                  {isLoading ? "Updating..." : "Update Password"}
                </Button>
              </form>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Organization Tab */}
        <TabsContent value="organization" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>Organization Details</CardTitle>
              <CardDescription>Information about your organization</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label>Organization Name</Label>
                <Input value={organization?.name || ""} disabled />
              </div>

              <div className="space-y-2">
                <Label>Organization ID</Label>
                <Input value={organization?.id || ""} disabled className="font-mono text-sm" />
              </div>

              <div className="space-y-2">
                <Label>Current Plan</Label>
                <div className="flex items-center space-x-2">
                  <Input value={organization?.plan.toUpperCase() || ""} disabled className="flex-1" />
                  <Badge variant="outline" className="bg-primary/10 text-primary border-primary/20">
                    <Crown className="mr-1 h-3 w-3" />
                    {organization?.plan}
                  </Badge>
                </div>
              </div>

              <div className="space-y-2">
                <Label>Created</Label>
                <Input
                  value={organization?.created_at ? new Date(organization.created_at).toLocaleDateString() : ""}
                  disabled
                />
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Danger Zone</CardTitle>
              <CardDescription>Irreversible actions for your organization</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="rounded-lg border border-destructive/20 bg-destructive/10 p-4">
                <div className="flex items-start justify-between">
                  <div>
                    <p className="text-sm font-medium text-destructive">Delete Organization</p>
                    <p className="text-xs text-muted-foreground mt-1">
                      Permanently delete your organization and all associated data
                    </p>
                  </div>
                  <Button variant="destructive" size="sm" disabled>
                    Delete
                  </Button>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Plan & Billing Tab */}
        <TabsContent value="plan" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>Current Plan</CardTitle>
              <CardDescription>Your active subscription</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="flex items-center justify-between rounded-lg border border-border p-4">
                <div>
                  <p className="text-lg font-semibold capitalize">{organization?.plan} Plan</p>
                  <p className="text-sm text-muted-foreground">
                    {organization?.plan === "free" && "Limited to 1,000 requests per month"}
                    {organization?.plan === "starter" && "Up to 10,000 requests per month - $29/month"}
                    {organization?.plan === "pro" && "Unlimited requests - $99/month"}
                  </p>
                </div>
                <Badge variant="outline" className="bg-primary/10 text-primary border-primary/20">
                  Active
                </Badge>
              </div>
            </CardContent>
          </Card>

          <div className="grid gap-6 md:grid-cols-3">
            {/* Free Plan */}
            <Card className={organization?.plan === "free" ? "border-primary" : ""}>
              <CardHeader>
                <CardTitle>Free</CardTitle>
                <CardDescription>Perfect for testing</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div>
                  <p className="text-3xl font-bold">$0</p>
                  <p className="text-sm text-muted-foreground">per month</p>
                </div>
                <ul className="space-y-2 text-sm">
                  <li className="flex items-start">
                    <CheckCircle2 className="mr-2 h-4 w-4 text-success mt-0.5" />
                    <span>1,000 requests/month</span>
                  </li>
                  <li className="flex items-start">
                    <CheckCircle2 className="mr-2 h-4 w-4 text-success mt-0.5" />
                    <span>Basic support</span>
                  </li>
                  <li className="flex items-start">
                    <CheckCircle2 className="mr-2 h-4 w-4 text-success mt-0.5" />
                    <span>1 team member</span>
                  </li>
                </ul>
                <Button className="w-full" disabled={organization?.plan === "free"}>
                  {organization?.plan === "free" ? "Current Plan" : "Downgrade"}
                </Button>
              </CardContent>
            </Card>

            {/* Starter Plan */}
            <Card className={organization?.plan === "starter" ? "border-primary" : ""}>
              <CardHeader>
                <CardTitle>Starter</CardTitle>
                <CardDescription>For growing projects</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div>
                  <p className="text-3xl font-bold">$29</p>
                  <p className="text-sm text-muted-foreground">per month</p>
                </div>
                <ul className="space-y-2 text-sm">
                  <li className="flex items-start">
                    <CheckCircle2 className="mr-2 h-4 w-4 text-success mt-0.5" />
                    <span>10,000 requests/month</span>
                  </li>
                  <li className="flex items-start">
                    <CheckCircle2 className="mr-2 h-4 w-4 text-success mt-0.5" />
                    <span>Priority support</span>
                  </li>
                  <li className="flex items-start">
                    <CheckCircle2 className="mr-2 h-4 w-4 text-success mt-0.5" />
                    <span>5 team members</span>
                  </li>
                  <li className="flex items-start">
                    <CheckCircle2 className="mr-2 h-4 w-4 text-success mt-0.5" />
                    <span>Advanced analytics</span>
                  </li>
                </ul>
                <Button
                  className="w-full"
                  disabled={organization?.plan === "starter" || isLoading}
                  onClick={() => handleUpgradePlan("starter")}
                >
                  {organization?.plan === "starter" ? "Current Plan" : "Upgrade"}
                </Button>
              </CardContent>
            </Card>

            {/* Pro Plan */}
            <Card className={organization?.plan === "pro" ? "border-primary" : ""}>
              <CardHeader>
                <CardTitle>Pro</CardTitle>
                <CardDescription>For production apps</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div>
                  <p className="text-3xl font-bold">$99</p>
                  <p className="text-sm text-muted-foreground">per month</p>
                </div>
                <ul className="space-y-2 text-sm">
                  <li className="flex items-start">
                    <CheckCircle2 className="mr-2 h-4 w-4 text-success mt-0.5" />
                    <span>Unlimited requests</span>
                  </li>
                  <li className="flex items-start">
                    <CheckCircle2 className="mr-2 h-4 w-4 text-success mt-0.5" />
                    <span>24/7 support</span>
                  </li>
                  <li className="flex items-start">
                    <CheckCircle2 className="mr-2 h-4 w-4 text-success mt-0.5" />
                    <span>Unlimited team members</span>
                  </li>
                  <li className="flex items-start">
                    <CheckCircle2 className="mr-2 h-4 w-4 text-success mt-0.5" />
                    <span>Custom integrations</span>
                  </li>
                  <li className="flex items-start">
                    <CheckCircle2 className="mr-2 h-4 w-4 text-success mt-0.5" />
                    <span>SLA guarantee</span>
                  </li>
                </ul>
                <Button
                  className="w-full"
                  disabled={organization?.plan === "pro" || isLoading}
                  onClick={() => handleUpgradePlan("pro")}
                >
                  {organization?.plan === "pro" ? "Current Plan" : "Upgrade"}
                </Button>
              </CardContent>
            </Card>
          </div>

          <Card>
            <CardHeader>
              <CardTitle>Billing Information</CardTitle>
              <CardDescription>Manage your payment methods and billing details</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="flex items-center justify-between rounded-lg border border-border p-4">
                <div className="flex items-center space-x-3">
                  <div className="rounded-lg bg-muted p-2">
                    <CreditCard className="h-5 w-5" />
                  </div>
                  <div>
                    <p className="text-sm font-medium">No payment method on file</p>
                    <p className="text-xs text-muted-foreground">Add a payment method to upgrade your plan</p>
                  </div>
                </div>
                <Button variant="outline" size="sm">
                  Add Payment Method
                </Button>
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}
