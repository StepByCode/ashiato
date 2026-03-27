import { create } from "zustand";

export type Owner = "kido" | "kitahara" | "sogo" | "nakai";
export type TaskState = "in_progress" | "done" | "approved";

export type Task = {
  id: string;
  title: string;
  owner: Owner;
  state: TaskState;
  url: string;
};

const initialTasks: Task[] = [
  { id: "connpass", title: "connpass", owner: "kido", state: "in_progress", url: "" },
  { id: "figma", title: "Figma", owner: "kitahara", state: "in_progress", url: "" },
  { id: "place", title: "Place", owner: "nakai", state: "done", url: "" },
];

type TaskStore = {
  tasks: Task[];
  addTask: (task: Task) => void;
  updateOwner: (id: Task["id"], owner: Owner) => void;
  updateUrl: (id: Task["id"], url: string) => void;
  setTaskState: (id: Task["id"], state: TaskState) => void;
  resetTasks: () => void;
};

export const useTaskStore = create<TaskStore>((set) => ({
  tasks: initialTasks,
  addTask: (task) => set((state) => ({ tasks: [...state.tasks, task] })),
  updateOwner: (id, owner) =>
    set((state) => ({
      tasks: state.tasks.map((task) => (task.id === id ? { ...task, owner } : task)),
    })),
  updateUrl: (id, url) =>
    set((state) => ({
      tasks: state.tasks.map((task) => (task.id === id ? { ...task, url } : task)),
    })),
  setTaskState: (id, state) =>
    set((prev) => ({
      tasks: prev.tasks.map((task) => (task.id === id ? { ...task, state } : task)),
    })),
  resetTasks: () => set({ tasks: initialTasks }),
}));
