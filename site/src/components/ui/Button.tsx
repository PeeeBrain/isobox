import * as React from "react"
import { cn } from "@/lib/utils"

export interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'default' | 'destructive' | 'outline' | 'secondary' | 'ghost' | 'link' | 'terminal'
  size?: 'default' | 'sm' | 'lg' | 'icon'
}

const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant = "default", size = "default", ...props }, ref) => {
    return (
      <button
        className={cn(
          "inline-flex items-center justify-center whitespace-nowrap text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-zinc-400 disabled:pointer-events-none disabled:opacity-50",
          {
            "bg-zinc-100 text-zinc-950 shadow hover:bg-zinc-200": variant === "default",
            "bg-red-950/30 text-red-400 border border-red-900/50 hover:bg-red-900/20": variant === "destructive",
            "border border-zinc-800 bg-transparent text-zinc-200 shadow-sm hover:bg-zinc-900 hover:text-zinc-100": variant === "outline",
            "bg-zinc-900 text-zinc-100 shadow-sm hover:bg-zinc-800": variant === "secondary",
            "hover:bg-zinc-900 hover:text-zinc-100 text-zinc-400": variant === "ghost",
            "text-zinc-200 underline-offset-4 hover:underline": variant === "link",
            "border border-emerald-950 bg-emerald-950/10 text-emerald-400 hover:bg-emerald-950/30 font-mono": variant === "terminal",
          },
          {
            "h-9 px-4 py-2": size === "default",
            "h-8 px-3 text-xs": size === "sm",
            "h-10 px-8": size === "lg",
            "h-9 w-9 p-0": size === "icon",
          },
          className
        )}
        ref={ref}
        {...props}
      />
    )
  }
)
Button.displayName = "Button"

export { Button }
