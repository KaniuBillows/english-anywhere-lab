import { useState, type FormEvent } from 'react';
import { Link, useNavigate } from 'react-router';
import { login as apiLogin } from '../api/auth';
import { getTodayPlan } from '../api/plans';
import { getUserTimezone } from '../lib/date';
import { useAuth } from './useAuth';

export default function LoginPage() {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const { login, refreshUser } = useAuth();
  const navigate = useNavigate();

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError('');
    setSubmitting(true);

    try {
      const res = await apiLogin({ email, password });
      login(res.user, res.tokens.access_token, res.tokens.refresh_token);
      await refreshUser();

      // Check if onboarding is needed by looking for an existing plan
      let needsOnboarding = true;
      try {
        const plan = await getTodayPlan(getUserTimezone());
        needsOnboarding = !plan.daily_plan.plan_id;
      } catch {
        // No plan available → needs onboarding
      }

      if (needsOnboarding) {
        navigate('/onboarding', { replace: true });
      } else {
        navigate('/today', { replace: true });
      }
    } catch (err: unknown) {
      const msg =
        err && typeof err === 'object' && 'message' in err
          ? (err as { message: string }).message
          : 'Login failed';
      setError(msg);
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center px-4">
      <div className="w-full max-w-sm">
        <h1 className="text-2xl font-bold text-center mb-6">Sign In</h1>

        <form onSubmit={handleSubmit} className="space-y-4">
          {error && (
            <div className="bg-danger-50 text-danger-700 rounded-lg px-4 py-3 text-sm">
              {error}
            </div>
          )}

          <div>
            <label htmlFor="email" className="block text-sm font-medium text-gray-700 mb-1">
              Email
            </label>
            <input
              id="email"
              type="email"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="w-full rounded-lg border border-gray-300 px-3 py-2 focus:outline-none focus:ring-2 focus:ring-primary-500"
            />
          </div>

          <div>
            <label htmlFor="password" className="block text-sm font-medium text-gray-700 mb-1">
              Password
            </label>
            <input
              id="password"
              type="password"
              required
              minLength={8}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="w-full rounded-lg border border-gray-300 px-3 py-2 focus:outline-none focus:ring-2 focus:ring-primary-500"
            />
          </div>

          <button
            type="submit"
            disabled={submitting}
            className="w-full bg-primary-600 text-white rounded-lg py-2.5 font-medium hover:bg-primary-700 disabled:opacity-50"
          >
            {submitting ? 'Signing in…' : 'Sign In'}
          </button>
        </form>

        <p className="mt-4 text-center text-sm text-gray-600">
          Don't have an account?{' '}
          <Link to="/register" className="text-primary-600 hover:underline">
            Register
          </Link>
        </p>
      </div>
    </div>
  );
}
