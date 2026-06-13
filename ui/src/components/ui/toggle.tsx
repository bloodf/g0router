import * as React from "react";
import * as SwitchPrimitives from "@radix-ui/react-switch";

import { cn } from "@/lib/utils";

const sizeStyles = {
  sm: {
    root: "h-4 w-7",
    thumb: "size-3 data-[state=checked]:translate-x-3",
  },
  md: {
    root: "h-5 w-9",
    thumb: "size-4 data-[state=checked]:translate-x-4",
  },
} as const;

export interface ToggleProps
  extends React.ComponentPropsWithoutRef<typeof SwitchPrimitives.Root> {
  size?: keyof typeof sizeStyles;
}

const Toggle = React.forwardRef<
  React.ElementRef<typeof SwitchPrimitives.Root>,
  ToggleProps
>(({ className, size = "md", ...props }, ref) => {
  const styles = sizeStyles[size];
  return (
    <SwitchPrimitives.Root
      ref={ref}
      className={cn(
        "peer inline-flex shrink-0 cursor-pointer items-center rounded-full border-2 border-transparent shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background disabled:cursor-not-allowed disabled:opacity-50 data-[state=checked]:bg-primary data-[state=unchecked]:bg-input",
        styles.root,
        className
      )}
      {...props}
    >
      <SwitchPrimitives.Thumb
        className={cn(
          "pointer-events-none block rounded-full bg-background shadow-lg ring-0 transition-transform data-[state=unchecked]:translate-x-0",
          styles.thumb
        )}
      />
    </SwitchPrimitives.Root>
  );
});
Toggle.displayName = "Toggle";

export { Toggle };
