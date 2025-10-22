"use client"

import { useEffect, useState } from "react"
import { useRouter, useSearchParams } from "next/navigation"
import Link from "next/link"
import { Button } from "@/components/ui/button"
import { authService } from "@/lib/auth"
import { CheckCircle2, XCircle, Loader2 } from "lucide-react"

export default function VerifyEmailPage() {
  const router = useRouter()
  const searchParams = useSearchParams()
  const token = searchParams.get("token")

  const [status, setStatus] = useState<"loading" | "success" | "error">("loading")
  const [message, setMessage] = useState("")

  useEffect(() => {
    const verifyEmail = async () => {
      if (!token) {
        setStatus("error")
        setMessage("Invalid verification link")
        return
      }

      try {
        const response = await authService.verifyEmail(token)
        if (response.success) {
          setStatus("success")
          setMessage("Your email has been verified successfully!")
        } else {
          setStatus("error")
          setMessage(response.error?.message || "Verification failed")
        }
      } catch (error) {
        setStatus("error")
        setMessage("An error occurred during verification")
      }
    }

    verifyEmail()
  }, [token])

  return (
    <div className="flex min-h-screen items-center justify-center p-4">
      <div className="w-full max-w-md space-y-8 text-center">
        <div className="flex justify-center">
          {status === "loading" && <Loader2 className="h-16 w-16 animate-spin text-primary" />}
          {status === "success" && <CheckCircle2 className="h-16 w-16 text-success" />}
          {status === "error" && <XCircle className="h-16 w-16 text-destructive" />}
        </div>

        <div>
          <h1 className="text-3xl font-bold tracking-tight">
            {status === "loading" && "Verifying your email..."}
            {status === "success" && "Email verified!"}
            {status === "error" && "Verification failed"}
          </h1>
          <p className="mt-2 text-sm text-muted-foreground">{message}</p>
        </div>

        {status !== "loading" && (
          <div className="space-y-4">
            <Button onClick={() => router.push(status === "success" ? "/dashboard" : "/login")} className="w-full">
              {status === "success" ? "Go to Dashboard" : "Back to Login"}
            </Button>
            {status === "error" && (
              <Link href="/forgot-password" className="block text-sm text-primary hover:underline">
                Request new verification email
              </Link>
            )}
          </div>
        )}
      </div>
    </div>
  )
}
