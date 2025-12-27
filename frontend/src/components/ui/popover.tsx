import * as React from "react"
import * as PopoverPrimitive from "@radix-ui/react-popover"
import { cn } from "@/lib/utils"

export const Popover = PopoverPrimitive.Root

export const PopoverTrigger = PopoverPrimitive.Trigger

export const PopoverAnchor = PopoverPrimitive.Anchor

export const PopoverContent = React.forwardRef<
  React.ElementRef<typeof PopoverPrimitive.Content>,
  React.ComponentPropsWithoutRef<typeof PopoverPrimitive.Content>
>(({ className, align = "center", side = "top", onOpenAutoFocus, onCloseAutoFocus, ...props }, ref) => (
  <PopoverPrimitive.Content
    ref={ref}
    align={align}
    side={side}
    onOpenAutoFocus={(e) => {
      e.preventDefault()
      onOpenAutoFocus?.(e)
    }}
    onCloseAutoFocus={(e) => {
      e.preventDefault()
      onCloseAutoFocus?.(e)
    }}
    className={cn(
      "z-50 rounded-md border border-gray-200 bg-white p-3 text-sm shadow-md outline-none",
      "data-[state=open]:animate-in data-[state=closed]:animate-out",
      "data-[state=open]:fade-in data-[state=closed]:fade-out",
      "data-[side=bottom]:slide-in-from-top-2 data-[side=top]:slide-in-from-bottom-2 data-[side=left]:slide-in-from-right-2 data-[side=right]:slide-in-from-left-2",
      className
    )}
    {...props}
  />
))
PopoverContent.displayName = PopoverPrimitive.Content.displayName
