"use client";

import { FormEvent, useState } from "react";
import { signInWithEmailAndPassword } from "firebase/auth";
import { useRouter } from "next/navigation";
import { getFirebaseAuth } from "@/lib/firebase";
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

  if (loading) return null;
  if (user) {
    router.replace("/");
    return null;
  }

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError("");
    setSubmitting(true);
    try {
      const auth = getFirebaseAuth();
      if (!auth) {
        setError("Firebase が設定されていません");
        return;
      }
      const credential = await signInWithEmailAndPassword(auth, email, password);

      // Check if profile exists, redirect to setup if not
      try {
        const uid = credential.user.uid;
        const res = await apiFetch(`/api/v1/profile/${uid}`, null);
        if (res.ok) {
          const profile = await res.json();
          if (!profile.name) {
            router.replace("/profile-setup");
            return;
          }
        } else {
          router.replace("/profile-setup");
          return;
        }
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
        <h1 className="login-title">Ashiato</h1>
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
