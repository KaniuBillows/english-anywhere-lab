import { useState, useEffect } from 'react';
import { useAuth } from '../auth/useAuth';
import { updateProfile } from '../api/profile';
import type { UpdateProfileRequest } from '../api/types';
import { useNavigate } from 'react-router';

const LEVELS = ['A1', 'A2', 'B1', 'B2', 'C1', 'C2'];
const DOMAINS = ['General', 'Business', 'Travel', 'Tech', 'Academic', 'Daily Life'];
const MINUTES = [10, 15, 20, 30, 45, 60];

export default function ProfilePage() {
  const { user, learningProfile, logout, refreshUser } = useAuth();
  const navigate = useNavigate();
  const [editing, setEditing] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  // Edit form state
  const [level, setLevel] = useState('');
  const [domain, setDomain] = useState('');
  const [dailyMinutes, setDailyMinutes] = useState(20);
  const [weeklyGoalDays, setWeeklyGoalDays] = useState(5);

  useEffect(() => {
    if (learningProfile) {
      setLevel(learningProfile.current_level);
      setDomain(learningProfile.target_domain);
      setDailyMinutes(learningProfile.daily_minutes);
      setWeeklyGoalDays(learningProfile.weekly_goal_days);
    }
  }, [learningProfile]);

  async function handleSave() {
    setSaving(true);
    setError('');
    try {
      const req: UpdateProfileRequest = {
        current_level: level,
        target_domain: domain,
        daily_minutes: dailyMinutes,
        weekly_goal_days: weeklyGoalDays,
      };
      await updateProfile(req);
      await refreshUser();
      setEditing(false);
    } catch (err: unknown) {
      const msg =
        err && typeof err === 'object' && 'message' in err
          ? (err as { message: string }).message
          : 'Failed to save';
      setError(msg);
    } finally {
      setSaving(false);
    }
  }

  function handleLogout() {
    logout();
    navigate('/login', { replace: true });
  }

  if (!user) return null;

  return (
    <div>
      <h1 className="text-xl font-bold mb-4">Profile</h1>

      {error && (
        <div className="bg-danger-50 text-danger-700 rounded-lg px-4 py-3 text-sm mb-4">
          {error}
        </div>
      )}

      {/* User info */}
      <div className="rounded-lg border border-gray-200 bg-white p-4 mb-4">
        <p className="text-sm text-gray-500">Email</p>
        <p className="font-medium">{user.email}</p>
        <div className="flex gap-4 mt-2">
          <div>
            <p className="text-sm text-gray-500">Locale</p>
            <p className="text-sm">{user.locale || '—'}</p>
          </div>
          <div>
            <p className="text-sm text-gray-500">Timezone</p>
            <p className="text-sm">{user.timezone || '—'}</p>
          </div>
        </div>
      </div>

      {/* Learning profile */}
      <div className="rounded-lg border border-gray-200 bg-white p-4 mb-4">
        <div className="flex justify-between items-center mb-3">
          <p className="font-medium">Learning Profile</p>
          {!editing && (
            <button
              onClick={() => setEditing(true)}
              className="text-sm text-primary-600 hover:underline"
            >
              Edit
            </button>
          )}
        </div>

        {editing ? (
          <div className="space-y-4">
            {/* Level */}
            <div>
              <p className="text-sm text-gray-500 mb-1">Level</p>
              <div className="flex flex-wrap gap-2">
                {LEVELS.map((l) => (
                  <button
                    key={l}
                    onClick={() => setLevel(l)}
                    className={`rounded-full px-3 py-1 text-sm font-medium border ${
                      level === l
                        ? 'border-primary-600 bg-primary-50 text-primary-700'
                        : 'border-gray-200'
                    }`}
                  >
                    {l}
                  </button>
                ))}
              </div>
            </div>

            {/* Domain */}
            <div>
              <p className="text-sm text-gray-500 mb-1">Domain</p>
              <div className="flex flex-wrap gap-2">
                {DOMAINS.map((d) => (
                  <button
                    key={d}
                    onClick={() => setDomain(d)}
                    className={`rounded-full px-3 py-1 text-sm font-medium border ${
                      domain === d
                        ? 'border-primary-600 bg-primary-50 text-primary-700'
                        : 'border-gray-200'
                    }`}
                  >
                    {d}
                  </button>
                ))}
              </div>
            </div>

            {/* Daily minutes */}
            <div>
              <p className="text-sm text-gray-500 mb-1">Daily Minutes</p>
              <div className="flex flex-wrap gap-2">
                {MINUTES.map((m) => (
                  <button
                    key={m}
                    onClick={() => setDailyMinutes(m)}
                    className={`rounded-full px-3 py-1 text-sm font-medium border ${
                      dailyMinutes === m
                        ? 'border-primary-600 bg-primary-50 text-primary-700'
                        : 'border-gray-200'
                    }`}
                  >
                    {m}
                  </button>
                ))}
              </div>
            </div>

            {/* Weekly goal */}
            <div>
              <p className="text-sm text-gray-500 mb-1">Weekly Goal (days)</p>
              <div className="flex gap-2">
                {[3, 4, 5, 6, 7].map((d) => (
                  <button
                    key={d}
                    onClick={() => setWeeklyGoalDays(d)}
                    className={`rounded-full px-3 py-1 text-sm font-medium border ${
                      weeklyGoalDays === d
                        ? 'border-primary-600 bg-primary-50 text-primary-700'
                        : 'border-gray-200'
                    }`}
                  >
                    {d}
                  </button>
                ))}
              </div>
            </div>

            <div className="flex gap-3 pt-2">
              <button
                onClick={() => setEditing(false)}
                className="flex-1 border border-gray-300 rounded-lg py-2 font-medium hover:bg-gray-50"
              >
                Cancel
              </button>
              <button
                onClick={handleSave}
                disabled={saving}
                className="flex-1 bg-primary-600 text-white rounded-lg py-2 font-medium hover:bg-primary-700 disabled:opacity-50"
              >
                {saving ? 'Saving…' : 'Save'}
              </button>
            </div>
          </div>
        ) : learningProfile ? (
          <div className="grid grid-cols-2 gap-3 text-sm">
            <div>
              <p className="text-gray-500">Level</p>
              <p className="font-medium">{learningProfile.current_level || '—'}</p>
            </div>
            <div>
              <p className="text-gray-500">Domain</p>
              <p className="font-medium">{learningProfile.target_domain || '—'}</p>
            </div>
            <div>
              <p className="text-gray-500">Daily Minutes</p>
              <p className="font-medium">{learningProfile.daily_minutes || '—'}</p>
            </div>
            <div>
              <p className="text-gray-500">Weekly Goal</p>
              <p className="font-medium">{learningProfile.weekly_goal_days || '—'} days</p>
            </div>
          </div>
        ) : (
          <p className="text-sm text-gray-500">No learning profile set up yet.</p>
        )}
      </div>

      <button
        onClick={handleLogout}
        className="w-full border border-danger-500 text-danger-600 rounded-lg py-2.5 font-medium hover:bg-danger-50"
      >
        Log Out
      </button>
    </div>
  );
}
