"use client";

import { useEffect, useState, type ReactNode } from "react";
import Link from "next/link";
import { Plus } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Card, CardContent } from "@/components/ui/card";
import { cn } from "@/lib/utils";
import { apiFetch } from "@/lib/api";
import { usePeriod, periodsEqual, periodLabel, type Period } from "@/lib/period-context";

export type WorkflowStep = "定例" | "作成" | "広報";

const workflowSteps = [
  { label: "定例" as const, href: "/meeting" },
  { label: "作成" as const, href: "/" },
  { label: "広報" as const, href: "/publicity" },
] as const;

const progressWidths: Record<WorkflowStep, string> = {
  定例: "10%",
  作成: "45%",
  広報: "80%",
};

const workflowStepOrder: WorkflowStep[] = ["定例", "作成", "広報"];
const lastStepStorageKey = "ashiato:last-workflow-step";
const themeModeStorageKey = "ashiato:theme-mode";

function isWorkflowStep(value: string | null): value is WorkflowStep {
  return value === "定例" || value === "作成" || value === "広報";
}

export function WorkflowShell({
  activeStep,
  isWorkflowComplete = false,
  children,
}: {
  activeStep: WorkflowStep;
  isWorkflowComplete?: boolean;
  children: ReactNode;
}) {
  const { periods, selectedPeriod, selectPeriod, refetchPeriods } = usePeriod();

  const [themeMode, setThemeMode] = useState<"light" | "dark">(() => {
    if (typeof window === "undefined") return "light";

    const savedThemeMode = window.localStorage.getItem(themeModeStorageKey);
    if (savedThemeMode === "light" || savedThemeMode === "dark") {
      return savedThemeMode;
    }

    return "light";
  });
  const [animatedStep, setAnimatedStep] = useState<WorkflowStep>(() => {
    if (typeof window === "undefined") return activeStep;
    const savedStep = window.sessionStorage.getItem(lastStepStorageKey);
    return isWorkflowStep(savedStep) ? savedStep : activeStep;
  });
  const [showCompleteLabel, setShowCompleteLabel] = useState(false);
  const [completionPhase, setCompletionPhase] = useState<"idle" | "showing" | "after">("idle");
  const [provisioning, setProvisioning] = useState(false);

  const progressWidth = isWorkflowComplete ? "calc(100% - 4rem)" : progressWidths[animatedStep];
  const showCompletionFrame = isWorkflowComplete && completionPhase === "after";

  // Build display months from periods (up to 4, newest first).
  const displayMonths: Period[] =
    periods.length > 0
      ? periods.slice(0, 4)
      : [selectedPeriod];

  // 当月・来月のうち未発行の月を検出する。
  const missingPeriods: Period[] = (() => {
    const now = new Date();
    const targets: Period[] = [
      { year: now.getFullYear(), month: now.getMonth() + 1 }, // 当月
    ];
    const next = new Date(now.getFullYear(), now.getMonth() + 1, 1);
    targets.push({ year: next.getFullYear(), month: next.getMonth() + 1 }); // 来月

    return targets.filter(
      (t) => !periods.some((p) => periodsEqual(p, t))
    );
  })();

  const handleProvision = async () => {
    if (missingPeriods.length === 0) return;
    setProvisioning(true);
    try {
      for (const target of missingPeriods) {
        await apiFetch("/api/v1/workflow-periods/provision", null, {
          method: "POST",
          body: JSON.stringify({ year: target.year, month: target.month }),
        });
      }
      await refetchPeriods();
      // 発行した中で最も新しい月に遷移
      const newest = missingPeriods[missingPeriods.length - 1];
      selectPeriod(newest);
    } catch {
      // ignore
    } finally {
      setProvisioning(false);
    }
  };

  useEffect(() => {
    document.documentElement.classList.toggle("dark", themeMode === "dark");
    window.localStorage.setItem(themeModeStorageKey, themeMode);
  }, [themeMode]);

  useEffect(() => {
    window.sessionStorage.setItem(lastStepStorageKey, activeStep);

    const animationFrameId = window.requestAnimationFrame(() => {
      setAnimatedStep(activeStep);
    });

    return () => window.cancelAnimationFrame(animationFrameId);
  }, [activeStep]);

  useEffect(() => {
    if (!isWorkflowComplete) {
      setShowCompleteLabel(false);
      setCompletionPhase("idle");
      return;
    }

    setShowCompleteLabel(true);
    setCompletionPhase("showing");

    const timerId = window.setTimeout(() => {
      setShowCompleteLabel(false);
      setCompletionPhase("after");
    }, 2000);

    return () => window.clearTimeout(timerId);
  }, [isWorkflowComplete]);

  return (
    <div className="min-h-screen px-4 py-4 sm:px-6 sm:py-6 lg:h-screen lg:overflow-hidden">
      <div className="mx-auto flex max-w-7xl flex-col gap-4 lg:h-full lg:flex-row">
        <aside className="hidden w-full max-w-64 shrink-0 rounded-[2rem] border border-border/70 bg-card/80 p-4 shadow-lg shadow-black/5 backdrop-blur lg:sticky lg:top-0 lg:relative lg:flex lg:h-full lg:flex-col lg:overflow-hidden">
          <div className="space-y-1 px-2 py-2">
            <h1 className="text-2xl font-semibold tracking-tight">Backstage</h1>
          </div>

          <div className="mt-8 flex flex-1 flex-col justify-center gap-3">
            {displayMonths.map((period) => {
              const isSelected = periodsEqual(period, selectedPeriod);
              return (
                <button
                  key={`${period.year}-${period.month}`}
                  type="button"
                  onClick={() => selectPeriod(period)}
                >
                  <Badge
                    variant="outline"
                    className={cn(
                      "min-h-12 rounded-full border px-4 py-3 text-base font-semibold shadow-sm cursor-pointer",
                      isSelected
                        ? "border-[color:var(--accent)] text-[color:var(--accent)] ring-2 ring-[color:var(--accent)]/40"
                        : "border-border/70 bg-background/80 text-foreground hover:bg-background/95"
                    )}
                  >
                    {periodLabel(period)}
                  </Badge>
                </button>
              );
            })}
            {missingPeriods.length > 0 && (
              <button
                type="button"
                disabled={provisioning}
                onClick={handleProvision}
                className="flex min-h-12 items-center justify-center gap-2 rounded-full border border-dashed border-border/70 px-4 py-3 text-sm font-semibold text-muted-foreground shadow-sm transition-colors hover:border-primary/40 hover:text-foreground disabled:opacity-50"
              >
                <Plus className="size-4" />
                {provisioning
                  ? "発行中..."
                  : missingPeriods.map((p) => periodLabel(p)).join("・") + " を発行"}
              </button>
            )}
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
                <h2 className="text-xl font-semibold tracking-tight">Backstage</h2>
              </div>

              <div className="lg:hidden">
                <div className="flex items-center justify-between gap-3 pb-1">
                  <div className="flex min-w-0 gap-3 overflow-x-auto">
                    {displayMonths.map((period) => {
                      const isSelected = periodsEqual(period, selectedPeriod);
                      return (
                        <button
                          key={`mobile-${period.year}-${period.month}`}
                          type="button"
                          onClick={() => selectPeriod(period)}
                        >
                          <Badge
                            variant="outline"
                            className={cn(
                              "min-h-11 shrink-0 rounded-full px-4 text-sm font-semibold shadow-sm cursor-pointer",
                              isSelected
                                ? "border-[color:var(--accent)] text-[color:var(--accent)] ring-2 ring-[color:var(--accent)]/40"
                                : "border-border/70 bg-background/80 text-foreground"
                            )}
                          >
                            {periodLabel(period)}
                          </Badge>
                        </button>
                      );
                    })}
                    {missingPeriods.length > 0 && (
                      <button
                        type="button"
                        disabled={provisioning}
                        onClick={handleProvision}
                        className="flex min-h-11 shrink-0 items-center gap-1 rounded-full border border-dashed border-border/70 px-3 text-xs font-semibold text-muted-foreground shadow-sm hover:border-primary/40 hover:text-foreground disabled:opacity-50"
                      >
                        <Plus className="size-3" />
                        {provisioning ? "..." : "発行"}
                      </button>
                    )}
                  </div>

                  <div className="inline-grid shrink-0 grid-cols-2 gap-1 rounded-xl border border-border/70 bg-background/80 p-1 shadow-sm">
                    <button
                      type="button"
                      className={cn(
                        "h-9 min-w-14 rounded-lg px-2 text-xs font-semibold transition-colors",
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
                        "h-9 min-w-14 rounded-lg px-2 text-xs font-semibold transition-colors",
                        themeMode === "dark"
                          ? "bg-primary text-primary-foreground"
                          : "text-muted-foreground hover:bg-background/80 hover:text-foreground"
                      )}
                      onClick={() => setThemeMode("dark")}
                    >
                      Dark
                    </button>
                  </div>
                </div>
              </div>

              <Card
                className={cn(
                  "rounded-[1.75rem] border-border/70 bg-background/70",
                  showCompletionFrame && "border-[color:var(--accent)] ring-2 ring-[color:var(--accent)]/40"
                )}
              >
                <CardContent className="p-4 sm:p-5">
                  <div className="relative min-h-16">
                    <div
                      className={cn(
                        "relative grid grid-cols-3 gap-3 transition-opacity duration-300 sm:gap-5",
                        showCompleteLabel ? "pointer-events-none opacity-0" : "opacity-100"
                      )}
                    >
                      <div className="pointer-events-none absolute left-8 right-8 top-6 h-px bg-[color:var(--accent)]/35 sm:top-7" />
                      <div
                        className="pointer-events-none absolute left-8 top-[22px] h-2 rounded-full bg-[color:var(--accent)] transition-[width] duration-700 ease-in-out sm:top-[25px]"
                        style={{ width: progressWidth }}
                      />
                      {workflowSteps.map((step) => {
                        const isActive = step.label === activeStep;
                        const isCompleted =
                          isWorkflowComplete ||
                          workflowStepOrder.indexOf(step.label) < workflowStepOrder.indexOf(animatedStep);
                        const isCompletedVisual = isCompleted && (!isActive || isWorkflowComplete);

                        return (
                          <Link
                            key={step.label}
                            href={step.href}
                            aria-current={isActive ? "page" : undefined}
                            className="relative z-10 flex flex-col items-center gap-2 text-center"
                          >
                            <div
                              className={cn(
                                "flex size-12 items-center justify-center rounded-full border-2 bg-background text-sm font-semibold shadow-sm sm:size-14",
                                "transition-all duration-700 ease-out",
                                isActive
                                  ? "border-[color:var(--accent)] bg-[color:var(--accent)] text-white"
                                  : isCompletedVisual
                                    ? "border-[color:var(--accent)] text-foreground hover:-translate-y-0.5 hover:bg-background/95"
                                    : "border-border/80 text-foreground hover:-translate-y-0.5 hover:border-[color:var(--accent)]/50 hover:bg-background/95"
                              )}
                            >
                              {step.label}
                            </div>
                          </Link>
                        );
                      })}
                    </div>

                    <div
                      className={cn(
                        "pointer-events-none absolute inset-0 z-20 flex items-center justify-center transition-all duration-500",
                        showCompleteLabel
                          ? "translate-y-0 opacity-100"
                          : "-translate-y-6 opacity-0"
                      )}
                    >
                      <span className="text-4xl font-black uppercase tracking-[0.2em] text-zinc-700 dark:text-primary sm:text-5xl">
                        complete!
                      </span>
                    </div>
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
