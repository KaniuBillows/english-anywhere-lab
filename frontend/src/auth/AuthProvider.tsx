import {
  useState,
  useEffect,
  useCallback,
  useRef,
  type ReactNode,
} from 'react';
import type { User, LearningProfile } from '../api/types';
import { getMe } from '../api/profile';
import { clearTokens, getRefreshToken, setTokens } from '../api/client';
import { AuthContext } from './authContext';

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [learningProfile, setLearningProfile] = useState<LearningProfile | null>(null);
  const [loading, setLoading] = useState(() => !!getRefreshToken());
  const initialized = useRef(false);

  const refreshUser = useCallback(async () => {
    try {
      const me = await getMe();
      setUser(me.user);
      setLearningProfile(me.learning_profile);
    } catch {
      setUser(null);
      setLearningProfile(null);
    }
  }, []);

  useEffect(() => {
    if (initialized.current) return;
    initialized.current = true;

    if (!getRefreshToken()) return;

    let cancelled = false;
    getMe()
      .then((me) => {
        if (cancelled) return;
        setUser(me.user);
        setLearningProfile(me.learning_profile);
      })
      .catch(() => {
        // token invalid
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });

    return () => { cancelled = true; };
  }, []);

  const login = useCallback(
    (u: User, accessToken: string, refreshToken: string) => {
      setTokens(accessToken, refreshToken);
      setUser(u);
    },
    [],
  );

  const logout = useCallback(() => {
    clearTokens();
    setUser(null);
    setLearningProfile(null);
  }, []);

  return (
    <AuthContext.Provider value={{ user, learningProfile, loading, login, logout, refreshUser }}>
      {children}
    </AuthContext.Provider>
  );
}
