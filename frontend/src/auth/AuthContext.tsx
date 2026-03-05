import {
  createContext,
  useContext,
  useState,
  useEffect,
  useCallback,
  type ReactNode,
} from 'react';
import type { User, MeResponse } from '../api/types';
import { getMe } from '../api/profile';
import { clearTokens, getRefreshToken, setTokens } from '../api/client';

interface AuthContextValue {
  user: User | null;
  learningProfile: MeResponse['learning_profile'] | null;
  loading: boolean;
  login: (user: User, accessToken: string, refreshToken: string) => void;
  logout: () => void;
  refreshUser: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [learningProfile, setLearningProfile] = useState<MeResponse['learning_profile'] | null>(null);
  const [loading, setLoading] = useState(true);

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
    if (getRefreshToken()) {
      refreshUser().finally(() => setLoading(false));
    } else {
      setLoading(false);
    }
  }, [refreshUser]);

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

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used within AuthProvider');
  return ctx;
}
