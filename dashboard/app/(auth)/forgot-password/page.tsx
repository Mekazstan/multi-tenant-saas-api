"use client"

import type React from "react"

import { useState } from "react"
import Link from "next/link"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { authService } from "@/lib/auth"
import { toast } from "sonner"
import { ArrowLeft, Mail } from "lucide-react"

export default function ForgotPasswordPage() {
  const [email, setEmail] = useState("")
  const [isLoading, setIsLoading] = useState(false)
  const [isSubmitted, setIsSubmitted] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setIsLoading(true)

    try {
      const response = await authService.requestPasswordReset(email)
      if (response.success) {
        setIsSubmitted(true)
        toast.success("Password reset email sent!")
      } else {
        toast.error(response.error?.message || "Failed to send reset email")
      }
    } catch (error) {
      toast.error("An error occurred. Please try again.")
    } finally {
      setIsLoading(false)
    }
  }

  if (isSubmitted) {
    return (
      <div className="flex min-h-screen items-center justify-center p-4">
        <div className="w-full max-w-md space-y-8 text-center">
          <div className="flex justify-center">
            <div className="rounded-full bg-primary/10 p-4">
              <Mail className="h-8 w-8 text-primary" />
            </div>
          </div>
          <div>
            <h1 className="text-3xl font-bold tracking-tight">Check your email</h1>
            <p className="mt-2 text-sm text-muted-foreground">
              We've sent a password reset link to <strong>{email}</strong>
            </p>
          </div>
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground">
              Didn't receive the email? Check your spam folder or try again.
            </p>
            <Button variant="outline" onClick={() => setIsSubmitted(false)} className="w-full">
              Try another email
            </Button>
          </div>
          <Link href="/login" className="inline-flex items-center text-sm text-primary hover:underline">
            <ArrowLeft className="mr-2 h-4 w-4" />
            Back to login
          </Link>
        </div>
      </div>
    )
  }

  return (
    <div className="flex min-h-screen items-center justify-center p-4">
      <div className="w-full max-w-md space-y-8">
        <div className="text-center">
          <h1 className="text-3xl font-bold tracking-tight">Forgot password?</h1>
          <p className="mt-2 text-sm text-muted-foreground">Enter your email and we'll send you a reset link</p>
        </div>

        <form onSubmit={handleSubmit} className="space-y-6">
          <div className="space-y-2">
            <Label htmlFor="email">Email</Label>
            <Input
              id="email"
              type="email"
              placeholder="you@example.com"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
              disabled={isLoading}
            />
          </div>

          <Button type="submit" className="w-full" disabled={isLoading}>
            {isLoading ? "Sending..." : "Send reset link"}
          </Button>
        </form>

        <Link href="/login" className="flex items-center justify-center text-sm text-primary hover:underline">
          <ArrowLeft className="mr-2 h-4 w-4" />
          Back to login
        </Link>
      </div>
    </div>
  )
}
