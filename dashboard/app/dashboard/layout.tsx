import type React from "react"
import { Sidebar } from "@/components/dashboard/sidebar"
import { Topbar } from "@/components/dashboard/topbar"

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <div className="flex h-screen overflow-hidden">
      <Sidebar />
      <div className="flex flex-1 flex-col overflow-hidden lg:pl-64">
        <Topbar />
        <main className="flex-1 overflow-y-auto bg-background p-6">{children}</main>
      </div>
    </div>
  )
}
