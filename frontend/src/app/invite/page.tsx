"use client";

import { FormEvent, useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "@/lib/auth-context";
import { isFirebaseConfigured } from "@/lib/firebase";
import { apiFetch } from "@/lib/api";

type Invite = {
  uid: string;
  email: string;
  createdAt: string;
};

export default function InvitePage() {
  const { user, loading } = useAuth();
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [error, setError] = useState("");
  const [result, setResult] = useState<{ email: string; password: string } | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [invites, setInvites] = useState<Invite[]>([]);
  const [loadingInvites, setLoadingInvites] = useState(true);

  const fetchInvites = useCallback(async () => {
    try {
      const res = await apiFetch("/api/v1/invites", null);
      if (res.ok) {
        const data = await res.json();
        setInvites(data.invites ?? []);
      }
    } catch {
      // API unreachable
    } finally {
      setLoadingInvites(false);
    }
  }, []);

  useEffect(() => {
    fetchInvites();
  }, [fetchInvites]);

  if (loading) return null;
  if (!user && isFirebaseConfigured) {
    router.replace("/login");
    return null;
  }

  const handleInvite = async (e: FormEvent) => {
    e.preventDefault();
    if (!email.trim()) return;
    setError("");
    setResult(null);
    setSubmitting(true);

    try {
      const res = await apiFetch("/api/v1/invite", null, {
        method: "POST",
        body: JSON.stringify({ email: email.trim() }),
      });
      const data = await res.json();
      if (!res.ok) {
        setError(data?.error?.message ?? "招待に失敗しました");
        return;
      }
      setResult({ email: data.email, password: data.password });
      setEmail("");
      await fetchInvites();
    } catch {
      setError("通信エラーが発生しました");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="flex min-h-dvh items-center justify-center bg-[var(--background)] p-4">
      <div className="w-full max-w-lg rounded-[var(--card-radius)] bg-white p-8 shadow-lg dark:bg-card">
        <div className="mb-6 flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold text-[var(--accent)]">Ashiato</h1>
            <p className="mt-1 text-sm text-muted-foreground">メンバー招待</p>
          </div>
          <button
            type="button"
            onClick={() => router.push("/settings")}
            className="rounded-full border border-border/70 px-4 py-2 text-sm font-medium transition hover:bg-background/80"
          >
            戻る
          </button>
        </div>

        <form className="space-y-4" onSubmit={handleInvite}>
          <div className="space-y-2">
            <label htmlFor="invite-email" className="text-sm font-medium text-foreground">
              招待するメールアドレス <span className="text-[var(--accent)]">*</span>
            </label>
            <input
              id="invite-email"
              type="email"
              required
              className="h-12 w-full rounded-xl border border-border/70 bg-background/85 px-4 text-base outline-none transition focus:border-[var(--accent)] focus:ring-2 focus:ring-[var(--accent)]/20"
              placeholder="user@example.com"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
            />
            <p className="text-xs text-muted-foreground">
              パスワードは自動生成され、招待メールで送信されます。
            </p>
          </div>

          {error && <p className="text-sm text-red-600">{error}</p>}

          <button
            type="submit"
            disabled={submitting || !email.trim()}
            className="h-12 w-full rounded-full bg-[var(--accent)] text-base font-semibold text-white transition hover:opacity-90 disabled:opacity-50"
          >
            {submitting ? "招待中..." : "招待を送信"}
          </button>
        </form>

        {result && (
          <div className="mt-6 rounded-xl border border-emerald-300/60 bg-emerald-50 p-4 dark:bg-emerald-950/20">
            <h3 className="mb-2 text-sm font-semibold text-emerald-800 dark:text-emerald-300">
              招待が完了しました
            </h3>
            <div className="space-y-1 text-sm">
              <p>
                <span className="font-medium">メール:</span> {result.email}
              </p>
              <p>
                <span className="font-medium">初回パスワード:</span>{" "}
                <code className="rounded bg-background px-2 py-0.5 font-mono text-xs">
                  {result.password}
                </code>
              </p>
            </div>
            <p className="mt-2 text-xs text-muted-foreground">
              招待メールが送信されました。この情報は画面を離れると再表示できません。
            </p>
          </div>
        )}

        <div className="mt-8">
          <h2 className="mb-3 text-lg font-semibold">招待済みメンバー</h2>
          {loadingInvites ? (
            <p className="text-sm text-muted-foreground">読み込み中...</p>
          ) : invites.length === 0 ? (
            <p className="text-sm text-muted-foreground">まだ招待されたメンバーはいません。</p>
          ) : (
            <div className="space-y-2">
              {invites.map((inv) => (
                <div
                  key={inv.uid}
                  className="flex items-center justify-between rounded-xl border border-border/40 px-4 py-3"
                >
                  <span className="text-sm">{inv.email}</span>
                  <span className="text-xs text-muted-foreground">
                    {inv.createdAt ? new Date(inv.createdAt).toLocaleDateString("ja-JP") : ""}
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
