"use client";

import { FormEvent, useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "@/lib/auth-context";
import { apiFetch } from "@/lib/api";
import "./members.css";

type Member = {
  id: string;
  email: string;
  name: string;
  role: string;
};

export default function MembersPage() {
  const { user, loading, getIdToken, logout } = useAuth();
  const router = useRouter();
  const [members, setMembers] = useState<Member[]>([]);
  const [loadingMembers, setLoadingMembers] = useState(true);

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [submitting, setSubmitting] = useState(false);

  const fetchMembers = useCallback(async () => {
    const token = await getIdToken();
    const res = await apiFetch("/api/v1/members", token);
    if (res.ok) {
      const data = await res.json();
      setMembers(data.members ?? []);
    }
    setLoadingMembers(false);
  }, [getIdToken]);

  useEffect(() => {
    if (!loading && user) {
      fetchMembers();
    }
  }, [loading, user, fetchMembers]);

  if (loading) return null;
  if (!user) {
    router.replace("/login");
    return null;
  }

  const handleCreate = async (e: FormEvent) => {
    e.preventDefault();
    setError("");
    setSuccess("");
    setSubmitting(true);

    try {
      const token = await getIdToken();
      const res = await apiFetch("/api/v1/members", token, {
        method: "POST",
        body: JSON.stringify({
          email,
          password,
          display_name: displayName || undefined,
        }),
      });

      if (!res.ok) {
        const body = await res.json().catch(() => null);
        setError(body?.message ?? `エラーが発生しました (${res.status})`);
        return;
      }

      setSuccess("メンバーを作成しました");
      setEmail("");
      setPassword("");
      setDisplayName("");
      await fetchMembers();
    } catch {
      setError("通信エラーが発生しました");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="members-page">
      <header className="members-header">
        <div className="members-header-left">
          <h1 className="members-title">メンバー管理</h1>
          <button
            type="button"
            className="members-back-btn"
            onClick={() => router.push("/")}
          >
            タスクに戻る
          </button>
        </div>
        <div className="members-header-right">
          <span>{user.email}</span>
          <button type="button" className="members-logout-btn" onClick={logout}>
            ログアウト
          </button>
        </div>
      </header>

      <main className="members-main">
        <section className="members-create-section">
          <h2 className="members-section-title">新しいメンバーを追加</h2>
          <form className="members-form" onSubmit={handleCreate}>
            <div className="members-form-row">
              <label className="members-label" htmlFor="member-email">
                メールアドレス *
              </label>
              <input
                id="member-email"
                className="members-input"
                type="email"
                required
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="user@example.com"
              />
            </div>
            <div className="members-form-row">
              <label className="members-label" htmlFor="member-password">
                パスワード *
              </label>
              <input
                id="member-password"
                className="members-input"
                type="password"
                required
                minLength={6}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="6文字以上"
              />
            </div>
            <div className="members-form-row">
              <label className="members-label" htmlFor="member-name">
                表示名
              </label>
              <input
                id="member-name"
                className="members-input"
                type="text"
                value={displayName}
                onChange={(e) => setDisplayName(e.target.value)}
                placeholder="省略時はメールアドレスが使用されます"
              />
            </div>
            {error && <p className="members-error">{error}</p>}
            {success && <p className="members-success">{success}</p>}
            <button
              className="members-submit-btn"
              type="submit"
              disabled={submitting}
            >
              {submitting ? "作成中..." : "メンバーを作成"}
            </button>
          </form>
        </section>

        <section className="members-list-section">
          <h2 className="members-section-title">メンバー一覧</h2>
          {loadingMembers ? (
            <p className="members-loading">読み込み中...</p>
          ) : members.length === 0 ? (
            <p className="members-empty">メンバーがいません</p>
          ) : (
            <table className="members-table">
              <thead>
                <tr>
                  <th>名前</th>
                  <th>メール</th>
                  <th>ロール</th>
                </tr>
              </thead>
              <tbody>
                {members.map((m) => (
                  <tr key={m.id}>
                    <td>{m.name}</td>
                    <td>{m.email}</td>
                    <td>
                      <span className={`role-badge role-${m.role.toLowerCase()}`}>
                        {m.role}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </section>
      </main>
    </div>
  );
}
