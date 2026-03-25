"use client";

import { useEffect, useState, type ReactNode } from "react";
import Link from "next/link";

import { Badge } from "@/components/ui/badge";
import { Card, CardContent } from "@/components/ui/card";
import { cn } from "@/lib/utils";

export type WorkflowStep = "定例" | "作成" | "広報";

const months = ["2026.4月", "2026.3月", "2026.2月", "2026.1月"] as const;
const currentMonth = "2026.2月";
const workflowSteps = [
  { label: "定例" as const, href: "/meeting" },
  { label: "作成" as const, href: "/" },
  { label: "広報" as const, href: "/publicity" },
] as const;

export function WorkflowShell({
  activeStep,
  children,
}: {
  activeStep: WorkflowStep;
  children: ReactNode;
}) {
  const [themeMode, setThemeMode] = useState<"light" | "dark">("light");

  useEffect(() => {
    setThemeMode(document.documentElement.classList.contains("dark") ? "dark" : "light");
  }, []);

  useEffect(() => {
    document.documentElement.classList.toggle("dark", themeMode === "dark");
  }, [themeMode]);

  return (
    <div className="min-h-screen px-4 py-4 sm:px-6 sm:py-6 lg:h-screen lg:overflow-hidden">
      <div className="mx-auto flex max-w-7xl flex-col gap-4 lg:h-full lg:flex-row">
        <aside className="hidden w-full max-w-64 shrink-0 rounded-[2rem] border border-border/70 bg-card/80 p-4 shadow-lg shadow-black/5 backdrop-blur lg:sticky lg:top-0 lg:relative lg:flex lg:h-full lg:flex-col lg:overflow-hidden">
          <div className="space-y-1 px-2 py-2">
            <h1 className="text-2xl font-semibold tracking-tight">Ashiato</h1>
          </div>

          <div className="mt-8 flex flex-1 flex-col justify-center gap-3">
            {months.map((month) => (
              <Badge
                key={month}
                variant="outline"
                className={cn(
                  "min-h-12 rounded-full border px-4 py-3 text-base font-semibold shadow-sm",
                  month === currentMonth
                    ? "border-primary/30 bg-primary text-primary-foreground"
                    : "border-border/70 bg-background/80 text-foreground"
                )}
              >
                {month}
              </Badge>
            ))}
          </div>

          <div className="absolute bottom-4 right-4 inline-grid grid-cols-2 gap-1 rounded-2xl border border-border/70 bg-background/80 p-1 shadow-sm">
            <button
              type="button"
              className={cn(
                "h-10 min-w-20 rounded-xl px-3 text-sm font-semibold transition-colors",
                themeMode === "light"
                  ? "bg-primary text-primary-foreground"
                  : "text-muted-foreground hover:bg-background/80 hover:text-foreground"
              )}
              onClick={() => setThemeMode("light")}
            >
              Light
            </button>
            <button
              type="button"
              className={cn(
                "h-10 min-w-20 rounded-xl px-3 text-sm font-semibold transition-colors",
                themeMode === "dark"
                  ? "bg-primary text-primary-foreground"
                  : "text-muted-foreground hover:bg-background/80 hover:text-foreground"
              )}
              onClick={() => setThemeMode("dark")}
            >
              Dark
            </button>
          </div>
        </aside>

        <main className="flex-1 rounded-[2rem] border border-border/70 bg-card/80 p-4 shadow-lg shadow-black/5 backdrop-blur sm:p-6 lg:h-full lg:overflow-y-auto lg:p-8">
          <div className="space-y-6">
            <header className="space-y-5">
              <div className="lg:hidden">
                <h2 className="text-xl font-semibold tracking-tight">Ashiato</h2>
              </div>

              <div className="lg:hidden">
                <div className="flex gap-3 overflow-x-auto pb-1">
                  {months.map((month) => (
                    <Badge
                      key={month}
                      variant="outline"
                      className={cn(
                        "min-h-11 shrink-0 rounded-full px-4 text-sm font-semibold shadow-sm",
                        month === currentMonth
                          ? "border-primary/30 bg-primary text-primary-foreground"
                          : "border-border/70 bg-background/80 text-foreground"
                      )}
                    >
                      {month}
                    </Badge>
                  ))}
                </div>
              </div>

              <Card className="rounded-[1.75rem] border-border/70 bg-background/70">
                <CardContent className="p-4 sm:p-5">
                  <div className="relative grid grid-cols-3 gap-3 sm:gap-5">
                    <div className="pointer-events-none absolute left-8 right-8 top-6 h-px bg-primary/30 sm:top-7" />
                    {workflowSteps.map((step) => {
                      const isActive = step.label === activeStep;

                      return (
                        <Link
                          key={step.label}
                          href={step.href}
                          aria-current={isActive ? "page" : undefined}
                          className="relative z-10 flex flex-col items-center gap-2 text-center"
                        >
                          <div
                            className={cn(
                              "flex size-12 items-center justify-center rounded-full border-2 bg-background text-sm font-semibold shadow-sm transition-all sm:size-14",
                              isActive
                                ? "border-primary bg-primary text-primary-foreground"
                                : "border-border/80 text-foreground hover:-translate-y-0.5 hover:border-primary/40 hover:bg-background/95"
                            )}
                          >
                            {step.label}
                          </div>
                        </Link>
                      );
                    })}
                  </div>
                </CardContent>
              </Card>
            </header>

            {children}
          </div>
        </main>
      </div>
    </div>
  );
}
