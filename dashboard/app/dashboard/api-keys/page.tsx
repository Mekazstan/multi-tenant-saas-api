"use client"

import { useState } from "react"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { CopyButton } from "@/components/shared/copy-button"
import { StatusBadge } from "@/components/shared/status-badge"
import { CreateApiKeyModal } from "@/components/dashboard/create-api-key-modal"
import { ConfirmDialog } from "@/components/shared/confirm-dialog"
import { useApiKeys } from "@/hooks/use-api-keys"
import { Plus, Trash2 } from "lucide-react"
import { format } from "date-fns"

export default function ApiKeysPage() {
  const { keys, isLoading, createKey, revokeKey } = useApiKeys()
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false)
  const [keyToRevoke, setKeyToRevoke] = useState<string | null>(null)

  const handleRevokeKey = async () => {
    if (keyToRevoke) {
      await revokeKey(keyToRevoke)
      setKeyToRevoke(null)
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
          <h1 className="text-3xl font-bold tracking-tight">API Keys</h1>
          <p className="text-muted-foreground">Manage your API keys for authentication</p>
        </div>
        <Button onClick={() => setIsCreateModalOpen(true)}>
          <Plus className="mr-2 h-4 w-4" />
          Create API Key
        </Button>
      </div>

      {/* API Keys Table */}
      <Card>
        <CardHeader>
          <CardTitle>Your API Keys</CardTitle>
          <CardDescription>
            These keys allow you to authenticate with the API. Keep them secure and never share them publicly.
          </CardDescription>
        </CardHeader>
        <CardContent>
          {keys.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-12 text-center">
              <div className="rounded-full bg-muted p-4 mb-4">
                <Plus className="h-8 w-8 text-muted-foreground" />
              </div>
              <h3 className="text-lg font-semibold mb-2">No API keys yet</h3>
              <p className="text-sm text-muted-foreground mb-4">Create your first API key to start making requests</p>
              <Button onClick={() => setIsCreateModalOpen(true)}>
                <Plus className="mr-2 h-4 w-4" />
                Create API Key
              </Button>
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Key</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Created</TableHead>
                  <TableHead>Last Used</TableHead>
                  <TableHead className="text-right">Requests (30d)</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {keys.map((key) => (
                  <TableRow key={key.id}>
                    <TableCell className="font-medium">{key.name}</TableCell>
                    <TableCell>
                      <div className="flex items-center space-x-2">
                        <code className="rounded bg-muted px-2 py-1 text-xs font-mono">{key.key}</code>
                        <CopyButton text={key.key} />
                      </div>
                    </TableCell>
                    <TableCell>
                      <StatusBadge status={key.status} />
                    </TableCell>
                    <TableCell className="text-muted-foreground">
                      {format(new Date(key.created_at), "MMM dd, yyyy")}
                    </TableCell>
                    <TableCell className="text-muted-foreground">
                      {key.last_used ? format(new Date(key.last_used), "MMM dd, yyyy") : "Never"}
                    </TableCell>
                    <TableCell className="text-right">{key.requests_30d.toLocaleString()}</TableCell>
                    <TableCell className="text-right">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => setKeyToRevoke(key.id)}
                        className="text-destructive hover:text-destructive"
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      {/* Usage Information */}
      <Card>
        <CardHeader>
          <CardTitle>Using Your API Keys</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div>
            <h4 className="text-sm font-medium mb-2">Authentication</h4>
            <p className="text-sm text-muted-foreground mb-3">
              Include your API key in the Authorization header of your requests:
            </p>
            <pre className="rounded-lg bg-muted p-4 text-xs overflow-x-auto">
              <code>{`curl -X POST https://api.example.com/v1/endpoint \\
  -H "Authorization: Bearer YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"key": "value"}'`}</code>
            </pre>
          </div>

          <div>
            <h4 className="text-sm font-medium mb-2">Best Practices</h4>
            <ul className="space-y-2 text-sm text-muted-foreground">
              <li className="flex items-start">
                <span className="mr-2">•</span>
                <span>Never commit API keys to version control or share them publicly</span>
              </li>
              <li className="flex items-start">
                <span className="mr-2">•</span>
                <span>Use environment variables to store keys in your applications</span>
              </li>
              <li className="flex items-start">
                <span className="mr-2">•</span>
                <span>Rotate keys regularly and revoke unused keys</span>
              </li>
              <li className="flex items-start">
                <span className="mr-2">•</span>
                <span>Use different keys for development, staging, and production environments</span>
              </li>
            </ul>
          </div>
        </CardContent>
      </Card>

      {/* Create API Key Modal */}
      <CreateApiKeyModal open={isCreateModalOpen} onOpenChange={setIsCreateModalOpen} onCreateKey={createKey} />

      {/* Revoke Confirmation Dialog */}
      <ConfirmDialog
        open={!!keyToRevoke}
        onOpenChange={(open) => !open && setKeyToRevoke(null)}
        title="Revoke API Key"
        description="Are you sure you want to revoke this API key? This action cannot be undone and any applications using this key will stop working."
        onConfirm={handleRevokeKey}
        confirmText="Revoke Key"
        isDestructive
      />
    </div>
  )
}
