"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Popover } from "@base-ui/react/popover";
import { CalendarDays, ChevronDown, Clock3, ExternalLink, Link2, Loader2, Share2 } from "lucide-react";

import { Button, buttonVariants } from "@/components/ui/button";
import { Calendar } from "@/components/ui/calendar";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import { apiFetch } from "@/lib/api";
import { usePeriod } from "@/lib/period-context";

import { WorkflowShell } from "./workflow-shell";

function getDefaultMeetingDateTime() {
  const nextMeeting = new Date();
  nextMeeting.setDate(nextMeeting.getDate() + 12);
  nextMeeting.setHours(20, 0, 0, 0);

  return toLocalDateTimeValue(nextMeeting);
}

function toLocalDateTimeValue(date: Date) {
  const offset = date.getTimezoneOffset() * 60_000;
  return new Date(date.getTime() - offset).toISOString().slice(0, 16);
}

function parseMeetingDateTime(value: string) {
  const parsedDate = new Date(value);
  if (Number.isNaN(parsedDate.getTime())) return undefined;
  return parsedDate;
}

function formatMeetingDateTime(value: string) {
  const parsedDate = parseMeetingDateTime(value);
  if (!parsedDate) return "日時を設定してください";

  return new Intl.DateTimeFormat("ja-JP", {
    month: "long",
    day: "numeric",
    weekday: "short",
    hour: "2-digit",
    minute: "2-digit",
  }).format(parsedDate);
}

function toJumpHref(url: string) {
  if (!url.trim()) return "";
  if (/^https?:\/\//i.test(url)) return url;
  return `https://${url}`;
}

function formatTimeUnit(value: number) {
  return String(value).padStart(2, "0");
}

const hourOptions = Array.from({ length: 24 }, (_, index) => index);
const minuteOptions = [0, 15, 30, 45];

export function MeetingBoard() {
  const { selectedPeriod } = usePeriod();
  const [meetingAt, setMeetingAt] = useState(getDefaultMeetingDateTime);
  const [meetUrl, setMeetUrl] = useState("");
  const [loadingMeeting, setLoadingMeeting] = useState(true);
  const [now, setNow] = useState(Date.now());
  const [calendarMonth, setCalendarMonth] = useState(() => {
    const defaultDate = parseMeetingDateTime(getDefaultMeetingDateTime());
    return defaultDate ? new Date(defaultDate.getFullYear(), defaultDate.getMonth(), 1) : new Date();
  });
  const saveTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const initializedRef = useRef(false);

  const saveMeeting = useCallback((nextMeetingAt: string, nextMeetUrl: string, year: number, month: number) => {
    if (saveTimer.current) clearTimeout(saveTimer.current);
    saveTimer.current = setTimeout(async () => {
      const meetingDate = parseMeetingDateTime(nextMeetingAt);
      const body: Record<string, unknown> = { year, month };
      if (meetingDate) {
        body.meetingAt = meetingDate.toISOString();
      }
      body.meetUrl = nextMeetUrl;
      await apiFetch("/api/v1/meeting", null, {
        method: "PATCH",
        body: JSON.stringify(body),
      });
    }, 600);
  }, []);

  useEffect(() => {
    initializedRef.current = false;
    setLoadingMeeting(true);
    setMeetingAt(getDefaultMeetingDateTime());
    setMeetUrl("");

    (async () => {
      try {
        const res = await apiFetch(`/api/v1/meeting?year=${selectedPeriod.year}&month=${selectedPeriod.month}`, null);
        if (res.ok) {
          const data = await res.json();
          if (data.meetingAt) {
            const parsed = new Date(data.meetingAt);
            if (!Number.isNaN(parsed.getTime())) {
              const localVal = toLocalDateTimeValue(parsed);
              setMeetingAt(localVal);
              setCalendarMonth(new Date(parsed.getFullYear(), parsed.getMonth(), 1));
            }
          }
          if (data.meetUrl) {
            setMeetUrl(data.meetUrl);
          }
          initializedRef.current = true;
        }
      } catch {
        // API unreachable - use defaults
      } finally {
        initializedRef.current = true;
        setLoadingMeeting(false);
      }
    })();
  }, [selectedPeriod.year, selectedPeriod.month]);

  useEffect(() => {
    const timerId = window.setInterval(() => {
      setNow(Date.now());
    }, 1000);

    return () => window.clearInterval(timerId);
  }, []);

  const selectedDate = parseMeetingDateTime(meetingAt);
  const selectedHours = selectedDate?.getHours() ?? 20;
  const selectedMinutes = selectedDate?.getMinutes() ?? 0;

  const applyNextMeetingAt = (nextDate: Date) => {
    nextDate.setSeconds(0, 0);
    const nextVal = toLocalDateTimeValue(nextDate);
    setMeetingAt(nextVal);
    setCalendarMonth(new Date(nextDate.getFullYear(), nextDate.getMonth(), 1));
    if (initializedRef.current) saveMeeting(nextVal, meetUrl, selectedPeriod.year, selectedPeriod.month);
  };

  const handleSelectDate = (nextDay?: Date) => {
    if (!nextDay) return;

    const baseDate = selectedDate ?? new Date();
    const nextDate = new Date(nextDay);
    nextDate.setHours(baseDate.getHours(), baseDate.getMinutes(), 0, 0);
    applyNextMeetingAt(nextDate);
  };

  const handleTimeChange = (part: "hours" | "minutes", nextValue: number) => {
    const baseDate = selectedDate ?? parseMeetingDateTime(getDefaultMeetingDateTime()) ?? new Date();
    const nextDate = new Date(baseDate);

    if (part === "hours") {
      nextDate.setHours(nextValue);
    } else {
      nextDate.setMinutes(nextValue);
    }

    applyNextMeetingAt(nextDate);
  };

  const countdown = useMemo(() => {
    const targetDate = parseMeetingDateTime(meetingAt);
    if (!targetDate) {
      return {
        dayCount: "--",
        detail: "日時を設定するとカウントダウンが表示されます",
        isExpired: false,
      };
    }

    const targetTime = targetDate.getTime();
    const difference = targetTime - now;
    if (difference <= 0) {
      return {
        dayCount: "0",
        detail: "開催時刻を過ぎています",
        isExpired: true,
      };
    }

    const totalSeconds = Math.floor(difference / 1000);
    const days = Math.floor(totalSeconds / 86_400);
    const hours = Math.floor((totalSeconds % 86_400) / 3_600);
    const minutes = Math.floor((totalSeconds % 3_600) / 60);
    const seconds = totalSeconds % 60;

    return {
      dayCount: String(Math.ceil(difference / 86_400_000)),
      detail: `あと ${days}日 ${hours}時間 ${minutes}分 ${seconds}秒`,
      isExpired: false,
    };
  }, [meetingAt, now]);

  const jumpHref = toJumpHref(meetUrl);

  if (loadingMeeting) {
    return (
      <WorkflowShell activeStep="定例">
        <div className="flex items-center justify-center py-20 text-muted-foreground">
          <Loader2 className="mr-2 size-5 animate-spin" />
          読み込み中...
        </div>
      </WorkflowShell>
    );
  }

  return (
    <WorkflowShell activeStep="定例">
      <section className="grid gap-4 xl:grid-cols-[minmax(0,1.3fr)_minmax(320px,0.85fr)]">
        <Card className="rounded-[1.75rem] border-border/70 bg-card/95 shadow-sm">
          <CardContent className="space-y-6 p-5 sm:p-6">
            <div className="space-y-2">
              <p className="text-sm font-semibold uppercase tracking-[0.2em] text-muted-foreground">Regular Meeting</p>
              <h3 className="text-2xl font-semibold tracking-tight sm:text-3xl">定例情報</h3>
            </div>

            <div className="flex flex-wrap items-center gap-3 sm:justify-end">
              <Button
                type="button"
                variant="outline"
                className="min-h-11 rounded-full border-2 border-[color:var(--accent)] bg-[color:var(--accent)] px-6 text-base font-semibold text-white shadow-lg shadow-[color:var(--accent)]/30 transition hover:-translate-y-0.5 hover:bg-[color:var(--accent)]/90 hover:shadow-xl focus-visible:ring-[color:var(--accent)]"
              >
                <Share2 className="size-4" />
                定例の予定を共有
              </Button>
            </div>

            <div className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]">
              <div className="space-y-3">
                <label htmlFor="meeting-datetime" className="flex items-center gap-2 text-sm font-semibold text-foreground">
                  <CalendarDays className="size-4 text-primary" />
                  定例日時
                </label>
                <Popover.Root>
                  <Popover.Trigger
                    id="meeting-datetime"
                    className={cn(
                      buttonVariants({ variant: "outline" }),
                      "h-12 w-full justify-between rounded-2xl border-border/70 bg-background/85 px-4 text-left text-base font-medium text-foreground shadow-sm hover:bg-background/95"
                    )}
                  >
                    <span className="truncate">{formatMeetingDateTime(meetingAt)}</span>
                    <CalendarDays className="size-4 text-primary" />
                  </Popover.Trigger>

                  <Popover.Portal>
                    <Popover.Positioner side="bottom" align="start" sideOffset={10} className="z-50">
                      <Popover.Popup
                        initialFocus={false}
                        className="w-[min(92vw,360px)] rounded-[1.5rem] border border-border/70 bg-[var(--surface-popover)] p-4 shadow-xl shadow-black/10 backdrop-blur"
                      >
                        <div className="space-y-4">
                          <Calendar
                            mode="single"
                            month={calendarMonth}
                            onMonthChange={setCalendarMonth}
                            selected={selectedDate}
                            onSelect={handleSelectDate}
                            className="rounded-[1.25rem] bg-background/85 p-3"
                          />

                          <div className="grid gap-3 border-t border-border/70 pt-4 sm:grid-cols-2">
                            <div className="space-y-2">
                              <span className="flex items-center gap-2 text-sm font-semibold text-foreground">
                                <Clock3 className="size-4 text-primary" />
                                時
                              </span>
                              <div className="relative">
                                <select
                                  className="h-12 w-full appearance-none rounded-2xl border border-border/70 bg-background/85 px-4 pr-10 text-base font-medium shadow-sm outline-none transition focus:border-primary focus:ring-4 focus:ring-primary/15"
                                  aria-label="定例の時間"
                                  value={selectedHours}
                                  onChange={(event) => handleTimeChange("hours", Number(event.target.value))}
                                >
                                  {hourOptions.map((hour) => (
                                    <option key={hour} value={hour}>
                                      {formatTimeUnit(hour)}
                                    </option>
                                  ))}
                                </select>
                                <ChevronDown className="pointer-events-none absolute right-4 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
                              </div>
                            </div>

                            <div className="space-y-2">
                              <span className="text-sm font-semibold text-foreground">分</span>
                              <div className="relative">
                                <select
                                  className="h-12 w-full appearance-none rounded-2xl border border-border/70 bg-background/85 px-4 pr-10 text-base font-medium shadow-sm outline-none transition focus:border-primary focus:ring-4 focus:ring-primary/15"
                                  aria-label="定例の分"
                                  value={selectedMinutes}
                                  onChange={(event) => handleTimeChange("minutes", Number(event.target.value))}
                                >
                                  {minuteOptions.map((minute) => (
                                    <option key={minute} value={minute}>
                                      {formatTimeUnit(minute)}
                                    </option>
                                  ))}
                                </select>
                                <ChevronDown className="pointer-events-none absolute right-4 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
                              </div>
                            </div>
                          </div>
                        </div>
                      </Popover.Popup>
                    </Popover.Positioner>
                  </Popover.Portal>
                </Popover.Root>
                <p className="text-sm text-muted-foreground">{formatMeetingDateTime(meetingAt)}</p>
              </div>

              <div className="space-y-3">
                <label htmlFor="meeting-url" className="flex items-center gap-2 text-sm font-semibold text-foreground">
                  <Link2 className="size-4 text-primary" />
                  meetingLink
                </label>
                <Input
                  id="meeting-url"
                  className="h-12 rounded-2xl border-border/70 bg-background/85 px-4 text-[color:var(--url-color)] shadow-sm placeholder:text-[color:var(--url-color)] placeholder:opacity-60"
                  type="url"
                  placeholder="https://meet.google.com/..."
                  value={meetUrl}
                  onChange={(event) => {
                    const nextUrl = event.target.value;
                    setMeetUrl(nextUrl);
                    if (initializedRef.current) saveMeeting(meetingAt, nextUrl, selectedPeriod.year, selectedPeriod.month);
                  }}
                />
                {jumpHref ? (
                  <a
                    className={cn(
                      buttonVariants({ variant: "outline" }),
                      "min-h-12 rounded-2xl px-5 text-base font-semibold shadow-sm"
                    )}
                    href={jumpHref}
                    target="_blank"
                    rel="noreferrer"
                  >
                    Meetを開く
                    <ExternalLink className="size-4" />
                  </a>
                ) : (
                  <div className="flex min-h-12 items-center justify-center rounded-2xl border border-dashed border-border/70 px-5 text-sm font-medium text-muted-foreground">
                    Meetリンク未設定
                  </div>
                )}
              </div>
            </div>
          </CardContent>
        </Card>

        <Card
          className={cn(
            "rounded-[1.75rem] border-2 shadow-sm",
            countdown.isExpired
              ? "border-amber-300/60 bg-[color:color-mix(in_srgb,var(--accent)_16%,white)]"
              : "border-sky-300/60 bg-[var(--surface-accent)]"
          )}
        >
          <CardContent className="flex h-full flex-col justify-between gap-6 p-5 sm:p-6">
            <div className="space-y-2">
              <p className="text-sm font-semibold uppercase tracking-[0.2em] text-muted-foreground">Countdown</p>
              <h3 className="flex items-center gap-2 text-xl font-semibold tracking-tight">
                <Clock3 className="size-5 text-primary" />
                定例まで
              </h3>
            </div>

            <div className="space-y-3">
              <div className="text-5xl font-semibold tracking-tight sm:text-6xl">D-{countdown.dayCount}</div>
              <p className="text-base font-medium text-foreground/85">{countdown.detail}</p>
            </div>

            <div className="rounded-[1.25rem] border border-border/60 bg-background/70 px-4 py-3 text-sm text-muted-foreground">
              {formatMeetingDateTime(meetingAt)}
            </div>
          </CardContent>
        </Card>
      </section>
    </WorkflowShell>
  );
}
