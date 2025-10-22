"use client"

import { useState } from "react"
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"

interface InviteMemberModalProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onInvite: (email: string, role: "admin" | "member") => Promise<boolean>
}

export function InviteMemberModal({ open, onOpenChange, onInvite }: InviteMemberModalProps) {
  const [email, setEmail] = useState("")
  const [role, setRole] = useState<"admin" | "member">("member")
  const [isLoading, setIsLoading] = useState(false)

  const handleInvite = async () => {
    if (!email.trim()) return

    setIsLoading(true)
    const success = await onInvite(email, role)
    setIsLoading(false)

    if (success) {
      setEmail("")
      setRole("member")
      onOpenChange(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Invite Team Member</DialogTitle>
          <DialogDescription>Send an invitation to join your organization</DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="email">Email Address</Label>
            <Input
              id="email"
              type="email"
              placeholder="colleague@example.com"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              disabled={isLoading}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="role">Role</Label>
            <Select value={role} onValueChange={(value) => setRole(value as "admin" | "member")} disabled={isLoading}>
              <SelectTrigger id="role">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="member">Member</SelectItem>
                <SelectItem value="admin">Admin</SelectItem>
              </SelectContent>
            </Select>
            <p className="text-xs text-muted-foreground">
              {role === "admin"
                ? "Admins can manage team members and API keys"
                : "Members can view data and use API keys"}
            </p>
          </div>

          <div className="flex justify-end space-x-2">
            <Button variant="outline" onClick={() => onOpenChange(false)} disabled={isLoading}>
              Cancel
            </Button>
            <Button onClick={handleInvite} disabled={isLoading || !email.trim()}>
              {isLoading ? "Sending..." : "Send Invitation"}
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}
