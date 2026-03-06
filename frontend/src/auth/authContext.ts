import { createContext } from 'react';
import type { User, LearningProfile } from '../api/types';

export interface AuthContextValue {
  user: User | null;
  learningProfile: LearningProfile | null;
  loading: boolean;
  login: (user: User, accessToken: string, refreshToken: string) => void;
  logout: () => void;
  refreshUser: () => Promise<void>;
}

export const AuthContext = createContext<AuthContextValue | null>(null);
