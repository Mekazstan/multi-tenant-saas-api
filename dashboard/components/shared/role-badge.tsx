import { cn } from "@/lib/utils"

interface RoleBadgeProps {
  role: "owner" | "admin" | "member"
  className?: string
}

export function RoleBadge({ role, className }: RoleBadgeProps) {
  const variants = {
    owner: "bg-primary/10 text-primary border-primary/20",
    admin: "bg-blue-500/10 text-blue-500 border-blue-500/20",
    member: "bg-muted text-muted-foreground border-border",
  }

  return (
    <span
      className={cn(
        "inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-medium",
        variants[role],
        className,
      )}
    >
      {role.charAt(0).toUpperCase() + role.slice(1)}
    </span>
  )
}
