"use client";

import { FormEvent, useEffect, useState } from "react";
import { updatePassword } from "firebase/auth";
import { useRouter } from "next/navigation";

import { useAuth } from "@/lib/auth-context";
import { apiFetch } from "@/lib/api";

export default function InitialPasswordPage() {
  const { user, loading, getIdToken } = useAuth();
  const router = useRouter();
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    if (!loading && !user) {
      router.replace("/login");
    }
  }, [loading, router, user]);

  if (loading || !user) return null;

  const resolveNextRoute = async (token: string) => {
    const profileRes = await apiFetch(`/api/v1/profile/${user.uid}`, token);
    if (!profileRes.ok) {
      return "/profile-setup";
    }
    const profile = await profileRes.json();
    if (!profile.name) {
      return "/profile-setup";
    }
    return "/";
  };

  const handleSubmit = async (event: FormEvent) => {
    event.preventDefault();
    setError("");

    if (password.length < 8) {
      setError("新しいパスワードは8文字以上にしてください");
      return;
    }
    if (password !== confirmPassword) {
      setError("確認用パスワードが一致しません");
      return;
    }

    setSubmitting(true);
    try {
      await updatePassword(user, password);
      const token = await getIdToken();
      if (!token) {
        setError("ログイン状態を確認できませんでした");
        return;
      }

      const res = await apiFetch(`/api/v1/invite/${user.uid}/password-changed`, token, {
        method: "PATCH",
      });
      if (!res.ok) {
        const data = await res.json().catch(() => null);
        setError(data?.error?.message ?? "初回パスワード変更の保存に失敗しました");
        return;
      }

      router.replace(await resolveNextRoute(token));
    } catch {
      setError("パスワード変更に失敗しました。再度ログインしてやり直してください。");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="flex min-h-dvh items-center justify-center bg-[var(--background)] p-4">
      <div className="w-full max-w-md rounded-[var(--card-radius)] bg-white p-8 shadow-lg dark:bg-card">
        <div className="mb-6 text-center">
          <h1 className="text-2xl font-bold text-[var(--accent)]">Backstage</h1>
          <p className="mt-1 text-sm text-muted-foreground">初回パスワード変更</p>
          <p className="mt-3 text-sm text-muted-foreground">
            初回ログインのため、最初にパスワードを変更してください。
          </p>
        </div>

        <form className="space-y-5" onSubmit={handleSubmit}>
          <div className="space-y-2">
            <label htmlFor="new-password" className="text-sm font-medium text-foreground">
              新しいパスワード
            </label>
            <input
              id="new-password"
              type="password"
              required
              minLength={8}
              className="h-12 w-full rounded-xl border border-border/70 bg-background/85 px-4 text-base outline-none transition focus:border-[var(--accent)] focus:ring-2 focus:ring-[var(--accent)]/20"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
            />
          </div>

          <div className="space-y-2">
            <label htmlFor="confirm-password" className="text-sm font-medium text-foreground">
              新しいパスワード（確認）
            </label>
            <input
              id="confirm-password"
              type="password"
              required
              minLength={8}
              className="h-12 w-full rounded-xl border border-border/70 bg-background/85 px-4 text-base outline-none transition focus:border-[var(--accent)] focus:ring-2 focus:ring-[var(--accent)]/20"
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
            />
          </div>

          {error ? <p className="text-sm text-red-600">{error}</p> : null}

          <button
            type="submit"
            disabled={submitting || password.length < 8 || confirmPassword.length < 8}
            className="h-12 w-full rounded-full bg-[var(--accent)] text-base font-semibold text-white transition hover:opacity-90 disabled:opacity-50"
          >
            {submitting ? "変更中..." : "パスワードを変更して続行"}
          </button>
        </form>
      </div>
    </div>
  );
}
