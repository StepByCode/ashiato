"use client";

import { FormEvent, useState } from "react";
import { CheckCheck, ChevronDown, ExternalLink, Plus, RotateCcw } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button, buttonVariants } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";

type Owner = "kido" | "kitahara" | "sogo" | "nakai";
type TaskState = "in_progress" | "done" | "approved";

type Task = {
  id: string;
  title: string;
  owner: Owner;
  state: TaskState;
  url: string;
};

const owners: Owner[] = ["kido", "kitahara", "sogo", "nakai"];
const months = ["2026.4月", "2026.3月", "2026.2月", "2026.1月"] as const;
const progressSteps = ["定例", "作成", "広報"] as const;
const currentMonth = "2026.2月";

const initialTasks: Task[] = [
  { id: "connpass", title: "connpass", owner: "kido", state: "in_progress", url: "" },
  { id: "figma", title: "Figma", owner: "kitahara", state: "in_progress", url: "" },
  { id: "place", title: "Place", owner: "nakai", state: "done", url: "" },
];

const stateMeta: Record<
  TaskState,
  {
    cardClassName: string;
  }
> = {
  in_progress: {
    cardClassName: "border-border/70 bg-card/95",
  },
  done: {
    cardClassName: "border-sky-300/60 bg-[var(--surface-accent)]",
  },
  approved: {
    cardClassName: "border-emerald-300/60 bg-[var(--surface-success)]",
  },
};

function makeTaskId() {
  return `task-${Date.now()}-${Math.floor(Math.random() * 100000)}`;
}

export function TaskBoard() {
  const [tasks, setTasks] = useState<Task[]>(initialTasks);
  const [isCreateOpen, setCreateOpen] = useState(false);
  const [newTitle, setNewTitle] = useState("");
  const [newOwner, setNewOwner] = useState<Owner | "">("");
  const [pendingAction, setPendingAction] = useState<{
    taskId: string;
    type: "approve" | "mark_done" | "mark_in_progress";
  } | null>(null);

  const handleCreateTask = (event: FormEvent) => {
    event.preventDefault();
    if (!newTitle.trim() || !newOwner) return;

    const task: Task = {
      id: makeTaskId(),
      title: newTitle.trim(),
      owner: newOwner,
      state: "in_progress",
      url: "",
    };

    setTasks((prev) => [...prev, task]);
    setNewTitle("");
    setNewOwner("");
    setCreateOpen(false);
  };

  const updateOwner = (id: string, owner: Owner) => {
    setTasks((prev) => prev.map((task) => (task.id === id ? { ...task, owner } : task)));
  };

  const updateUrl = (id: string, url: string) => {
    setTasks((prev) => prev.map((task) => (task.id === id ? { ...task, url } : task)));
  };

  const approveTask = (id: string) => {
    setTasks((prev) => prev.map((task) => (task.id === id ? { ...task, state: "approved" } : task)));
    setPendingAction(null);
  };

  const setTaskState = (id: string, state: TaskState) => {
    setTasks((prev) => prev.map((task) => (task.id === id ? { ...task, state } : task)));
  };

  const toJumpHref = (url: string) => {
    if (!url.trim()) return "";
    if (/^https?:\/\//i.test(url)) return url;
    return `https://${url}`;
  };

  const runPendingAction = (taskId: string, type: "approve" | "mark_done" | "mark_in_progress") => {
    if (type === "approve") {
      approveTask(taskId);
      return;
    }

    setTaskState(taskId, type === "mark_in_progress" ? "in_progress" : "done");
    setPendingAction(null);
  };

  return (
    <div className="min-h-screen px-4 py-4 sm:px-6 sm:py-6">
      <div className="mx-auto flex max-w-7xl flex-col gap-4 lg:flex-row">
        <aside className="hidden w-full max-w-64 shrink-0 rounded-[2rem] border border-border/70 bg-card/80 p-4 shadow-lg shadow-black/5 backdrop-blur lg:flex lg:flex-col">
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
        </aside>

        <main className="flex-1 rounded-[2rem] border border-border/70 bg-card/80 p-4 shadow-lg shadow-black/5 backdrop-blur sm:p-6 lg:p-8">
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
                    {progressSteps.map((step) => {
                      const isActive = step === "作成";

                      return (
                        <div key={step} className="relative z-10 flex flex-col items-center gap-2 text-center">
                          <div
                            className={cn(
                              "flex size-12 items-center justify-center rounded-full border-2 bg-background text-sm font-semibold shadow-sm sm:size-14",
                              isActive
                                ? "border-primary bg-primary text-primary-foreground"
                                : "border-border/80 text-foreground"
                            )}
                          >
                            {step}
                          </div>
                        </div>
                      );
                    })}
                  </div>
                </CardContent>
              </Card>
            </header>

            <section className="grid gap-4">
              {tasks.map((task) => {
                const showApprove = task.owner !== "nakai" && (task.state === "done" || task.state === "approved");
                const isApproved = task.state === "approved";
                const isDoneLike = task.state !== "in_progress";
                const isApproveConfirmOpen =
                  pendingAction?.taskId === task.id && pendingAction.type === "approve" && !isApproved;
                const isDoneConfirmOpen =
                  pendingAction?.taskId === task.id &&
                  (pendingAction.type === "mark_done" || pendingAction.type === "mark_in_progress");
                const taskStateMeta = stateMeta[task.state];
                const jumpHref = toJumpHref(task.url);
                const pendingCopy =
                  pendingAction?.type === "mark_in_progress"
                    ? "in progressに変更しますか"
                    : "Doneに変更しますか";

                return (
                  <Card
                    key={task.id}
                    className={cn("overflow-hidden rounded-[1.75rem] border-2 shadow-sm", taskStateMeta.cardClassName)}
                  >
                    <CardContent className="space-y-5 p-5 sm:p-6">
                      <div className="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
                        <div className="min-w-0 flex-1 space-y-4">
                          <div className="flex flex-wrap items-center gap-3">
                            <h3 className="min-w-0 break-words text-2xl font-semibold tracking-tight sm:text-3xl">
                              {task.title}
                            </h3>
                            <div className="relative w-full sm:max-w-60">
                              <select
                                id={`${task.id}-owner`}
                                className="h-12 w-full appearance-none rounded-full border border-border/70 bg-background/85 px-4 pr-10 text-base font-medium shadow-sm outline-none transition focus:border-primary focus:ring-4 focus:ring-primary/15"
                                aria-label={`${task.title}担当者`}
                                value={task.owner}
                                onChange={(e) => updateOwner(task.id, e.target.value as Owner)}
                              >
                                {owners.map((owner) => (
                                  <option key={owner} value={owner}>
                                    {owner}
                                  </option>
                                ))}
                              </select>
                              <ChevronDown className="pointer-events-none absolute right-4 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
                            </div>
                          </div>
                        </div>

                        <div className="flex w-full flex-col gap-2 xl:min-w-[240px] xl:max-w-[280px] xl:items-end">
                          {showApprove ? (
                            <>
                              <Button
                                type="button"
                                variant={isApproved ? "secondary" : "default"}
                                className={cn(
                                  "min-h-12 w-full rounded-full px-5 text-base font-semibold shadow-sm",
                                  isApproved && "bg-emerald-100 text-emerald-900 hover:bg-emerald-100"
                                )}
                                disabled={isApproved}
                                onClick={() => {
                                  if (!isApproved) setPendingAction({ taskId: task.id, type: "approve" });
                                }}
                              >
                                <CheckCheck className="size-4" />
                                {isApproved ? "Approved" : "Approve"}
                              </Button>

                              {isApproveConfirmOpen ? (
                                <div className="grid w-full gap-2 rounded-[1.25rem] border border-border/70 bg-popover/95 p-3 text-sm font-medium shadow-sm">
                                  <span className="whitespace-nowrap text-foreground">Approve?</span>
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
                                      onClick={() => runPendingAction(task.id, "approve")}
                                    >
                                      はい
                                    </Button>
                                  </div>
                                </div>
                              ) : null}
                            </>
                          ) : task.owner === "nakai" ? (
                            <>
                              <Button
                                type="button"
                                variant={isDoneLike ? "secondary" : "default"}
                                className="min-h-12 w-full rounded-full px-5 text-base font-semibold shadow-sm"
                                onClick={() => {
                                  if (isDoneLike) {
                                    setPendingAction({ taskId: task.id, type: "mark_in_progress" });
                                  } else {
                                    setPendingAction({ taskId: task.id, type: "mark_done" });
                                  }
                                }}
                              >
                                {isDoneLike ? <RotateCcw className="size-4" /> : <CheckCheck className="size-4" />}
                                {isDoneLike ? "Done" : "in progress"}
                              </Button>

                              {isDoneConfirmOpen ? (
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
                                      onClick={() =>
                                        runPendingAction(
                                          task.id,
                                          pendingAction?.type === "mark_in_progress" ? "mark_in_progress" : "mark_done"
                                        )
                                      }
                                    >
                                      はい
                                    </Button>
                                  </div>
                                </div>
                              ) : null}
                            </>
                          ) : (
                            <Badge
                              variant="outline"
                              className="flex min-h-12 w-full items-center justify-center rounded-full px-5 text-sm font-semibold"
                            >
                              in progress
                            </Badge>
                          )}
                        </div>
                      </div>

                      <div className="grid gap-3 lg:grid-cols-[minmax(0,1fr)_auto] lg:items-end">
                        <div>
                          <Input
                            id={`${task.id}-url`}
                            className="h-12 rounded-2xl border-border/70 bg-background/85 px-4 text-[color:var(--url-color)] shadow-sm placeholder:text-[color:var(--url-color)] placeholder:opacity-60"
                            type="url"
                            placeholder="URL"
                            value={task.url}
                            onChange={(e) => updateUrl(task.id, e.target.value)}
                          />
                        </div>

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
                            URL
                            <ExternalLink className="size-4" />
                          </a>
                        ) : (
                          <div className="flex min-h-12 items-center justify-center rounded-2xl border border-dashed border-border/70 px-5 text-sm font-medium text-muted-foreground">
                            URL
                          </div>
                        )}
                      </div>
                    </CardContent>
                  </Card>
                );
              })}

              <Card className="rounded-[1.75rem] border-border/70 bg-[var(--surface-panel)]">
                <CardContent className="p-5 sm:p-6">
                  <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
                    <h3 className="text-xl font-semibold tracking-tight">追加</h3>

                    <Button
                      type="button"
                      variant={isCreateOpen ? "secondary" : "default"}
                      className="min-h-12 rounded-full px-5 text-base font-semibold shadow-sm"
                      aria-expanded={isCreateOpen}
                      aria-controls="create-form-box"
                      onClick={() => setCreateOpen((prev) => !prev)}
                    >
                      <Plus className={cn("size-4 transition-transform", isCreateOpen && "rotate-45")} />
                      {isCreateOpen ? "閉じる" : "開く"}
                    </Button>
                  </div>

                  {isCreateOpen ? (
                    <div id="create-form-box" className="pt-6">
                      <form
                        className="grid gap-4 lg:grid-cols-[minmax(0,1.5fr)_minmax(220px,0.9fr)_auto] lg:items-end"
                        onSubmit={handleCreateTask}
                      >
                        <div>
                          <Input
                            id="create-task-title"
                            className="h-12 rounded-2xl border-border/70 bg-background/85 px-4 shadow-sm"
                            type="text"
                            placeholder="タスク名"
                            required
                            value={newTitle}
                            onChange={(e) => setNewTitle(e.target.value)}
                          />
                        </div>

                        <div>
                          <div className="relative">
                            <select
                              id="create-task-owner"
                              className="h-12 w-full appearance-none rounded-2xl border border-border/70 bg-background/85 px-4 pr-10 text-base font-medium shadow-sm outline-none transition focus:border-primary focus:ring-4 focus:ring-primary/15"
                              aria-label="新規作成の担当者"
                              required
                              value={newOwner}
                              onChange={(e) => setNewOwner(e.target.value as Owner | "")}
                            >
                              <option value="">担当者</option>
                              {owners.map((owner) => (
                                <option key={owner} value={owner}>
                                  {owner}
                                </option>
                              ))}
                            </select>
                            <ChevronDown className="pointer-events-none absolute right-4 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
                          </div>
                        </div>

                        <Button type="submit" className="min-h-12 rounded-2xl px-6 text-base font-semibold shadow-sm">
                          作成
                        </Button>
                      </form>
                    </div>
                  ) : null}
                </CardContent>
              </Card>
            </section>
          </div>
        </main>
      </div>
    </div>
  );
}
