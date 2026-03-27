"use client";

import {
  createContext,
  useContext,
  useEffect,
  useState,
  type ReactNode,
} from "react";
import type { User } from "firebase/auth";

type AuthState = {
  user: User | null;
  loading: boolean;
  getIdToken: () => Promise<string | null>;
  logout: () => Promise<void>;
};

const AuthContext = createContext<AuthState>({
  user: null,
  loading: true,
  getIdToken: async () => null,
  logout: async () => {},
});

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let unsubscribe: (() => void) | undefined;

    (async () => {
      const { getFirebaseAuth, isFirebaseConfigured } = await import(
        "./firebase"
      );
      if (!isFirebaseConfigured) {
        setLoading(false);
        return;
      }
      const auth = await getFirebaseAuth();
      if (!auth) {
        setLoading(false);
        return;
      }
      const { onAuthStateChanged } = await import("firebase/auth");
      unsubscribe = onAuthStateChanged(auth, (u) => {
        setUser(u);
        setLoading(false);
      });
    })();

    return () => unsubscribe?.();
  }, []);

  const getIdToken = async () => {
    if (!user) return null;
    return user.getIdToken();
  };

  const logout = async () => {
    const { getFirebaseAuth } = await import("./firebase");
    const auth = await getFirebaseAuth();
    if (auth) {
      const { signOut } = await import("firebase/auth");
      await signOut(auth);
    }
  };

  return (
    <AuthContext.Provider value={{ user, loading, getIdToken, logout }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  return useContext(AuthContext);
}
