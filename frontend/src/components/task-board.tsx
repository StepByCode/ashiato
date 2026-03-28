"use client";

import { FormEvent, useCallback, useEffect, useRef, useState } from "react";
import { CheckCheck, ChevronDown, ExternalLink, Loader2, Plus, RefreshCw, RotateCcw } from "lucide-react";

import { Button, buttonVariants } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import { apiFetch } from "@/lib/api";
import { useAuth } from "@/lib/auth-context";
import { usePeriod } from "@/lib/period-context";

import { WorkflowShell } from "./workflow-shell";

type TaskState = "in_progress" | "done" | "approved";
type Task = {
  id: string;
  title: string;
  assigneeId: string;
  assigneeName: string;
  state: TaskState;
  url: string;
};

type Member = {
  id: string;
  name: string;
  email: string;
};

type MeResponse = {
  user?: {
    id?: string;
  };
};

const fixedTasks = [
  { title: "イベント名", assigneeRequired: false },
  { title: "connpass URL", assigneeRequired: true },
  { title: "Place", assigneeRequired: true },
] as const;

const fixedTaskOrder = new Map<string, number>(fixedTasks.map((task, index) => [task.title, index]));

const taskStateLabels: Record<TaskState, string> = {
  in_progress: "in progress",
  done: "Done",
  approved: "Approved",
};

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
    cardClassName: "border-[var(--success)] bg-[var(--success)]",
  },
  approved: {
    cardClassName: "border-emerald-300/60 bg-[var(--surface-success)]",
  },
};

function sortTasks(tasks: Task[]): Task[] {
  return [...tasks].sort((left, right) => {
    const leftIndex = fixedTaskOrder.get(left.title);
    const rightIndex = fixedTaskOrder.get(right.title);
    const leftFixed = leftIndex !== undefined;
    const rightFixed = rightIndex !== undefined;

    if (leftFixed && rightFixed) return leftIndex - rightIndex;
    if (leftFixed) return -1;
    if (rightFixed) return 1;

    return left.title.localeCompare(right.title, "ja");
  });
}

function tasksEqual(left: Task[], right: Task[]): boolean {
  if (left.length !== right.length) return false;
  return left.every((task, index) => {
    const other = right[index];
    return (
      task.id === other.id &&
      task.title === other.title &&
      task.assigneeId === other.assigneeId &&
      task.assigneeName === other.assigneeName &&
      task.state === other.state &&
      task.url === other.url
    );
  });
}

export function TaskBoard() {
  const { getIdToken } = useAuth();
  const { selectedPeriod } = usePeriod();
  const [tasks, setTasks] = useState<Task[]>([]);
  const [members, setMembers] = useState<Member[]>([]);
  const [currentMemberId, setCurrentMemberId] = useState("");
  const [loadingTasks, setLoadingTasks] = useState(true);
  const [isCreateOpen, setCreateOpen] = useState(false);
  const [newTitle, setNewTitle] = useState("");
  const [newAssigneeId, setNewAssigneeId] = useState("");
  const [savedTaskUrls, setSavedTaskUrls] = useState<Record<string, string>>({});
  const [pendingAction, setPendingAction] = useState<{
    taskId: string;
    type: "approve" | "mark_done" | "mark_in_progress";
  } | null>(null);
  const saveNoticeTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const [refreshing, setRefreshing] = useState(false);
  const [showSaveNotice, setShowSaveNotice] = useState(false);

  const flashSaveNotice = useCallback(() => {
    setShowSaveNotice(true);
    if (saveNoticeTimer.current) clearTimeout(saveNoticeTimer.current);
    saveNoticeTimer.current = setTimeout(() => setShowSaveNotice(false), 3000);
  }, []);

  const fetchTasks = useCallback(async (year: number, month: number, options?: { silent?: boolean }) => {
    const silent = options?.silent ?? false;
    if (!silent) {
      setLoadingTasks(true);
    }
    try {
      const res = await apiFetch(`/api/v1/tasks?year=${year}&month=${month}`, null);
      if (res.ok) {
        const data = await res.json();
        const fetched = sortTasks((data.tasks ?? []).map((t: Record<string, string>) => ({
          id: t.id,
          title: t.title,
          assigneeId: t.assigneeId ?? "",
          assigneeName: t.assigneeName ?? "",
          state: t.state as TaskState,
          url: t.url ?? "",
        })));
        setTasks((prev) => (tasksEqual(prev, fetched) ? prev : fetched));
        setSavedTaskUrls(
          fetched.reduce<Record<string, string>>((acc, task) => {
            acc[task.id] = task.url ?? "";
            return acc;
          }, {})
        );
      }
    } catch {
      // API unreachable - show empty state
    } finally {
      if (!silent) {
        setLoadingTasks(false);
      }
    }
  }, []);

  const fetchMembers = useCallback(async () => {
    try {
      const token = await getIdToken();
      if (!token) return;
      const res = await apiFetch("/api/v1/members", token);
      if (!res.ok) return;
      const data = await res.json();
      const deduped = new Map<string, Member>();
      for (const member of data.members ?? []) {
        const normalizedEmail = (member.email ?? "").trim().toLowerCase();
        const key = normalizedEmail || member.id;
        if (deduped.has(key)) continue;
        deduped.set(key, {
          id: member.id,
          name: member.name || "表示名未設定",
          email: member.email,
        });
      }
      setMembers(Array.from(deduped.values()));
    } catch {
      // ignore
    }
  }, [getIdToken]);

  const fetchMe = useCallback(async () => {
    try {
      const token = await getIdToken();
      if (!token) return;
      const res = await apiFetch("/api/v1/me", token);
      if (!res.ok) return;
      const data = (await res.json()) as MeResponse;
      setCurrentMemberId(data.user?.id ?? "");
    } catch {
      // ignore
    }
  }, [getIdToken]);

  useEffect(() => {
    void fetchTasks(selectedPeriod.year, selectedPeriod.month);
  }, [fetchTasks, selectedPeriod.year, selectedPeriod.month]);

  useEffect(() => {
    void fetchMembers();
    void fetchMe();
  }, [fetchMe, fetchMembers]);

  const handleManualRefresh = useCallback(async () => {
    setRefreshing(true);
    try {
      await Promise.all([
        fetchTasks(selectedPeriod.year, selectedPeriod.month, { silent: true }),
        fetchMembers(),
        fetchMe(),
      ]);
    } finally {
      setRefreshing(false);
    }
  }, [fetchMe, fetchMembers, fetchTasks, selectedPeriod.month, selectedPeriod.year]);

  const handleCreateTask = async (event: FormEvent) => {
    event.preventDefault();
    if (!newTitle.trim() || !newAssigneeId) return;

    try {
      const token = await getIdToken();
      const res = await apiFetch("/api/v1/tasks", token, {
        method: "POST",
        body: JSON.stringify({
          title: newTitle.trim(),
          assigneeId: newAssigneeId,
          year: selectedPeriod.year,
          month: selectedPeriod.month,
        }),
      });
      if (res.ok) {
        const data = await res.json();
        const t = data.task;
        setTasks((prev) => sortTasks([
          ...prev,
          {
            id: t.id,
            title: t.title,
            assigneeId: t.assigneeId ?? "",
            assigneeName: t.assigneeName ?? "",
            state: t.state as TaskState,
            url: t.url ?? "",
          },
        ]));
        setSavedTaskUrls((prev) => ({ ...prev, [t.id]: t.url ?? "" }));
        setNewTitle("");
        setNewAssigneeId("");
        setCreateOpen(false);
        void fetchTasks(selectedPeriod.year, selectedPeriod.month, { silent: true });
      }
    } catch {
      // ignore network errors
    }
  };

  const updateAssignee = async (id: string, assigneeId: string) => {
    const member = members.find((entry) => entry.id === assigneeId);
    setTasks((prev) =>
      prev.map((task) =>
        task.id === id
          ? { ...task, assigneeId, assigneeName: assigneeId ? (member?.name ?? "表示名未設定") : "" }
          : task
      )
    );
    try {
      const token = await getIdToken();
      const res = await apiFetch(`/api/v1/tasks/${id}/assignee`, token, {
        method: "PATCH",
        body: JSON.stringify({ assigneeId }),
      });
      if (res.ok) {
        const data = await res.json();
        const t = data.task;
        setTasks((prev) =>
          prev.map((task) =>
            task.id === id
              ? {
                  ...task,
                  assigneeId: t.assigneeId ?? "",
                  assigneeName: t.assigneeName ?? "",
                }
              : task
          )
        );
      }
    } catch {
      // ignore network errors
    }
  };

  const updateUrl = useCallback((id: string, url: string) => {
    setTasks((prev) => prev.map((task) => (task.id === id ? { ...task, url } : task)));
  }, []);

  const saveUrlOnBlur = useCallback(async (id: string, url: string) => {
    const savedUrl = savedTaskUrls[id] ?? "";
    if (url.trim() === savedUrl.trim()) return;

    try {
      const token = await getIdToken();
      const res = await apiFetch(`/api/v1/tasks/${id}/url`, token, {
        method: "PATCH",
        body: JSON.stringify({ url }),
      });
      if (res.ok) {
        const data = await res.json();
        const t = data.task;
        setTasks((prev) =>
          prev.map((task) => (task.id === id ? { ...task, url: t.url ?? "" } : task))
        );
        setSavedTaskUrls((prev) => ({ ...prev, [id]: t.url ?? "" }));
        flashSaveNotice();
      }
    } catch {
      // ignore network errors
    }
  }, [flashSaveNotice, getIdToken, savedTaskUrls]);

  const changeTaskState = async (id: string, state: TaskState) => {
    try {
      const token = await getIdToken();
      const res = await apiFetch(`/api/v1/tasks/${id}/state`, token, {
        method: "PATCH",
        body: JSON.stringify({ state }),
      });
      if (res.ok) {
        const data = await res.json();
        const t = data.task;
        setTasks((prev) =>
          prev.map((task) => (task.id === id ? { ...task, state: t.state as TaskState } : task))
        );
        void fetchTasks(selectedPeriod.year, selectedPeriod.month, { silent: true });
      }
    } catch {
      // ignore network errors
    }
    setPendingAction(null);
  };

  const toJumpHref = (url: string) => {
    if (!url.trim()) return "";
    if (/^https?:\/\//i.test(url)) return url;
    return `https://${url}`;
  };

  const runPendingAction = (taskId: string, type: "approve" | "mark_done" | "mark_in_progress") => {
    const nextState: TaskState =
      type === "approve" ? "approved" : type === "mark_in_progress" ? "in_progress" : "done";
    changeTaskState(taskId, nextState);
  };

  if (loadingTasks) {
    return (
      <WorkflowShell activeStep="作成">
        <div className="flex items-center justify-center py-20 text-muted-foreground">
          <Loader2 className="mr-2 size-5 animate-spin" />
          読み込み中...
        </div>
      </WorkflowShell>
    );
  }

  return (
    <WorkflowShell activeStep="作成">
      <section className="grid gap-4">
        {showSaveNotice ? (
          <div className="fixed left-1/2 top-5 z-50 -translate-x-1/2 rounded-full border border-emerald-300/60 bg-emerald-50 px-6 py-3 text-base font-semibold text-emerald-900 shadow-lg">
            保存しました
          </div>
        ) : null}
        <div className="flex justify-end">
          <Button
            type="button"
            variant="outline"
            className="min-h-11 rounded-full px-5 text-sm font-semibold shadow-sm"
            onClick={handleManualRefresh}
            disabled={refreshing}
          >
            <RefreshCw className={cn("size-4", refreshing && "animate-spin")} />
            {refreshing ? "更新中..." : "手動更新"}
          </Button>
        </div>

        {tasks.map((task) => {
          const isEventNameTask = task.title === "イベント名";
          const hasAssignee = Boolean(task.assigneeId);
          const isAssigneeSelf = Boolean(currentMemberId) && currentMemberId === task.assigneeId;
          const showApprove =
            !isEventNameTask && hasAssignee && !isAssigneeSelf && (task.state === "done" || task.state === "approved");
          const isApproved = task.state === "approved";
          const showDoneAction = !isEventNameTask && hasAssignee && isAssigneeSelf && !isApproved;
          const isDoneLike = task.state !== "in_progress";
          const isApproveConfirmOpen =
            pendingAction?.taskId === task.id && pendingAction.type === "approve" && !isApproved;
          const isDoneConfirmOpen =
            pendingAction?.taskId === task.id &&
            (pendingAction.type === "mark_done" || pendingAction.type === "mark_in_progress");
          const taskStateMeta = isEventNameTask ? stateMeta.in_progress : stateMeta[task.state];
          const jumpHref = toJumpHref(task.url);
          const pendingCopy =
            pendingAction?.type === "approve"
              ? "Approveしますか？"
              : `${taskStateLabels[pendingAction?.type === "mark_in_progress" ? "in_progress" : "done"]} に変更しますか？`;

          const fixedTask = fixedTasks.find((entry) => entry.title === task.title);
          const allowEmptyAssignee = !fixedTask?.assigneeRequired;
          const assigneeLabel = task.assigneeName || members.find((member) => member.id === task.assigneeId)?.name || "";

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
                      {!isEventNameTask && assigneeLabel ? (
                        <span className="rounded-full border border-border/70 bg-background/85 px-3 py-1 text-sm font-medium text-foreground/80 shadow-sm">
                          担当: {assigneeLabel}
                        </span>
                      ) : null}
                      {!isEventNameTask ? (
                        <div className="relative w-full sm:max-w-60">
                          <select
                            id={`${task.id}-assignee`}
                            className="h-12 w-full appearance-none rounded-full border border-border/70 bg-background/85 px-4 pr-10 text-base font-medium shadow-sm outline-none transition focus:border-primary focus:ring-4 focus:ring-primary/15"
                            aria-label={`${task.title}担当者`}
                            value={task.assigneeId}
                            onChange={(e) => updateAssignee(task.id, e.target.value)}
                          >
                            {allowEmptyAssignee ? <option value="">担当者なし</option> : <option value="">担当者</option>}
                            {members.map((member) => (
                              <option key={member.id} value={member.id}>
                                {member.name}
                              </option>
                            ))}
                          </select>
                          <ChevronDown className="pointer-events-none absolute right-4 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
                        </div>
                      ) : null}
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
                          Approve
                        </Button>

                        {isApproveConfirmOpen ? (
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
                                onClick={() => runPendingAction(task.id, "approve")}
                              >
                                はい
                              </Button>
                            </div>
                          </div>
                        ) : null}
                      </>
                    ) : showDoneAction ? (
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
                          {isDoneLike ? "Doneを取り消す" : "Done"}
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
                      !isEventNameTask ? (
                        <div className="flex min-h-12 w-full items-center justify-center rounded-full border border-border/70 px-5 text-sm font-semibold text-muted-foreground">
                          {isApproved ? "このタスクは完了済みです。" : hasAssignee ? "担当者が対応中です" : "担当者を設定してください"}
                        </div>
                      ) : null
                    )}
                  </div>
                </div>

                {fixedTask ? (
                  <p className="text-xs text-muted-foreground">
                    固定タスク
                    {fixedTask.assigneeRequired ? " / 担当者必須" : ""}
                  </p>
                ) : null}

                <div className={cn("grid gap-3 lg:items-end", isEventNameTask ? "lg:grid-cols-1" : "lg:grid-cols-[minmax(0,1fr)_auto]")}>
                  <div>
                    <Input
                      id={`${task.id}-url`}
                      className="h-12 rounded-2xl border-border/70 bg-background/85 px-4 text-[color:var(--url-color)] shadow-sm placeholder:text-[color:var(--url-color)] placeholder:opacity-60"
                      type={isEventNameTask ? "text" : "url"}
                      placeholder={isEventNameTask ? "イベント名" : "URL"}
                      value={task.url}
                      onChange={(e) => updateUrl(task.id, e.target.value)}
                      onBlur={(e) => void saveUrlOnBlur(task.id, e.target.value)}
                    />
                  </div>

                  {!isEventNameTask && jumpHref ? (
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
                  ) : !isEventNameTask ? (
                    <div className="flex min-h-12 items-center justify-center rounded-2xl border border-dashed border-border/70 px-5 text-sm font-medium text-muted-foreground">
                      URL
                    </div>
                  ) : null}
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
                        value={newAssigneeId}
                        onChange={(e) => setNewAssigneeId(e.target.value)}
                      >
                        <option value="">担当者</option>
                        {members.map((member) => (
                          <option key={member.id} value={member.id}>
                            {member.name}
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
    </WorkflowShell>
  );
}
