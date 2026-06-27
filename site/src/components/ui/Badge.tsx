import * as React from "react"
import { cn } from "@/lib/utils"

export interface BadgeProps extends React.HTMLAttributes<HTMLDivElement> {
  variant?: 'default' | 'secondary' | 'destructive' | 'outline' | 'success' | 'warning'
}

function Badge({ className, variant = "default", ...props }: BadgeProps) {
  return (
    <div
      className={cn(
        "inline-flex items-center border px-2.5 py-0.5 text-xs font-semibold font-mono transition-colors focus:outline-none focus:ring-2 focus:ring-zinc-400 focus:ring-offset-2",
        {
          "border-transparent bg-zinc-100 text-zinc-900 shadow": variant === "default",
          "border-transparent bg-zinc-800 text-zinc-300": variant === "secondary",
          "border-transparent bg-red-950/40 text-red-400 border border-red-900/50": variant === "destructive",
          "border-zinc-800 text-zinc-300": variant === "outline",
          "border-transparent bg-emerald-950/40 text-emerald-400 border border-emerald-900/50": variant === "success",
          "border-transparent bg-amber-950/40 text-amber-400 border border-amber-900/50": variant === "warning",
        },
        className
      )}
      {...props}
    />
  )
}

export { Badge }
