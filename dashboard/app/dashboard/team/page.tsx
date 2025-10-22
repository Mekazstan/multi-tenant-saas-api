"use client"

import { useState } from "react"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { RoleBadge } from "@/components/shared/role-badge"
import { InviteMemberModal } from "@/components/dashboard/invite-member-modal"
import { ConfirmDialog } from "@/components/shared/confirm-dialog"
import { useTeam } from "@/hooks/use-team"
import { useAuth } from "@/hooks/use-auth"
import { UserPlus, Trash2, X, CheckCircle2 } from "lucide-react"
import { format } from "date-fns"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"

export default function TeamPage() {
  const { user } = useAuth()
  const { members, pendingInvitations, isLoading, inviteMember, removeMember, updateRole, cancelInvitation } = useTeam()
  const [isInviteModalOpen, setIsInviteModalOpen] = useState(false)
  const [memberToRemove, setMemberToRemove] = useState<string | null>(null)
  const [invitationToCancel, setInvitationToCancel] = useState<string | null>(null)

  const isOwnerOrAdmin = user?.role === "owner" || user?.role === "admin"

  const handleRemoveMember = async () => {
    if (memberToRemove) {
      await removeMember(memberToRemove)
      setMemberToRemove(null)
    }
  }

  const handleCancelInvitation = async () => {
    if (invitationToCancel) {
      await cancelInvitation(invitationToCancel)
      setInvitationToCancel(null)
    }
  }

  const handleRoleChange = async (memberId: string, newRole: string) => {
    await updateRole(memberId, newRole as "admin" | "member")
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
          <h1 className="text-3xl font-bold tracking-tight">Team</h1>
          <p className="text-muted-foreground">Manage your team members and invitations</p>
        </div>
        {isOwnerOrAdmin && (
          <Button onClick={() => setIsInviteModalOpen(true)}>
            <UserPlus className="mr-2 h-4 w-4" />
            Invite Member
          </Button>
        )}
      </div>

      {/* Team Members */}
      <Card>
        <CardHeader>
          <CardTitle>Team Members ({members.length})</CardTitle>
          <CardDescription>People who have access to your organization</CardDescription>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Email</TableHead>
                <TableHead>Role</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Joined</TableHead>
                {isOwnerOrAdmin && <TableHead className="text-right">Actions</TableHead>}
              </TableRow>
            </TableHeader>
            <TableBody>
              {members.map((member) => (
                <TableRow key={member.id}>
                  <TableCell className="font-medium">{member.email}</TableCell>
                  <TableCell>
                    {user?.role === "owner" && member.role !== "owner" ? (
                      <Select value={member.role} onValueChange={(value) => handleRoleChange(member.id, value)}>
                        <SelectTrigger className="w-32">
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="member">Member</SelectItem>
                          <SelectItem value="admin">Admin</SelectItem>
                        </SelectContent>
                      </Select>
                    ) : (
                      <RoleBadge role={member.role} />
                    )}
                  </TableCell>
                  <TableCell>
                    {member.email_verified ? (
                      <div className="flex items-center text-success">
                        <CheckCircle2 className="mr-2 h-4 w-4" />
                        <span className="text-sm">Verified</span>
                      </div>
                    ) : (
                      <span className="text-sm text-muted-foreground">Pending verification</span>
                    )}
                  </TableCell>
                  <TableCell className="text-muted-foreground">
                    {format(new Date(member.created_at), "MMM dd, yyyy")}
                  </TableCell>
                  {isOwnerOrAdmin && (
                    <TableCell className="text-right">
                      {member.role !== "owner" && member.id !== user?.id && (
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => setMemberToRemove(member.id)}
                          className="text-destructive hover:text-destructive"
                        >
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      )}
                    </TableCell>
                  )}
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* Pending Invitations */}
      {pendingInvitations.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle>Pending Invitations ({pendingInvitations.length})</CardTitle>
            <CardDescription>Invitations that haven't been accepted yet</CardDescription>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Email</TableHead>
                  <TableHead>Role</TableHead>
                  <TableHead>Invited By</TableHead>
                  <TableHead>Expires</TableHead>
                  {isOwnerOrAdmin && <TableHead className="text-right">Actions</TableHead>}
                </TableRow>
              </TableHeader>
              <TableBody>
                {pendingInvitations.map((invitation) => (
                  <TableRow key={invitation.id}>
                    <TableCell className="font-medium">{invitation.email}</TableCell>
                    <TableCell>
                      <RoleBadge role={invitation.role} />
                    </TableCell>
                    <TableCell className="text-muted-foreground">{invitation.invited_by}</TableCell>
                    <TableCell className="text-muted-foreground">
                      {format(new Date(invitation.expires_at), "MMM dd, yyyy")}
                    </TableCell>
                    {isOwnerOrAdmin && (
                      <TableCell className="text-right">
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => setInvitationToCancel(invitation.id)}
                          className="text-destructive hover:text-destructive"
                        >
                          <X className="h-4 w-4" />
                        </Button>
                      </TableCell>
                    )}
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}

      {/* Team Roles Information */}
      <Card>
        <CardHeader>
          <CardTitle>Team Roles</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div>
            <h4 className="text-sm font-medium mb-1">Owner</h4>
            <p className="text-sm text-muted-foreground">
              Full access to all features including billing, team management, and organization settings. Can transfer
              ownership.
            </p>
          </div>
          <div>
            <h4 className="text-sm font-medium mb-1">Admin</h4>
            <p className="text-sm text-muted-foreground">
              Can manage team members, API keys, and view all data. Cannot access billing or delete the organization.
            </p>
          </div>
          <div>
            <h4 className="text-sm font-medium mb-1">Member</h4>
            <p className="text-sm text-muted-foreground">
              Can view data, use API keys, and access basic features. Cannot manage team or organization settings.
            </p>
          </div>
        </CardContent>
      </Card>

      {/* Invite Member Modal */}
      <InviteMemberModal open={isInviteModalOpen} onOpenChange={setIsInviteModalOpen} onInvite={inviteMember} />

      {/* Remove Member Confirmation */}
      <ConfirmDialog
        open={!!memberToRemove}
        onOpenChange={(open) => !open && setMemberToRemove(null)}
        title="Remove Team Member"
        description="Are you sure you want to remove this team member? They will lose access to the organization immediately."
        onConfirm={handleRemoveMember}
        confirmText="Remove Member"
        isDestructive
      />

      {/* Cancel Invitation Confirmation */}
      <ConfirmDialog
        open={!!invitationToCancel}
        onOpenChange={(open) => !open && setInvitationToCancel(null)}
        title="Cancel Invitation"
        description="Are you sure you want to cancel this invitation? The recipient will not be able to join using this invitation link."
        onConfirm={handleCancelInvitation}
        confirmText="Cancel Invitation"
        isDestructive
      />
    </div>
  )
}
