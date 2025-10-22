import { cn } from "@/lib/utils"

interface StatusBadgeProps {
  status: "active" | "inactive" | "paid" | "pending" | "overdue" | "verified"
  className?: string
}

export function StatusBadge({ status, className }: StatusBadgeProps) {
  const variants = {
    active: "bg-success/10 text-success border-success/20",
    inactive: "bg-muted text-muted-foreground border-border",
    paid: "bg-success/10 text-success border-success/20",
    pending: "bg-yellow-500/10 text-yellow-500 border-yellow-500/20",
    overdue: "bg-destructive/10 text-destructive border-destructive/20",
    verified: "bg-success/10 text-success border-success/20",
  }

  return (
    <span
      className={cn(
        "inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-medium",
        variants[status],
        className,
      )}
    >
      {status.charAt(0).toUpperCase() + status.slice(1)}
    </span>
  )
}
