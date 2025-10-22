"use client"

import type React from "react"

import { useState } from "react"
import { useRouter } from "next/navigation"
import Link from "next/link"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Checkbox } from "@/components/ui/checkbox"
import { authService } from "@/lib/auth"
import { toast } from "sonner"

export default function RegisterPage() {
  const router = useRouter()
  const [isLoading, setIsLoading] = useState(false)
  const [formData, setFormData] = useState({
    organizationName: "",
    email: "",
    password: "",
    confirmPassword: "",
    acceptTerms: false,
  })

  const getPasswordStrength = (password: string) => {
    if (password.length === 0) return { strength: 0, label: "" }
    if (password.length < 8) return { strength: 25, label: "Weak" }

    let strength = 25
    if (/[a-z]/.test(password)) strength += 25
    if (/[A-Z]/.test(password)) strength += 25
    if (/[0-9]/.test(password)) strength += 25

    const label = strength <= 25 ? "Weak" : strength <= 50 ? "Fair" : strength <= 75 ? "Good" : "Strong"
    return { strength, label }
  }

  const passwordStrength = getPasswordStrength(formData.password)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    if (formData.password !== formData.confirmPassword) {
      toast.error("Passwords do not match")
      return
    }

    if (!formData.acceptTerms) {
      toast.error("Please accept the terms of service")
      return
    }

    setIsLoading(true)

    try {
      const response = await authService.register(formData.organizationName, formData.email, formData.password)

      if (response.success) {
        toast.success("Account created successfully!")
        router.push("/dashboard")
      } else {
        toast.error(response.error?.message || "Failed to create account")
      }
    } catch (error) {
      toast.error("An error occurred. Please try again.")
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center p-4">
      <div className="w-full max-w-md space-y-8">
        <div className="text-center">
          <h1 className="text-3xl font-bold tracking-tight">Create an account</h1>
          <p className="mt-2 text-sm text-muted-foreground">Get started with your free account</p>
        </div>

        <form onSubmit={handleSubmit} className="space-y-6">
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="organizationName">Organization Name</Label>
              <Input
                id="organizationName"
                type="text"
                placeholder="Acme Inc"
                value={formData.organizationName}
                onChange={(e) => setFormData({ ...formData, organizationName: e.target.value })}
                required
                disabled={isLoading}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="email">Email</Label>
              <Input
                id="email"
                type="email"
                placeholder="you@example.com"
                value={formData.email}
                onChange={(e) => setFormData({ ...formData, email: e.target.value })}
                required
                disabled={isLoading}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="password">Password</Label>
              <Input
                id="password"
                type="password"
                placeholder="••••••••"
                value={formData.password}
                onChange={(e) => setFormData({ ...formData, password: e.target.value })}
                required
                disabled={isLoading}
              />
              {formData.password && (
                <div className="space-y-1">
                  <div className="h-1 w-full bg-secondary rounded-full overflow-hidden">
                    <div
                      className={`h-full transition-all ${
                        passwordStrength.strength <= 25
                          ? "bg-destructive"
                          : passwordStrength.strength <= 50
                            ? "bg-orange-500"
                            : passwordStrength.strength <= 75
                              ? "bg-yellow-500"
                              : "bg-success"
                      }`}
                      style={{ width: `${passwordStrength.strength}%` }}
                    />
                  </div>
                  <p className="text-xs text-muted-foreground">Password strength: {passwordStrength.label}</p>
                </div>
              )}
            </div>

            <div className="space-y-2">
              <Label htmlFor="confirmPassword">Confirm Password</Label>
              <Input
                id="confirmPassword"
                type="password"
                placeholder="••••••••"
                value={formData.confirmPassword}
                onChange={(e) => setFormData({ ...formData, confirmPassword: e.target.value })}
                required
                disabled={isLoading}
              />
            </div>
          </div>

          <div className="flex items-start space-x-2">
            <Checkbox
              id="terms"
              checked={formData.acceptTerms}
              onCheckedChange={(checked) => setFormData({ ...formData, acceptTerms: checked as boolean })}
              disabled={isLoading}
            />
            <Label htmlFor="terms" className="text-sm font-normal cursor-pointer leading-relaxed">
              I agree to the{" "}
              <Link href="/terms" className="text-primary hover:underline">
                Terms of Service
              </Link>{" "}
              and{" "}
              <Link href="/privacy" className="text-primary hover:underline">
                Privacy Policy
              </Link>
            </Label>
          </div>

          <Button type="submit" className="w-full" disabled={isLoading}>
            {isLoading ? "Creating account..." : "Create account"}
          </Button>
        </form>

        <p className="text-center text-sm text-muted-foreground">
          Already have an account?{" "}
          <Link href="/login" className="text-primary hover:underline">
            Sign in
          </Link>
        </p>
      </div>
    </div>
  )
}
