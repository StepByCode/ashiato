"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { CheckCheck, Loader2, Megaphone, RotateCcw } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Textarea } from "@/components/ui/textarea";
import { cn } from "@/lib/utils";
import { apiFetch } from "@/lib/api";

import { WorkflowShell } from "./workflow-shell";

type ChannelState = "in_progress" | "done";
type ChannelId = "x" | "instagram" | "facebook";

type ChannelCard = {
  id: ChannelId;
  name: string;
  note: string;
  state: ChannelState;
};

const MAX_PUBLICITY_LENGTH = 140;
const channelStateLabels: Record<ChannelState, string> = {
  in_progress: "in progress",
  done: "Done",
};

const channelStateMeta: Record<
  ChannelState,
  {
    cardClassName: string;
    stateContainerClassName: string;
    stateLabelClassName: string;
    stateValueClassName: string;
  }
> = {
  in_progress: {
    cardClassName: "border-border/70 bg-card/95",
    stateContainerClassName: "border-border/70 bg-background/80",
    stateLabelClassName: "text-muted-foreground",
    stateValueClassName: "text-foreground",
  },
  done: {
    cardClassName: "border-emerald-300/60 bg-[var(--surface-success)]",
    stateContainerClassName: "border-emerald-300/60 bg-emerald-100",
    stateLabelClassName: "text-emerald-900/70",
    stateValueClassName: "text-emerald-950",
  },
};

export function PublicityBoard() {
  const [template, setTemplate] = useState("");
  const [channels, setChannels] = useState<ChannelCard[]>([]);
  const [loading, setLoading] = useState(true);
  const [pendingAction, setPendingAction] = useState<{
    channelId: ChannelId;
    targetState: ChannelState;
  } | null>(null);
  const templateTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  const fetchData = useCallback(async () => {
    try {
      const [templateRes, channelsRes] = await Promise.all([
        apiFetch("/api/v1/publicity/template", null),
        apiFetch("/api/v1/publicity/channels", null),
      ]);
      if (templateRes.ok) {
        const data = await templateRes.json();
        setTemplate(data.text ?? "");
      }
      if (channelsRes.ok) {
        const data = await channelsRes.json();
        const fetched = (data.channels ?? []).map((ch: Record<string, string>) => ({
          id: ch.id as ChannelId,
          name: ch.name,
          note: ch.note ?? "",
          state: ch.state as ChannelState,
        }));
        setChannels(fetched);
      }
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const templateLength = template.length;
  const isWorkflowComplete = channels.length > 0 && channels.every((channel) => channel.state === "done");

  const handleTemplateChange = (text: string) => {
    setTemplate(text);
    if (templateTimer.current) clearTimeout(templateTimer.current);
    templateTimer.current = setTimeout(async () => {
      await apiFetch("/api/v1/publicity/template", null, {
        method: "PATCH",
        body: JSON.stringify({ text }),
      });
    }, 600);
  };

  const updateChannelState = async (id: ChannelId, state: ChannelState) => {
    setChannels((currentChannels) =>
      currentChannels.map((channel) =>
        channel.id === id ? { ...channel, state } : channel
      )
    );
    setPendingAction(null);
    await apiFetch(`/api/v1/publicity/channels/${id}/state`, null, {
      method: "PATCH",
      body: JSON.stringify({ state }),
    });
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
            const pendingCopy = `（${channelStateLabels[targetState]}）に変更しますか？`;

            return (
              <Card
                key={channel.id}
                className={cn("rounded-[1.75rem] border-2 shadow-sm transition-colors", meta.cardClassName)}
              >
                <CardContent className="p-5 sm:p-6">
                  <div className="mb-[10px] flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
                    <div className="space-y-2">
                      <h4 className="text-2xl font-semibold tracking-tight">{channel.name}</h4>
                      <p className="text-sm text-muted-foreground">{channel.note}</p>
                    </div>

                    <Button
                      type="button"
                      variant={isDone ? "secondary" : "default"}
                      className="min-h-14 w-full rounded-[1.25rem] px-6 text-base font-semibold shadow-sm sm:w-auto sm:min-w-52"
                      onClick={() => setPendingAction({ channelId: channel.id, targetState })}
                    >
                      {isDone ? <RotateCcw className="size-4" /> : <CheckCheck className="size-4" />}
                      {isDone ? "in progressに戻す" : "Doneにする"}
                    </Button>
                  </div>

                  <div
                    className={cn(
                      "flex min-h-28 flex-col justify-center rounded-[1.5rem] border px-5 py-[5px] shadow-sm",
                      isConfirmOpen ? "mb-[10px]" : "",
                      meta.stateContainerClassName
                    )}
                  >
                    <span className={cn("text-[1.75rem] font-semibold tracking-tight sm:text-[2.25rem]", meta.stateValueClassName)}>
                      {isDone ? "Done" : "in progress"}
                    </span>
                  </div>

                  {isConfirmOpen ? (
                    <div className="grid w-full gap-2 rounded-[1.25rem] border border-border/70 bg-popover/95 p-3 text-sm font-medium shadow-sm">
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
