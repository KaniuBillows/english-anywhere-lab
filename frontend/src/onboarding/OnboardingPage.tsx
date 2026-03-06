import { useState } from 'react';
import { useNavigate } from 'react-router';
import { bootstrapPlan } from '../api/plans';
import { updateProfile } from '../api/profile';
import { useAuth } from '../auth/useAuth';

const LEVELS = ['A1', 'A2', 'B1', 'B2', 'C1', 'C2'] as const;
const DOMAINS = ['General', 'Business', 'Travel', 'Tech', 'Academic', 'Daily Life'];
const MINUTES = [10, 15, 20, 30, 45, 60];
const GOAL_DAYS = [3, 4, 5, 6, 7];
const TOTAL_STEPS = 4;

export default function OnboardingPage() {
  const [step, setStep] = useState(0);
  const [level, setLevel] = useState('');
  const [domain, setDomain] = useState('');
  const [customDomain, setCustomDomain] = useState('');
  const [dailyMinutes, setDailyMinutes] = useState(20);
  const [weeklyGoalDays, setWeeklyGoalDays] = useState(5);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState('');
  const navigate = useNavigate();
  const { refreshUser } = useAuth();

  const selectedDomain = domain === '__custom' ? customDomain : domain;

  async function handleFinish() {
    if (!level || !selectedDomain || !dailyMinutes) return;
    setSubmitting(true);
    setError('');
    try {
      await updateProfile({
        current_level: level,
        target_domain: selectedDomain,
        daily_minutes: dailyMinutes,
        weekly_goal_days: weeklyGoalDays,
      });
      await bootstrapPlan({
        level,
        target_domain: selectedDomain,
        daily_minutes: dailyMinutes,
      });
      await refreshUser();
      navigate('/today', { replace: true });
    } catch (err: unknown) {
      const msg =
        err && typeof err === 'object' && 'message' in err
          ? (err as { message: string }).message
          : 'Failed to create plan';
      setError(msg);
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="min-h-screen flex flex-col items-center justify-center px-4">
      <div className="w-full max-w-md">
        {/* Progress dots */}
        <div className="flex justify-center gap-2 mb-8">
          {Array.from({ length: TOTAL_STEPS }, (_, i) => (
            <div
              key={i}
              className={`h-2 w-2 rounded-full ${i <= step ? 'bg-primary-600' : 'bg-gray-300'}`}
            />
          ))}
        </div>

        {error && (
          <div className="bg-danger-50 text-danger-700 rounded-lg px-4 py-3 text-sm mb-4">
            {error}
          </div>
        )}

        {/* Step 1: Level */}
        {step === 0 && (
          <div>
            <h1 className="text-xl font-bold mb-2">What's your English level?</h1>
            <p className="text-sm text-gray-500 mb-6">Select your current CEFR level</p>
            <div className="grid grid-cols-3 gap-3">
              {LEVELS.map((l) => (
                <button
                  key={l}
                  onClick={() => setLevel(l)}
                  className={`rounded-lg border-2 py-4 text-center font-semibold transition-colors ${
                    level === l
                      ? 'border-primary-600 bg-primary-50 text-primary-700'
                      : 'border-gray-200 hover:border-gray-300'
                  }`}
                >
                  {l}
                </button>
              ))}
            </div>
            <button
              disabled={!level}
              onClick={() => setStep(1)}
              className="mt-6 w-full bg-primary-600 text-white rounded-lg py-2.5 font-medium hover:bg-primary-700 disabled:opacity-50"
            >
              Next
            </button>
          </div>
        )}

        {/* Step 2: Domain */}
        {step === 1 && (
          <div>
            <h1 className="text-xl font-bold mb-2">What do you want to focus on?</h1>
            <p className="text-sm text-gray-500 mb-6">Pick a target domain</p>
            <div className="flex flex-wrap gap-2 mb-4">
              {DOMAINS.map((d) => (
                <button
                  key={d}
                  onClick={() => { setDomain(d); setCustomDomain(''); }}
                  className={`rounded-full px-4 py-2 text-sm font-medium border transition-colors ${
                    domain === d
                      ? 'border-primary-600 bg-primary-50 text-primary-700'
                      : 'border-gray-200 hover:border-gray-300'
                  }`}
                >
                  {d}
                </button>
              ))}
              <button
                onClick={() => setDomain('__custom')}
                className={`rounded-full px-4 py-2 text-sm font-medium border transition-colors ${
                  domain === '__custom'
                    ? 'border-primary-600 bg-primary-50 text-primary-700'
                    : 'border-gray-200 hover:border-gray-300'
                }`}
              >
                Other…
              </button>
            </div>
            {domain === '__custom' && (
              <input
                type="text"
                placeholder="Enter your domain"
                value={customDomain}
                onChange={(e) => setCustomDomain(e.target.value)}
                className="w-full rounded-lg border border-gray-300 px-3 py-2 mb-4 focus:outline-none focus:ring-2 focus:ring-primary-500"
              />
            )}
            <div className="flex gap-3">
              <button
                onClick={() => setStep(0)}
                className="flex-1 border border-gray-300 rounded-lg py-2.5 font-medium hover:bg-gray-50"
              >
                Back
              </button>
              <button
                disabled={!selectedDomain}
                onClick={() => setStep(2)}
                className="flex-1 bg-primary-600 text-white rounded-lg py-2.5 font-medium hover:bg-primary-700 disabled:opacity-50"
              >
                Next
              </button>
            </div>
          </div>
        )}

        {/* Step 3: Daily minutes */}
        {step === 2 && (
          <div>
            <h1 className="text-xl font-bold mb-2">How much time per day?</h1>
            <p className="text-sm text-gray-500 mb-6">Choose your daily learning time</p>
            <div className="grid grid-cols-3 gap-3">
              {MINUTES.map((m) => (
                <button
                  key={m}
                  onClick={() => setDailyMinutes(m)}
                  className={`rounded-lg border-2 py-3 text-center font-medium transition-colors ${
                    dailyMinutes === m
                      ? 'border-primary-600 bg-primary-50 text-primary-700'
                      : 'border-gray-200 hover:border-gray-300'
                  }`}
                >
                  {m} min
                </button>
              ))}
            </div>
            <div className="flex gap-3 mt-6">
              <button
                onClick={() => setStep(1)}
                className="flex-1 border border-gray-300 rounded-lg py-2.5 font-medium hover:bg-gray-50"
              >
                Back
              </button>
              <button
                onClick={() => setStep(3)}
                className="flex-1 bg-primary-600 text-white rounded-lg py-2.5 font-medium hover:bg-primary-700"
              >
                Next
              </button>
            </div>
          </div>
        )}

        {/* Step 4: Weekly goal days */}
        {step === 3 && (
          <div>
            <h1 className="text-xl font-bold mb-2">How many days per week?</h1>
            <p className="text-sm text-gray-500 mb-6">Set your weekly learning goal</p>
            <div className="flex justify-center gap-3">
              {GOAL_DAYS.map((d) => (
                <button
                  key={d}
                  onClick={() => setWeeklyGoalDays(d)}
                  className={`h-12 w-12 rounded-full border-2 text-center font-semibold transition-colors ${
                    weeklyGoalDays === d
                      ? 'border-primary-600 bg-primary-50 text-primary-700'
                      : 'border-gray-200 hover:border-gray-300'
                  }`}
                >
                  {d}
                </button>
              ))}
            </div>
            <p className="text-center text-sm text-gray-400 mt-2">days / week</p>
            <div className="flex gap-3 mt-6">
              <button
                onClick={() => setStep(2)}
                className="flex-1 border border-gray-300 rounded-lg py-2.5 font-medium hover:bg-gray-50"
              >
                Back
              </button>
              <button
                disabled={submitting}
                onClick={handleFinish}
                className="flex-1 bg-primary-600 text-white rounded-lg py-2.5 font-medium hover:bg-primary-700 disabled:opacity-50"
              >
                {submitting ? 'Creating plan…' : 'Start Learning'}
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
