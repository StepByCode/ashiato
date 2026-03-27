"use client";

import { useRouter } from "next/navigation";
import { useAuth } from "@/lib/auth-context";
import { TaskBoard } from "@/components/task-board";

export default function HomePage() {
  const { user, loading, logout } = useAuth();
  const router = useRouter();

  if (loading) return null;
  if (!user) {
    router.replace("/login");
    return null;
  }

  return (
    <>
      <header
        style={{
          display: "flex",
          justifyContent: "flex-end",
          alignItems: "center",
          gap: "0.75rem",
          padding: "0.5rem 1rem",
          fontSize: "0.8125rem",
          color: "var(--letter-light)",
        }}
      >
        <span>{user.email}</span>
        <button
          type="button"
          onClick={() => router.push("/members")}
          style={{
            fontSize: "0.8125rem",
            padding: "0.25rem 0.75rem",
            border: "1px solid var(--regular)",
            borderRadius: "var(--button-radius)",
            background: "transparent",
            cursor: "pointer",
            color: "var(--letter)",
          }}
        >
          メンバー管理
        </button>
        <button
          type="button"
          onClick={logout}
          style={{
            fontSize: "0.8125rem",
            padding: "0.25rem 0.75rem",
            border: "1px solid var(--regular)",
            borderRadius: "var(--button-radius)",
            background: "transparent",
            cursor: "pointer",
            color: "var(--letter)",
          }}
        >
          ログアウト
        </button>
      </header>
      <TaskBoard />
    </>
  );
}
