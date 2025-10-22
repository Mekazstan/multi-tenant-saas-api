"use client"

import { useState, useEffect } from "react"
import { api } from "@/lib/api"
import type { TeamMember, PendingInvitation } from "@/types/api"
import { toast } from "sonner"

interface TeamData {
  members: TeamMember[]
  pending_invitations: PendingInvitation[]
  total_members: number
  total_pending: number
}

export function useTeam() {
  const [members, setMembers] = useState<TeamMember[]>([])
  const [pendingInvitations, setPendingInvitations] = useState<PendingInvitation[]>([])
  const [isLoading, setIsLoading] = useState(true)

  const fetchTeam = async () => {
    setIsLoading(true)
    try {
      const response = await api.get<TeamData>("/team/members")
      if (response.success && response.data) {
        setMembers(response.data.members)
        setPendingInvitations(response.data.pending_invitations)
      }
    } catch (error) {
      toast.error("Failed to load team members")
    } finally {
      setIsLoading(false)
    }
  }

  const inviteMember = async (email: string, role: "admin" | "member"): Promise<boolean> => {
    try {
      const response = await api.post("/team/invite", { email, role })
      if (response.success) {
        await fetchTeam()
        toast.success("Invitation sent successfully")
        return true
      } else {
        toast.error(response.error?.message || "Failed to send invitation")
        return false
      }
    } catch (error) {
      toast.error("Failed to send invitation")
      return false
    }
  }

  const removeMember = async (id: string): Promise<boolean> => {
    try {
      const response = await api.delete(`/team/members/${id}`)
      if (response.success) {
        await fetchTeam()
        toast.success("Member removed successfully")
        return true
      } else {
        toast.error(response.error?.message || "Failed to remove member")
        return false
      }
    } catch (error) {
      toast.error("Failed to remove member")
      return false
    }
  }

  const updateRole = async (id: string, role: "admin" | "member"): Promise<boolean> => {
    try {
      const response = await api.put(`/team/members/${id}/role`, { role })
      if (response.success) {
        await fetchTeam()
        toast.success("Role updated successfully")
        return true
      } else {
        toast.error(response.error?.message || "Failed to update role")
        return false
      }
    } catch (error) {
      toast.error("Failed to update role")
      return false
    }
  }

  const cancelInvitation = async (id: string): Promise<boolean> => {
    try {
      const response = await api.delete(`/team/invitations/${id}`)
      if (response.success) {
        await fetchTeam()
        toast.success("Invitation cancelled")
        return true
      } else {
        toast.error(response.error?.message || "Failed to cancel invitation")
        return false
      }
    } catch (error) {
      toast.error("Failed to cancel invitation")
      return false
    }
  }

  useEffect(() => {
    fetchTeam()
  }, [])

  return {
    members,
    pendingInvitations,
    isLoading,
    inviteMember,
    removeMember,
    updateRole,
    cancelInvitation,
    refetch: fetchTeam,
  }
}
