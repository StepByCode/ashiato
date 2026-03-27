"use client";

import { FormEvent, useState } from "react";
import { CheckCheck, ChevronDown, ExternalLink, Plus, RotateCcw } from "lucide-react";

import { Button, buttonVariants } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";

import { WorkflowShell } from "./workflow-shell";

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
const taskStateLabels: Record<TaskState, string> = {
  in_progress: "in progress",
  done: "Done",
  approved: "Approved",
};

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
    cardClassName: "border-[var(--success)] bg-[var(--success)] dark:border-[#5E936C] dark:bg-[#3E5F44]",
  },
  approved: {
    cardClassName: "border-emerald-300/60 bg-[var(--surface-success)] dark:border-[#5E936C] dark:bg-[#3E5F44]",
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
    <WorkflowShell activeStep="作成">
      <section className="grid gap-2.5 lg:gap-2">
        {tasks.map((task) => {
          const showApprove = task.owner !== "nakai" && (task.state === "done" || task.state === "approved");
          const isApproved = task.state === "approved";
          const isDoneLike = task.state !== "in_progress";
          const isApproveConfirmOpen =
            pendingAction?.taskId === task.id && pendingAction.type === "approve" && !isApproved;
          const isDoneConfirmOpen =
            pendingAction?.taskId === task.id &&
            (pendingAction.type === "mark_done" || pendingAction.type === "mark_in_progress");
          const needsWideActionPanel = isApproveConfirmOpen || isDoneConfirmOpen;
          const taskStateMeta = stateMeta[task.state];
          const jumpHref = toJumpHref(task.url);
          const pendingCopy =
            pendingAction?.type === "approve"
              ? "Approveしますか？"
              : `（${
                  taskStateLabels[pendingAction?.type === "mark_in_progress" ? "in_progress" : "done"]
                }）に変更しますか？`;

          return (
            <Card
              key={task.id}
              className={cn("overflow-hidden rounded-[1.75rem] border-2 shadow-sm", taskStateMeta.cardClassName)}
            >
              <CardContent className="space-y-2.5 p-3.5 sm:p-4">
                <div className="flex flex-col gap-2 lg:flex-row lg:items-start lg:justify-between">
                  <div className="min-w-0 flex-1 space-y-2.5">
                    <div className="flex items-center gap-3">
                      <h3 className="min-w-0 flex-1 break-words text-lg font-semibold tracking-tight sm:text-xl">
                        {task.title}
                      </h3>
                      <div className="relative ml-auto w-32 shrink-0 sm:w-40 lg:ml-auto lg:w-36">
                        <select
                          id={`${task.id}-owner`}
                          className="h-10 w-full appearance-none rounded-full border border-border/70 bg-background/85 px-4 pr-10 text-sm font-medium shadow-sm outline-none transition focus:border-primary focus:ring-4 focus:ring-primary/15 lg:px-3 lg:pr-8 lg:text-xs"
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

                  <div
                    className={cn(
                      "flex w-full flex-col gap-1.5 lg:items-end",
                      needsWideActionPanel
                        ? "lg:min-w-[200px] lg:max-w-[220px]"
                        : "lg:w-36 lg:min-w-0 lg:max-w-36"
                    )}
                  >
                    {showApprove ? (
                      <>
                        <Button
                          type="button"
                          variant={isApproved ? "secondary" : "default"}
                          className={cn(
                            "h-10 w-full rounded-full px-4 text-sm font-semibold shadow-sm",
                            !isApproved &&
                              "dark:bg-[#5E936C] dark:text-white dark:hover:bg-[#5E936C]",
                            isApproved &&
                              "bg-emerald-100 text-emerald-900 hover:bg-emerald-100 dark:bg-[#5E936C] dark:text-white dark:hover:bg-[#5E936C]"
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
                          <div className="grid w-full gap-1.5 rounded-[1.1rem] border border-border/70 bg-popover/95 p-2 text-xs font-medium shadow-sm">
                            <span className="whitespace-nowrap text-foreground">{pendingCopy}</span>
                            <div className="flex flex-wrap gap-2">
                              <Button
                                type="button"
                                variant="outline"
                                className="h-8 rounded-full px-3 text-xs font-semibold"
                                onClick={() => setPendingAction(null)}
                              >
                                いいえ
                              </Button>
                              <Button
                                type="button"
                                className="h-8 rounded-full px-3 text-xs font-semibold"
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
                          className={cn(
                            "h-10 w-full rounded-full px-4 text-sm font-semibold shadow-sm lg:w-36 lg:px-3 lg:text-xs",
                            isDoneLike && "dark:bg-[#5E936C] dark:text-white dark:hover:bg-[#5E936C]",
                            !isDoneLike &&
                              "dark:border dark:border-border/70 dark:bg-[var(--surface-button)] dark:text-foreground dark:hover:bg-[var(--surface-button-subtle)]"
                          )}
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
                          <div className="grid w-full gap-1.5 rounded-[1.1rem] border border-border/70 bg-popover/95 p-2 text-xs font-medium shadow-sm">
                            <span className="whitespace-nowrap text-foreground">{pendingCopy}</span>
                            <div className="flex flex-wrap gap-2">
                              <Button
                                type="button"
                                variant="outline"
                                className="h-8 rounded-full px-3 text-xs font-semibold"
                                onClick={() => setPendingAction(null)}
                              >
                                いいえ
                              </Button>
                              <Button
                                type="button"
                                className="h-8 rounded-full px-3 text-xs font-semibold"
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
                      <div className="flex h-10 w-full items-center justify-center rounded-full border border-border/70 px-4 text-xs font-semibold lg:w-36 lg:px-3">
                        in progress
                      </div>
                    )}
                  </div>
                </div>

                <div className="grid gap-2 lg:grid-cols-[minmax(0,1fr)_auto] lg:items-end">
                  <div>
                    <Input
                      id={`${task.id}-url`}
                      className="h-10 rounded-xl border-border/70 bg-background/85 px-4 text-sm text-[color:var(--url-color)] shadow-sm placeholder:text-[color:var(--url-color)] placeholder:opacity-60"
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
                        "h-10 rounded-xl px-4 text-sm font-semibold shadow-sm dark:border-[#5E936C] dark:bg-[#5E936C] dark:text-white dark:hover:bg-[#5E936C]"
                      )}
                      href={jumpHref}
                      target="_blank"
                      rel="noreferrer"
                    >
                      URL
                      <ExternalLink className="size-4" />
                    </a>
                  ) : (
                    <div className="flex h-10 items-center justify-center rounded-xl border border-dashed border-border/70 px-4 text-xs font-medium text-muted-foreground">
                      URL
                    </div>
                  )}
                </div>
              </CardContent>
            </Card>
          );
        })}

        <Card className="rounded-[1.75rem] border-border/70 bg-[var(--surface-panel)] max-[900px]:hidden">
          <CardContent className="p-3.5 sm:p-4">
            <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
              <h3 className="text-lg font-semibold tracking-tight">追加</h3>

              <Button
                type="button"
                variant={isCreateOpen ? "secondary" : "default"}
                className="h-10 rounded-full px-4 text-sm font-semibold shadow-sm max-[900px]:hidden dark:border dark:border-border/70 dark:bg-[var(--surface-button)] dark:text-foreground dark:hover:bg-[var(--surface-button-subtle)]"
                aria-expanded={isCreateOpen}
                aria-controls="create-form-box"
                onClick={() => setCreateOpen((prev) => !prev)}
              >
                <Plus className={cn("size-4 transition-transform", isCreateOpen && "rotate-45")} />
                {isCreateOpen ? "閉じる" : "開く"}
              </Button>
            </div>

            {isCreateOpen ? (
              <div id="create-form-box" className="pt-3">
                <form
                  className="grid gap-2.5 lg:grid-cols-[minmax(0,1.5fr)_minmax(220px,0.9fr)_auto] lg:items-end"
                  onSubmit={handleCreateTask}
                >
                  <div>
                    <Input
                      id="create-task-title"
                      className="h-10 rounded-xl border-border/70 bg-background/85 px-4 text-sm shadow-sm"
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
                        className="h-10 w-full appearance-none rounded-xl border border-border/70 bg-background/85 px-4 pr-10 text-sm font-medium shadow-sm outline-none transition focus:border-primary focus:ring-4 focus:ring-primary/15"
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

                  <Button
                    type="submit"
                    className="h-10 rounded-xl px-5 text-sm font-semibold shadow-sm dark:border dark:border-border/70 dark:bg-[var(--surface-button)] dark:text-foreground dark:hover:bg-[var(--surface-button-subtle)]"
                  >
                    作成
                  </Button>
                </form>
              </div>
            ) : null}
          </CardContent>
        </Card>
      </section>
    </WorkflowShell>
  );
}
