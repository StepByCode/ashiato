"use client";

import { useState } from "react";
import { CheckCheck, Megaphone, RotateCcw } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Textarea } from "@/components/ui/textarea";
import { cn } from "@/lib/utils";
import { ChannelId, ChannelState, useWorkflowStore } from "@/store/workflow-store";

import { WorkflowShell } from "./workflow-shell";

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
  const { template, channels, setTemplate, updateChannelState } = useWorkflowStore();
  const [pendingAction, setPendingAction] = useState<{
    channelId: ChannelId;
    targetState: ChannelState;
  } | null>(null);

  const templateLength = template.length;
  const isWorkflowComplete = channels.every((channel) => channel.state === "done");

  const handleUpdateChannelState = (id: ChannelId, state: ChannelState) => {
    updateChannelState(id, state);
    setPendingAction(null);
  };

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
                onChange={(event) => setTemplate(event.target.value)}
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
                          onClick={() => handleUpdateChannelState(channel.id, targetState)}
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
