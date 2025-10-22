"use client"

import { useState } from "react"
import { Check, Copy } from "lucide-react"
import { Button } from "@/components/ui/button"
import { toast } from "sonner"

interface CopyButtonProps {
  text: string
  label?: string
}

export function CopyButton({ text, label = "Copy" }: CopyButtonProps) {
  const [copied, setCopied] = useState(false)

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(text)
      setCopied(true)
      toast.success("Copied to clipboard")
      setTimeout(() => setCopied(false), 2000)
    } catch (error) {
      toast.error("Failed to copy")
    }
  }

  return (
    <Button variant="ghost" size="sm" onClick={handleCopy} className="h-8 px-2">
      {copied ? <Check className="h-4 w-4 text-success" /> : <Copy className="h-4 w-4" />}
      <span className="ml-2 text-xs">{copied ? "Copied" : label}</span>
    </Button>
  )
}
