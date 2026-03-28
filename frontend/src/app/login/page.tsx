"use client";

import { FormEvent, useEffect, useState } from "react";
import { signInWithEmailAndPassword } from "firebase/auth";
import { useRouter } from "next/navigation";
import { getFirebaseAuth, getMissingFirebaseConfigKeys } from "@/lib/firebase";
import { useAuth } from "@/lib/auth-context";
import { apiFetch } from "@/lib/api";
import "./login.css";

export default function LoginPage() {
  const router = useRouter();
  const { user, loading } = useAuth();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    if (!loading && user) {
      router.replace("/");
    }
  }, [loading, router, user]);

  if (loading || user) return null;

  const resolvePostLoginRoute = async (uid: string, token: string) => {
    const inviteStatusRes = await apiFetch(`/api/v1/invite/${uid}/status`, token);
    if (inviteStatusRes.ok) {
      const inviteStatus = await inviteStatusRes.json();
      if (inviteStatus?.needsPasswordReset) {
        return "/initial-password";
      }
    }

    const profileRes = await apiFetch(`/api/v1/profile/${uid}`, token);
    if (!profileRes.ok) {
      return "/profile-setup";
    }

    const profile = await profileRes.json();
    if (!profile.name) {
      return "/profile-setup";
    }
    return "/";
  };

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError("");
    setSubmitting(true);
    try {
      const auth = getFirebaseAuth();
      if (!auth) {
        setError(
          `Firebase が設定されていません: ${getMissingFirebaseConfigKeys().join(", ")}`
        );
        return;
      }
      const credential = await signInWithEmailAndPassword(auth, email, password);
      const token = await credential.user.getIdToken();

      try {
        const uid = credential.user.uid;
        router.replace(await resolvePostLoginRoute(uid, token));
        return;
      } catch {
        // API unreachable, go to home
      }

      router.replace("/");
    } catch {
      setError("メールアドレスまたはパスワードが正しくありません");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="login-page">
      <div className="login-card">
        <h1 className="login-title">Backstage</h1>
        <p className="login-subtitle">ログイン</p>
        <form className="login-form" onSubmit={handleSubmit}>
          <label className="login-label" htmlFor="email">
            メールアドレス
          </label>
          <input
            id="email"
            className="login-input"
            type="email"
            autoComplete="email"
            required
            value={email}
            onChange={(e) => setEmail(e.target.value)}
          />
          <label className="login-label" htmlFor="password">
            パスワード
          </label>
          <input
            id="password"
            className="login-input"
            type="password"
            autoComplete="current-password"
            required
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
          {error && <p className="login-error">{error}</p>}
          <button className="login-btn" type="submit" disabled={submitting}>
            {submitting ? "ログイン中..." : "ログイン"}
          </button>
        </form>
      </div>
    </div>
  );
}
