"use client"

import { useState } from "react"
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Button } from "@/components/ui/button"
import { Label } from "@/components/ui/label"
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group"

interface PaymentModalProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  invoiceId: string
  amount: number
  onPay: (invoiceId: string, provider: "stripe" | "paystack") => Promise<void>
}

export function PaymentModal({ open, onOpenChange, invoiceId, amount, onPay }: PaymentModalProps) {
  const [provider, setProvider] = useState<"stripe" | "paystack">("stripe")
  const [isLoading, setIsLoading] = useState(false)

  const handlePay = async () => {
    setIsLoading(true)
    await onPay(invoiceId, provider)
    setIsLoading(false)
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Pay Invoice</DialogTitle>
          <DialogDescription>Select a payment provider to complete your payment</DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="rounded-lg bg-muted p-4">
            <p className="text-sm text-muted-foreground">Amount Due</p>
            <p className="text-2xl font-bold">${amount.toFixed(2)}</p>
          </div>

          <div className="space-y-2">
            <Label>Payment Provider</Label>
            <RadioGroup value={provider} onValueChange={(value) => setProvider(value as "stripe" | "paystack")}>
              <div className="flex items-center space-x-2 rounded-lg border border-border p-3">
                <RadioGroupItem value="stripe" id="stripe" />
                <Label htmlFor="stripe" className="flex-1 cursor-pointer font-normal">
                  <div className="flex items-center justify-between">
                    <span>Stripe</span>
                    <span className="text-xs text-muted-foreground">Credit/Debit Card</span>
                  </div>
                </Label>
              </div>
              <div className="flex items-center space-x-2 rounded-lg border border-border p-3">
                <RadioGroupItem value="paystack" id="paystack" />
                <Label htmlFor="paystack" className="flex-1 cursor-pointer font-normal">
                  <div className="flex items-center justify-between">
                    <span>Paystack</span>
                    <span className="text-xs text-muted-foreground">Multiple Options</span>
                  </div>
                </Label>
              </div>
            </RadioGroup>
          </div>

          <div className="flex justify-end space-x-2">
            <Button variant="outline" onClick={() => onOpenChange(false)} disabled={isLoading}>
              Cancel
            </Button>
            <Button onClick={handlePay} disabled={isLoading}>
              {isLoading ? "Processing..." : "Continue to Payment"}
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}
