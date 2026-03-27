"use client";

import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { apiFetch } from "./api";

export type Period = {
  year: number;
  month: number;
};

type WorkflowPeriodData = Period & {
  meetingStatus: string;
  taskCompletionStatus: string;
  announcementStatus: string;
  nextStageReady: boolean;
};

type PeriodContextValue = {
  periods: WorkflowPeriodData[];
  selectedPeriod: Period;
  selectPeriod: (period: Period) => void;
  loading: boolean;
  refetchPeriods: () => Promise<void>;
};

const STORAGE_KEY = "ashiato:selected-period";

function getDefaultPeriod(): Period {
  const now = new Date();
  return { year: now.getFullYear(), month: now.getMonth() + 1 };
}

function loadSavedPeriod(): Period | null {
  if (typeof window === "undefined") return null;
  try {
    const saved = window.sessionStorage.getItem(STORAGE_KEY);
    if (!saved) return null;
    const parsed = JSON.parse(saved);
    if (typeof parsed.year === "number" && typeof parsed.month === "number") {
      return parsed as Period;
    }
  } catch {
    // ignore
  }
  return null;
}

function savePeriod(period: Period) {
  if (typeof window === "undefined") return;
  window.sessionStorage.setItem(STORAGE_KEY, JSON.stringify(period));
}

export function periodLabel(period: Period): string {
  return `${period.year}.${period.month}月`;
}

export function periodsEqual(a: Period, b: Period): boolean {
  return a.year === b.year && a.month === b.month;
}

const PeriodContext = createContext<PeriodContextValue | null>(null);

export function PeriodProvider({ children }: { children: ReactNode }) {
  const [periods, setPeriods] = useState<WorkflowPeriodData[]>([]);
  const [selectedPeriod, setSelectedPeriod] = useState<Period>(() => {
    return loadSavedPeriod() ?? getDefaultPeriod();
  });
  const [loading, setLoading] = useState(true);

  const fetchPeriods = useCallback(async () => {
    try {
      const res = await apiFetch("/api/v1/workflow-periods", null);
      if (res.ok) {
        const data = await res.json();
        const fetched: WorkflowPeriodData[] = (data.periods ?? []).map(
          (p: Record<string, unknown>) => ({
            year: p.year as number,
            month: p.month as number,
            meetingStatus: (p.meeting_status ?? p.meetingStatus ?? "missing") as string,
            taskCompletionStatus: (p.task_completion_status ?? p.taskCompletionStatus ?? "empty") as string,
            announcementStatus: (p.announcement_status ?? p.announcementStatus ?? "missing") as string,
            nextStageReady: (p.next_stage_ready ?? p.nextStageReady ?? false) as boolean,
          })
        );
        // Sort descending (newest first).
        fetched.sort((a, b) => {
          if (a.year !== b.year) return b.year - a.year;
          return b.month - a.month;
        });
        setPeriods(fetched);

        // If no periods match selected, default to the first available.
        if (fetched.length > 0 && !fetched.some((p) => periodsEqual(p, selectedPeriod))) {
          const first = fetched[0];
          setSelectedPeriod({ year: first.year, month: first.month });
          savePeriod({ year: first.year, month: first.month });
        }
      }
    } catch {
      // API unreachable - use defaults
    } finally {
      setLoading(false);
    }
  }, [selectedPeriod]);

  useEffect(() => {
    fetchPeriods();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const selectPeriod = useCallback((period: Period) => {
    setSelectedPeriod(period);
    savePeriod(period);
  }, []);

  const value = useMemo<PeriodContextValue>(
    () => ({
      periods,
      selectedPeriod,
      selectPeriod,
      loading,
      refetchPeriods: fetchPeriods,
    }),
    [periods, selectedPeriod, selectPeriod, loading, fetchPeriods]
  );

  return <PeriodContext.Provider value={value}>{children}</PeriodContext.Provider>;
}

export function usePeriod(): PeriodContextValue {
  const ctx = useContext(PeriodContext);
  if (!ctx) {
    throw new Error("usePeriod must be used within PeriodProvider");
  }
  return ctx;
}
