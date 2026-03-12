"use client";

import { FormEvent, useState } from "react";

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

const initialTasks: Task[] = [
  { id: "connpass", title: "connpass", owner: "kido", state: "in_progress", url: "" },
  { id: "figma", title: "Figma", owner: "kitahara", state: "in_progress", url: "" },
  { id: "place", title: "Place", owner: "nakai", state: "done", url: "" },
];

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
    type: "approve" | "mark_done";
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

  return (
    <div className="app">
      <aside className="month-sidebar">
        {(["2026.4月", "2026.3月", "2026.2月", "2026.1月"] as const).map((month) => (
          <button key={month} className={`month-btn ${month === "2026.2月" ? "active" : ""}`} type="button">
            {month}
          </button>
        ))}
      </aside>

      <main className="main">
        <div className="mobile-top">
          <strong>Ashiato</strong>
          <button className="menu-btn" type="button">
            メニュー
          </button>
        </div>

        <header className="progress-header status-footer">
          <div className="footer-line" />
          <div className="footer-steps">
            <div className="step">定例</div>
            <div className="step active">作成</div>
            <div className="step">広報</div>
          </div>
        </header>

        <section className="content">
          {tasks.map((task) => {
            const showApprove = task.owner !== "nakai" && (task.state === "done" || task.state === "approved");
            const isApproved = task.state === "approved";
            const isDoneLike = task.state !== "in_progress";
            const isApproveConfirmOpen = pendingAction?.taskId === task.id && pendingAction.type === "approve" && !isApproved;
            const isDoneConfirmOpen = pendingAction?.taskId === task.id && pendingAction.type === "mark_done";
            const cardStateClass = task.state === "in_progress" ? "in-progress" : "done";
            const jumpHref = toJumpHref(task.url);

            return (
              <article key={task.id} className={`task-card ${cardStateClass}`}>
                <div className="task-top">
                  <div className="task-left">
                    <h2 className="task-title">{task.title}</h2>
                    <select
                      className="select-like"
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
                  </div>

                  {showApprove ? (
                    <div className="approve-wrap">
                      <button
                        className={`action-btn ${isApproved ? "approved" : ""}`}
                        type="button"
                        disabled={isApproved}
                        onClick={() => {
                          if (!isApproved) setPendingAction({ taskId: task.id, type: "approve" });
                        }}
                      >
                        {isApproved ? "Approved" : "Approve"}
                      </button>
                      <div className={`approve-confirm ${isApproveConfirmOpen ? "show" : ""}`}>
                        <span>Approveしますか？</span>
                        <button className="confirm-btn" type="button" onClick={() => approveTask(task.id)}>
                          はい
                        </button>
                        <button className="confirm-btn" type="button" onClick={() => setPendingAction(null)}>
                          いいえ
                        </button>
                      </div>
                    </div>
                  ) : task.owner === "nakai" ? (
                    <div className="approve-wrap">
                      <button
                        className="action-btn"
                        type="button"
                        onClick={() => {
                          if (isDoneLike) {
                            setTaskState(task.id, "in_progress");
                            setPendingAction(null);
                          } else {
                            setPendingAction({ taskId: task.id, type: "mark_done" });
                          }
                        }}
                      >
                        {isDoneLike ? "in progress" : "Done"}
                      </button>
                      <div className={`approve-confirm ${isDoneConfirmOpen ? "show" : ""}`}>
                        <span>Doneに変更しますか？</span>
                        <button
                          className="confirm-btn"
                          type="button"
                          onClick={() => {
                            setTaskState(task.id, "done");
                            setPendingAction(null);
                          }}
                        >
                          はい
                        </button>
                        <button className="confirm-btn" type="button" onClick={() => setPendingAction(null)}>
                          いいえ
                        </button>
                      </div>
                    </div>
                  ) : (
                    <span className="task-status">in progress</span>
                  )}
                </div>

                <div className="url-row">
                  <input
                    className="url-field"
                    type="url"
                    placeholder="URLを入力"
                    value={task.url}
                    onChange={(e) => updateUrl(task.id, e.target.value)}
                  />
                  {jumpHref ? (
                    <a className="url-jump" href={jumpHref} target="_blank" rel="noreferrer">
                      URL
                    </a>
                  ) : (
                    <span className="url-jump disabled">URL</span>
                  )}
                </div>
              </article>
            );
          })}

          <div className="creator-toggle-row">
            <button
              className="plus-btn"
              type="button"
              aria-expanded={isCreateOpen}
              aria-controls="create-form-box"
              onClick={() => setCreateOpen((prev) => !prev)}
            >
              {isCreateOpen ? "−" : "+"}
            </button>
          </div>

          <section
            id="create-form-box"
            className={`creator-box accordion-panel ${isCreateOpen ? "open" : ""}`}
            aria-hidden={!isCreateOpen}
          >
            <form className="creator-form" onSubmit={handleCreateTask}>
              <input
                className="creator-input"
                type="text"
                placeholder="タスク名"
                required
                value={newTitle}
                onChange={(e) => setNewTitle(e.target.value)}
              />
              <select
                className="creator-select"
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
              <button className="create-btn" type="submit">
                作成
              </button>
            </form>
          </section>
        </section>
      </main>
    </div>
  );
}
