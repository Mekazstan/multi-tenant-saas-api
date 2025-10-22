"use client"

import { useState } from "react"
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { CopyButton } from "@/components/shared/copy-button"
import { AlertCircle } from "lucide-react"

interface CreateApiKeyModalProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onCreateKey: (name: string) => Promise<string | null>
}

export function CreateApiKeyModal({ open, onOpenChange, onCreateKey }: CreateApiKeyModalProps) {
  const [name, setName] = useState("")
  const [isLoading, setIsLoading] = useState(false)
  const [createdKey, setCreatedKey] = useState<string | null>(null)

  const handleCreate = async () => {
    if (!name.trim()) return

    setIsLoading(true)
    const key = await onCreateKey(name)
    setIsLoading(false)

    if (key) {
      setCreatedKey(key)
      setName("")
    }
  }

  const handleClose = () => {
    setCreatedKey(null)
    setName("")
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{createdKey ? "API Key Created" : "Create New API Key"}</DialogTitle>
          <DialogDescription>
            {createdKey
              ? "Save this key somewhere safe. You won't be able to see it again."
              : "Give your API key a descriptive name to help you identify it later."}
          </DialogDescription>
        </DialogHeader>

        {createdKey ? (
          <div className="space-y-4">
            <div className="rounded-lg border border-yellow-500/20 bg-yellow-500/10 p-4">
              <div className="flex items-start space-x-3">
                <AlertCircle className="h-5 w-5 text-yellow-500 mt-0.5" />
                <div className="flex-1 space-y-1">
                  <p className="text-sm font-medium text-yellow-500">Important</p>
                  <p className="text-xs text-muted-foreground">
                    Make sure to copy your API key now. You won't be able to see it again!
                  </p>
                </div>
              </div>
            </div>

            <div className="space-y-2">
              <Label>Your API Key</Label>
              <div className="flex items-center space-x-2">
                <code className="flex-1 rounded-lg bg-muted px-3 py-2 text-sm font-mono">{createdKey}</code>
                <CopyButton text={createdKey} />
              </div>
            </div>

            <Button onClick={handleClose} className="w-full">
              Done
            </Button>
          </div>
        ) : (
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="keyName">Key Name</Label>
              <Input
                id="keyName"
                placeholder="Production Key"
                value={name}
                onChange={(e) => setName(e.target.value)}
                disabled={isLoading}
              />
            </div>

            <div className="flex justify-end space-x-2">
              <Button variant="outline" onClick={handleClose} disabled={isLoading}>
                Cancel
              </Button>
              <Button onClick={handleCreate} disabled={isLoading || !name.trim()}>
                {isLoading ? "Creating..." : "Create Key"}
              </Button>
            </div>
          </div>
        )}
      </DialogContent>
    </Dialog>
  )
}
