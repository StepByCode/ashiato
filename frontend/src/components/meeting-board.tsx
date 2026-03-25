"use client";

import { useEffect, useMemo, useState } from "react";
import { CalendarDays, Clock3, ExternalLink, Link2 } from "lucide-react";

import { buttonVariants } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";

import { WorkflowShell } from "./workflow-shell";

function getDefaultMeetingDateTime() {
  const nextMeeting = new Date();
  nextMeeting.setDate(nextMeeting.getDate() + 12);
  nextMeeting.setHours(20, 0, 0, 0);

  const offset = nextMeeting.getTimezoneOffset() * 60_000;
  return new Date(nextMeeting.getTime() - offset).toISOString().slice(0, 16);
}

function formatMeetingDateTime(value: string) {
  const parsedDate = new Date(value);
  if (Number.isNaN(parsedDate.getTime())) return "日時を設定してください";

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

export function MeetingBoard() {
  const [meetingAt, setMeetingAt] = useState(getDefaultMeetingDateTime);
  const [meetUrl, setMeetUrl] = useState("https://meet.google.com/abc-defg-hij");
  const [now, setNow] = useState(Date.now());

  useEffect(() => {
    const timerId = window.setInterval(() => {
      setNow(Date.now());
    }, 1000);

    return () => window.clearInterval(timerId);
  }, []);

  const countdown = useMemo(() => {
    const targetTime = new Date(meetingAt).getTime();
    if (Number.isNaN(targetTime)) {
      return {
        dayCount: "--",
        detail: "日時を設定するとカウントダウンが表示されます",
        isExpired: false,
      };
    }

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

  return (
    <WorkflowShell activeStep="定例">
      <section className="grid gap-4 xl:grid-cols-[minmax(0,1.3fr)_minmax(320px,0.85fr)]">
        <Card className="rounded-[1.75rem] border-border/70 bg-card/95 shadow-sm">
          <CardContent className="space-y-6 p-5 sm:p-6">
            <div className="space-y-2">
              <p className="text-sm font-semibold uppercase tracking-[0.2em] text-muted-foreground">Regular Meeting</p>
              <h3 className="text-2xl font-semibold tracking-tight sm:text-3xl">定例情報</h3>
            </div>

            <div className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]">
              <div className="space-y-3">
                <label htmlFor="meeting-datetime" className="flex items-center gap-2 text-sm font-semibold text-foreground">
                  <CalendarDays className="size-4 text-primary" />
                  定例日時
                </label>
                <Input
                  id="meeting-datetime"
                  className="h-12 rounded-2xl border-border/70 bg-background/85 px-4 shadow-sm"
                  type="datetime-local"
                  value={meetingAt}
                  onChange={(event) => setMeetingAt(event.target.value)}
                />
                <p className="text-sm text-muted-foreground">{formatMeetingDateTime(meetingAt)}</p>
              </div>

              <div className="space-y-3">
                <label htmlFor="meeting-url" className="flex items-center gap-2 text-sm font-semibold text-foreground">
                  <Link2 className="size-4 text-primary" />
                  Meetのリンク貼付
                </label>
                <Input
                  id="meeting-url"
                  className="h-12 rounded-2xl border-border/70 bg-background/85 px-4 text-[color:var(--url-color)] shadow-sm placeholder:text-[color:var(--url-color)] placeholder:opacity-60"
                  type="url"
                  placeholder="https://meet.google.com/..."
                  value={meetUrl}
                  onChange={(event) => setMeetUrl(event.target.value)}
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
