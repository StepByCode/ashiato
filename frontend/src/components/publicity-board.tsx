"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { CheckCheck, Loader2, Megaphone, RotateCcw } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Textarea } from "@/components/ui/textarea";
import { useAuth } from "@/lib/auth-context";
import { cn } from "@/lib/utils";
import { apiFetch } from "@/lib/api";
import { usePeriod } from "@/lib/period-context";

import { WorkflowShell } from "./workflow-shell";

type ChannelState = "in_progress" | "done";
type ChannelId = "x" | "instagram" | "facebook";

type ChannelCard = {
  id: ChannelId;
  name: string;
  note: string;
  state: ChannelState;
};

type TaskState = "in_progress" | "done" | "approved";

const REQUIRED_CHANNELS: { id: ChannelId; name: string }[] = [
  { id: "x", name: "X" },
  { id: "instagram", name: "Instagram" },
  { id: "facebook", name: "Facebook" },
];

const DEFAULT_CHANNELS: ChannelCard[] = REQUIRED_CHANNELS.map((rc) => ({
  id: rc.id,
  name: rc.name,
  note: "",
  state: "in_progress",
}));

const MAX_PUBLICITY_LENGTH = 140;
const channelStateLabels: Record<ChannelState, string> = {
  in_progress: "in progress",
  done: "Done",
};

const channelStateMeta: Record<
  ChannelState,
  {
    cardClassName: string;
    surfaceClassName: string;
  }
> = {
  in_progress: {
    cardClassName: "border-border/70 bg-card/95",
    surfaceClassName: "border-border/70 bg-background/80",
  },
  done: {
    cardClassName: "border-emerald-300/60 bg-[var(--surface-success)]",
    surfaceClassName: "border-emerald-300/60 bg-emerald-50 dark:bg-emerald-900/30",
  },
};

export function PublicityBoard() {
  const { getIdToken } = useAuth();
  const { selectedPeriod } = usePeriod();
  const [template, setTemplate] = useState("");
  const [channels, setChannels] = useState<ChannelCard[]>([]);
  const [loading, setLoading] = useState(true);
  const [creationDone, setCreationDone] = useState(false);
  const [pendingAction, setPendingAction] = useState<{ channelId: ChannelId; targetState: ChannelState } | null>(null);
  const templateTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const saveNoticeTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const [showSaveNotice, setShowSaveNotice] = useState(false);

  const flashSaveNotice = useCallback(() => {
    setShowSaveNotice(true);
    if (saveNoticeTimer.current) clearTimeout(saveNoticeTimer.current);
    saveNoticeTimer.current = setTimeout(() => setShowSaveNotice(false), 3000);
  }, []);

  const fetchData = useCallback(async (year: number, month: number) => {
    setLoading(true);
    setCreationDone(false);
    setChannels(DEFAULT_CHANNELS);
    try {
      const [templateRes, channelsRes, tasksRes] = await Promise.all([
        apiFetch(`/api/v1/publicity/template?year=${year}&month=${month}`, null),
        apiFetch(`/api/v1/publicity/channels?year=${year}&month=${month}`, null),
        apiFetch(`/api/v1/tasks?year=${year}&month=${month}`, null),
      ]);
      if (templateRes.ok) {
        const data = await templateRes.json();
        setTemplate(data.text ?? "");
      }
      if (channelsRes.ok) {
        const data = await channelsRes.json();
        const fetched = new Map(
          (data.channels ?? []).map((ch: Record<string, string>) => [
            ch.id as ChannelId,
            {
              id: ch.id as ChannelId,
              name: ch.name,
              note: ch.note ?? "",
              state: (ch.state as ChannelState) ?? "in_progress",
            },
          ])
        );
        const ensured = REQUIRED_CHANNELS.map((rc) => {
          const existing = fetched.get(rc.id);
          const fallback: ChannelCard = {
            id: rc.id,
            name: rc.name,
            note: "",
            state: "in_progress",
          };
          return existing ?? fallback;
        }) as ChannelCard[];
        setChannels(ensured);
      } else {
        setChannels(DEFAULT_CHANNELS);
      }

      if (tasksRes.ok) {
        const data = await tasksRes.json();
        const taskList: { state?: TaskState }[] = data.tasks ?? [];
        const allCreationDone =
          taskList.length > 0 && taskList.every((task) => task.state && task.state !== "in_progress");
        setCreationDone(allCreationDone);
      }
    } catch {
      // API unreachable - show empty state
      setCreationDone(false);
      setChannels(DEFAULT_CHANNELS);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchData(selectedPeriod.year, selectedPeriod.month);
  }, [fetchData, selectedPeriod.year, selectedPeriod.month]);

  const templateLength = template.length;
  const publicityDone = channels.length > 0 && channels.every((channel) => channel.state === "done");
  const isWorkflowComplete = !loading && publicityDone && creationDone;

  const handleTemplateChange = useCallback((text: string) => {
    setTemplate(text);
    if (templateTimer.current) clearTimeout(templateTimer.current);
    templateTimer.current = setTimeout(async () => {
      const token = await getIdToken();
      const res = await apiFetch("/api/v1/publicity/template", token, {
        method: "PATCH",
        body: JSON.stringify({ text, year: selectedPeriod.year, month: selectedPeriod.month }),
      });
      if (res.ok) {
        flashSaveNotice();
      }
    }, 600);
  }, [flashSaveNotice, getIdToken, selectedPeriod.month, selectedPeriod.year]);

  const updateChannelState = async (id: ChannelId, state: ChannelState) => {
    setChannels((currentChannels) =>
      currentChannels.map((channel) =>
        channel.id === id ? { ...channel, state } : channel
      )
    );
    setPendingAction(null);
    try {
      const token = await getIdToken();
      await apiFetch(`/api/v1/publicity/channels/${id}/state`, token, {
        method: "PATCH",
        body: JSON.stringify({ state, year: selectedPeriod.year, month: selectedPeriod.month }),
      });
    } catch {
      // ignore network errors
    }
  };

  if (loading) {
    return (
      <WorkflowShell activeStep="広報">
        <div className="flex items-center justify-center py-20 text-muted-foreground">
          <Loader2 className="mr-2 size-5 animate-spin" />
          読み込み中...
        </div>
      </WorkflowShell>
    );
  }

  return (
    <WorkflowShell activeStep="広報" isWorkflowComplete={isWorkflowComplete}>
      <section className="grid gap-4 xl:grid-cols-[minmax(0,1.2fr)_minmax(320px,0.9fr)]">
        {showSaveNotice ? (
          <div className="fixed bottom-5 right-5 z-50 rounded-full border border-emerald-300/60 bg-emerald-50 px-4 py-2 text-sm font-semibold text-emerald-900 shadow-lg">
            保存しました
          </div>
        ) : null}
        <Card className="rounded-[1.75rem] border-border/70 bg-card/95 shadow-sm">
          <CardContent className="space-y-6 p-5 sm:p-6">
            <div className="space-y-2">
              <p className="text-sm font-semibold uppercase tracking-[0.2em] text-muted-foreground">Publicity Copy</p>
              <h3 className="flex items-center gap-2 text-2xl font-semibold tracking-tight sm:text-3xl">
                <Megaphone className="size-5 text-primary" />
                広報文章テンプレート
              </h3>
            </div>

            <div className="space-y-3">
              <Textarea
                id="publicity-template"
                className="min-h-64 rounded-[1.5rem] border-border/70 bg-background/85 px-4 py-4 text-base leading-7 shadow-sm"
                maxLength={MAX_PUBLICITY_LENGTH}
                value={template}
                onChange={(event) => handleTemplateChange(event.target.value)}
              />

              <div className="flex flex-col gap-2 rounded-[1.25rem] border border-border/70 bg-background/60 px-4 py-3 text-sm sm:flex-row sm:items-center sm:justify-between">
                <span className="font-medium text-muted-foreground">X準拠の140文字以内で管理</span>
                <span
                  className={cn(
                    "font-semibold tabular-nums",
                    templateLength >= MAX_PUBLICITY_LENGTH ? "text-primary" : "text-foreground"
                  )}
                >
                  {templateLength} / {MAX_PUBLICITY_LENGTH}
                </span>
              </div>
            </div>
          </CardContent>
        </Card>

        <div className="grid gap-4">
          {channels.map((channel) => {
            const isDone = channel.state === "done";
            const meta = channelStateMeta[channel.state];
            const targetState = isDone ? "in_progress" : "done";
            const isConfirmOpen = pendingAction?.channelId === channel.id;
            const pendingCopy = `${channelStateLabels[targetState]} に変更しますか？`;

            return (
              <Card
                key={channel.id}
                className={cn("rounded-[1.75rem] border-2 shadow-sm transition-colors", meta.cardClassName)}
              >
                <CardContent className="p-5 sm:p-6">
                  <div className="mb-[10px] flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
                    <h4 className="text-2xl font-semibold tracking-tight">{channel.name}</h4>

                    <Button
                      type="button"
                      variant={isDone ? "secondary" : "default"}
                      className="min-h-14 w-full rounded-[1.25rem] px-6 text-base font-semibold shadow-sm sm:w-auto sm:min-w-52"
                      onClick={() => setPendingAction({ channelId: channel.id, targetState })}
                    >
                      {isDone ? <RotateCcw className="size-4" /> : <CheckCheck className="size-4" />}
                      Done
                    </Button>
                  </div>

                  {isConfirmOpen ? (
                    <div className="mt-3 grid w-full gap-2 rounded-[1.25rem] border border-border/70 bg-popover/95 p-3 text-sm font-medium shadow-sm">
                      <span className="whitespace-nowrap text-foreground">{pendingCopy}</span>
                      <div className="flex flex-wrap gap-2">
                        <Button
                          type="button"
                          variant="outline"
                          className="h-10 rounded-full px-4 text-sm font-semibold"
                          onClick={() => setPendingAction(null)}
                        >
                          いいえ
                        </Button>
                        <Button
                          type="button"
                          className="h-10 rounded-full px-4 text-sm font-semibold"
                          onClick={() => updateChannelState(channel.id, targetState)}
                        >
                          はい
                        </Button>
                      </div>
                    </div>
                  ) : null}

                </CardContent>
              </Card>
            );
          })}
        </div>
      </section>
    </WorkflowShell>
  );
}
