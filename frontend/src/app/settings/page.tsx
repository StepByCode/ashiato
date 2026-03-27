"use client";

import { FormEvent, useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "@/lib/auth-context";
import { apiFetch } from "@/lib/api";

const positions = ["代表", "副代表", "エンジニア", "デザイナー", "広報", "その他"] as const;

export default function SettingsPage() {
  const { user, loading, logout, getIdToken } = useAuth();
  const router = useRouter();
  const [name, setName] = useState("");
  const [position, setPosition] = useState("");
  const [loadingProfile, setLoadingProfile] = useState(true);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [submitting, setSubmitting] = useState(false);

  const uid = user?.uid ?? "anonymous";

  const fetchProfile = useCallback(async () => {
    try {
      const token = await getIdToken();
      if (!token) return;
      const res = await apiFetch(`/api/v1/profile/${uid}`, token);
      if (res.ok) {
        const data = await res.json();
        setName(data.name ?? "");
        setPosition(data.position ?? "");
      }
    } catch {
      // API unreachable
    } finally {
      setLoadingProfile(false);
    }
  }, [uid]);

  useEffect(() => {
    if (!loading && !user) {
      router.replace("/login");
      return;
    }
    if (user) {
      void fetchProfile();
    }
  }, [fetchProfile, loading, router, user]);

  if (loading || !user) return null;

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    if (!name.trim()) return;
    setError("");
    setSuccess("");
    setSubmitting(true);

    try {
      const token = await getIdToken();
      if (!token) {
        setError("ログイン状態を確認できませんでした");
        return;
      }
      const res = await apiFetch(`/api/v1/profile/${uid}`, token, {
        method: "PATCH",
        body: JSON.stringify({ name: name.trim(), position }),
      });
      if (!res.ok) {
        const data = await res.json().catch(() => null);
        setError(data?.error?.message ?? "保存に失敗しました");
        return;
      }
      setSuccess("プロフィールを更新しました");
    } catch {
      setError("通信エラーが発生しました");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="flex min-h-dvh items-center justify-center bg-[var(--background)] p-4">
      <div className="w-full max-w-md rounded-[var(--card-radius)] bg-white p-8 shadow-lg dark:bg-card">
        <div className="mb-6 flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold text-[var(--accent)]">Backstage</h1>
            <p className="mt-1 text-sm text-muted-foreground">プロフィール設定</p>
          </div>
          <button
            type="button"
            onClick={() => router.push("/")}
            className="rounded-full border border-border/70 px-4 py-2 text-sm font-medium transition hover:bg-background/80"
          >
            戻る
          </button>
        </div>

        {loadingProfile ? (
          <p className="py-8 text-center text-muted-foreground">読み込み中...</p>
        ) : (
          <form className="space-y-5" onSubmit={handleSubmit}>
            <div className="space-y-2">
              <label className="text-sm font-medium text-muted-foreground">メールアドレス</label>
              <p className="rounded-xl border border-border/40 bg-muted/30 px-4 py-3 text-sm">
                {user?.email ?? "未設定"}
              </p>
            </div>

            <div className="space-y-2">
              <label htmlFor="settings-name" className="text-sm font-medium text-foreground">
                名前 <span className="text-[var(--accent)]">*</span>
              </label>
              <input
                id="settings-name"
                type="text"
                required
                className="h-12 w-full rounded-xl border border-border/70 bg-background/85 px-4 text-base outline-none transition focus:border-[var(--accent)] focus:ring-2 focus:ring-[var(--accent)]/20"
                value={name}
                onChange={(e) => setName(e.target.value)}
              />
            </div>

            <div className="space-y-2">
              <label htmlFor="settings-position" className="text-sm font-medium text-foreground">
                ポジション
              </label>
              <select
                id="settings-position"
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
            {success && <p className="text-sm text-emerald-600">{success}</p>}

            <button
              type="submit"
              disabled={submitting || !name.trim()}
              className="h-12 w-full rounded-full bg-[var(--accent)] text-base font-semibold text-white transition hover:opacity-90 disabled:opacity-50"
            >
              {submitting ? "保存中..." : "更新する"}
            </button>

            <div className="flex items-center justify-between border-t border-border/40 pt-4">
              <button
                type="button"
                onClick={() => router.push("/invite")}
                className="text-sm font-medium text-[var(--accent)] transition hover:opacity-80"
              >
                メンバーを招待
              </button>
              {user && (
                <button
                  type="button"
                  onClick={logout}
                  className="text-sm font-medium text-muted-foreground transition hover:text-foreground"
                >
                  ログアウト
                </button>
              )}
            </div>
          </form>
        )}
      </div>
    </div>
  );
}
