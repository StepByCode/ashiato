"use client";

import { FormEvent, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "@/lib/auth-context";
import { apiFetch } from "@/lib/api";

const positions = ["代表", "副代表", "エンジニア", "デザイナー", "広報", "その他"] as const;

export default function ProfileSetupPage() {
  const { user, loading, getIdToken } = useAuth();
  const router = useRouter();
  const [name, setName] = useState("");
  const [position, setPosition] = useState("");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    if (!loading && !user) {
      router.replace("/login");
    }
  }, [loading, router, user]);

  if (loading || !user) return null;

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    if (!name.trim()) return;
    setError("");
    setSubmitting(true);

    try {
      const token = await getIdToken();
      if (!token) {
        setError("ログイン状態を確認できませんでした");
        return;
      }
      const res = await apiFetch(`/api/v1/profile/${user.uid}`, token, {
        method: "PATCH",
        body: JSON.stringify({ name: name.trim(), position }),
      });
      if (!res.ok) {
        const data = await res.json().catch(() => null);
        setError(data?.error?.message ?? "保存に失敗しました");
        return;
      }
      router.replace("/");
    } catch {
      setError("通信エラーが発生しました");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="flex min-h-dvh items-center justify-center bg-[var(--background)] p-4">
      <div className="w-full max-w-md rounded-[var(--card-radius)] bg-white p-8 shadow-lg dark:bg-card">
        <div className="mb-6 text-center">
          <h1 className="text-2xl font-bold text-[var(--accent)]">Backstage</h1>
          <p className="mt-1 text-sm text-muted-foreground">プロフィール登録</p>
          <p className="mt-3 text-sm text-muted-foreground">
            はじめまして！まずあなたの情報を登録してください。
          </p>
        </div>

        <form className="space-y-5" onSubmit={handleSubmit}>
          <div className="space-y-2">
            <label htmlFor="setup-name" className="text-sm font-medium text-foreground">
              名前 <span className="text-[var(--accent)]">*</span>
            </label>
            <input
              id="setup-name"
              type="text"
              required
              className="h-12 w-full rounded-xl border border-border/70 bg-background/85 px-4 text-base outline-none transition focus:border-[var(--accent)] focus:ring-2 focus:ring-[var(--accent)]/20"
              placeholder="山田 太郎"
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
          </div>

          <div className="space-y-2">
            <label htmlFor="setup-position" className="text-sm font-medium text-foreground">
              ポジション
            </label>
            <select
              id="setup-position"
              className="h-12 w-full appearance-none rounded-xl border border-border/70 bg-background/85 px-4 text-base outline-none transition focus:border-[var(--accent)] focus:ring-2 focus:ring-[var(--accent)]/20"
              value={position}
              onChange={(e) => setPosition(e.target.value)}
            >
              <option value="">選択してください</option>
              {positions.map((p) => (
                <option key={p} value={p}>
                  {p}
                </option>
              ))}
            </select>
          </div>

          {error && <p className="text-sm text-red-600">{error}</p>}

          <button
            type="submit"
            disabled={submitting || !name.trim()}
            className="h-12 w-full rounded-full bg-[var(--accent)] text-base font-semibold text-white transition hover:opacity-90 disabled:opacity-50"
          >
            {submitting ? "保存中..." : "登録して始める"}
          </button>
        </form>
      </div>
    </div>
  );
}
