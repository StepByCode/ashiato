"use client";

import type { ComponentProps } from "react";
import { ChevronLeft, ChevronRight } from "lucide-react";
import { DayPicker } from "react-day-picker";
import { ja } from "react-day-picker/locale";

import { cn } from "@/lib/utils";

function Calendar({
  className,
  classNames,
  showOutsideDays = true,
  ...props
}: ComponentProps<typeof DayPicker>) {
  return (
    <DayPicker
      showOutsideDays={showOutsideDays}
      locale={ja}
      navLayout="around"
      fixedWeeks
      className={cn("w-fit", className)}
      classNames={{
        root: "w-full",
        months: "flex w-full flex-col",
        month: "grid w-full grid-cols-[auto_1fr_auto] items-center gap-x-2 gap-y-3",
        month_caption: "flex h-10 items-center justify-center",
        caption_label: "text-center text-sm font-semibold tracking-tight text-foreground",
        nav: "hidden",
        button_previous:
          "inline-flex size-8 items-center justify-center rounded-full border border-border/70 bg-background/90 text-foreground shadow-sm transition hover:border-primary/40 hover:bg-background",
        button_next:
          "inline-flex size-8 items-center justify-center rounded-full border border-border/70 bg-background/90 text-foreground shadow-sm transition hover:border-primary/40 hover:bg-background",
        chevron: "size-4 text-primary",
        month_grid: "col-span-3 w-full border-collapse",
        weekdays: "flex w-full",
        weekday: "w-10 text-center text-xs font-semibold text-muted-foreground",
        weeks: "mt-2 flex flex-col gap-1",
        week: "flex w-full",
        day: "h-10 w-10 p-0 text-center text-sm",
        day_button:
          "flex size-10 items-center justify-center rounded-2xl border border-transparent bg-transparent font-medium text-foreground transition hover:bg-primary/10 focus-visible:border-primary focus-visible:ring-4 focus-visible:ring-primary/15 focus-visible:outline-none",
        selected:
          "[&>button]:border-primary [&>button]:bg-primary [&>button]:text-primary-foreground [&>button]:shadow-sm",
        today: "[&>button]:border-primary/40 [&>button]:text-primary",
        outside: "text-muted-foreground/35",
        disabled: "text-muted-foreground/30",
        hidden: "invisible",
        ...classNames,
      }}
      components={{
        Chevron: ({ className: chevronClassName, orientation, disabled: _disabled, ...iconProps }) => {
          if (orientation === "left") {
            return <ChevronLeft className={cn("size-4", chevronClassName)} {...iconProps} />;
          }

          return <ChevronRight className={cn("size-4", chevronClassName)} {...iconProps} />;
        },
      }}
      {...props}
    />
  );
}

export { Calendar };
