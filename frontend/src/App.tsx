import { BrowserRouter, Routes, Route, Navigate } from 'react-router';
import { AuthProvider } from './auth/AuthContext';
import ProtectedRoute from './components/ProtectedRoute';
import AppShell from './components/AppShell';
import LoginPage from './auth/LoginPage';
import RegisterPage from './auth/RegisterPage';
import OnboardingPage from './onboarding/OnboardingPage';
import TodayPage from './today/TodayPage';
import ReviewPage from './review/ReviewPage';
import ProgressPage from './progress/ProgressPage';
import WeeklyReport from './progress/WeeklyReport';
import MonthlyReport from './progress/MonthlyReport';
import ProfilePage from './profile/ProfilePage';

export default function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <Routes>
          {/* Public */}
          <Route path="/login" element={<LoginPage />} />
          <Route path="/register" element={<RegisterPage />} />

          {/* Protected without nav */}
          <Route
            path="/onboarding"
            element={
              <ProtectedRoute>
                <OnboardingPage />
              </ProtectedRoute>
            }
          />

          {/* Protected with nav */}
          <Route
            element={
              <ProtectedRoute>
                <AppShell />
              </ProtectedRoute>
            }
          >
            <Route path="/today" element={<TodayPage />} />
            <Route path="/review" element={<ReviewPage />} />
            <Route path="/progress" element={<ProgressPage />} />
            <Route path="/progress/weekly" element={<WeeklyReport />} />
            <Route path="/progress/monthly" element={<MonthlyReport />} />
            <Route path="/profile" element={<ProfilePage />} />
          </Route>

          {/* Default redirect */}
          <Route path="*" element={<Navigate to="/today" replace />} />
        </Routes>
      </AuthProvider>
    </BrowserRouter>
  );
}
