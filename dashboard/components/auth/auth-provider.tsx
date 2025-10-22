"use client"

import type React from "react"

import { useEffect } from "react"
import { useAuth } from "@/hooks/use-auth"
import { usePathname, useRouter } from "next/navigation"

const publicRoutes = ["/login", "/register", "/forgot-password", "/reset-password", "/verify-email"]

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, isLoading, fetchUser } = useAuth()
  const pathname = usePathname()
  const router = useRouter()

  useEffect(() => {
    fetchUser()
  }, [fetchUser])

  useEffect(() => {
    if (!isLoading) {
      const isPublicRoute = publicRoutes.some((route) => pathname.startsWith(route))

      if (!isAuthenticated && !isPublicRoute) {
        router.push("/login")
      } else if (isAuthenticated && isPublicRoute) {
        router.push("/dashboard")
      }
    }
  }, [isAuthenticated, isLoading, pathname, router])

  if (isLoading) {
    return (
      <div className="flex h-screen items-center justify-center">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
      </div>
    )
  }

  return <>{children}</>
}
