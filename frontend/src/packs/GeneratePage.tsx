import { useEffect, useState } from 'react';
import { Link, useNavigate } from 'react-router';
import { generatePack, getGenerationJob } from '../api/packs';
import type { ApiError } from '../api/types';

const LEVELS = ['A1', 'A2', 'B1', 'B2', 'C1', 'C2'] as const;
const DOMAINS = ['General', 'Business', 'Travel', 'Tech', 'Academic', 'Daily Life'];
const TEMPLATES: { value: string; label: string }[] = [
  { value: 'vocab_foundation', label: 'Vocab Foundation' },
  { value: 'scenario_dialog', label: 'Scenario Dialog' },
  { value: 'intensive_listening', label: 'Intensive Listening' },
  { value: 'reading_comprehension', label: 'Reading Comprehension' },
  { value: 'writing_output', label: 'Writing Output' },
  { value: 'review_booster', label: 'Review Booster' },
  { value: 'speaking_bootcamp', label: 'Speaking Bootcamp' },
  { value: 'exam_drill', label: 'Exam Drill' },
];
const MINUTES = [10, 15, 20, 30, 45, 60];
const DAYS = [3, 5, 7, 10, 14];
const SKILLS = ['listening', 'speaking', 'reading', 'writing'];
const TOTAL_STEPS = 5;

type JobStatus = 'idle' | 'submitting' | 'polling' | 'success' | 'failed' | 'rate_limited';

export default function GeneratePage() {
  const navigate = useNavigate();
  const [step, setStep] = useState(0);
  const [level, setLevel] = useState('');
  const [domain, setDomain] = useState('');
  const [customDomain, setCustomDomain] = useState('');
  const [packTemplate, setPackTemplate] = useState('');
  const [dailyMinutes, setDailyMinutes] = useState(20);
  const [days, setDays] = useState(7);
  const [focusSkills, setFocusSkills] = useState<string[]>([]);

  const [jobStatus, setJobStatus] = useState<JobStatus>('idle');
  const [jobId, setJobId] = useState('');
  const [packId, setPackId] = useState('');
  const [errorMsg, setErrorMsg] = useState('');

  const selectedDomain = domain === '__custom' ? customDomain : domain;

  function toggleSkill(skill: string) {
    setFocusSkills((prev) =>
      prev.includes(skill) ? prev.filter((s) => s !== skill) : [...prev, skill],
    );
  }

  async function handleSubmit() {
    if (!level || !selectedDomain || !packTemplate) return;
    setJobStatus('submitting');
    setErrorMsg('');
    try {
      const res = await generatePack({
        level,
        domain: selectedDomain,
        daily_minutes: dailyMinutes,
        pack_template: packTemplate,
        days,
        focus_skills: focusSkills.length > 0 ? focusSkills : undefined,
      });
      setJobId(res.job_id);
      setJobStatus('polling');
    } catch (err: unknown) {
      const apiErr = err as ApiError;
      if (apiErr?.code === 'RATE_LIMIT' || apiErr?.message?.includes('429')) {
        setJobStatus('rate_limited');
        setErrorMsg('Too many requests. Please wait a moment and try again.');
      } else {
        setJobStatus('failed');
        setErrorMsg(apiErr?.message || 'Failed to start generation');
      }
    }
  }

  // Poll for job status
  useEffect(() => {
    if (jobStatus !== 'polling' || !jobId) return;
    const interval = setInterval(async () => {
      try {
        const job = await getGenerationJob(jobId);
        if (job.status === 'completed' || job.status === 'success') {
          setPackId(job.pack_id || '');
          setJobStatus('success');
        } else if (job.status === 'failed') {
          setErrorMsg(job.error_message || 'Generation failed');
          setJobStatus('failed');
        }
        // else still queued/generating — keep polling
      } catch {
        setErrorMsg('Failed to check generation status');
        setJobStatus('failed');
      }
    }, 3000);
    return () => clearInterval(interval);
  }, [jobStatus, jobId]);

  function handleRetry() {
    setJobStatus('idle');
    setJobId('');
    setPackId('');
    setErrorMsg('');
  }

  // Generation status screen
  if (jobStatus !== 'idle') {
    return (
      <div className="min-h-screen flex flex-col items-center justify-center px-4">
        <div className="w-full max-w-md text-center">
          {(jobStatus === 'submitting' || jobStatus === 'polling') && (
            <>
              <div className="animate-spin h-10 w-10 border-4 border-primary-600 border-t-transparent rounded-full mx-auto mb-4" />
              <h2 className="text-lg font-semibold mb-2">
                {jobStatus === 'submitting' ? 'Starting generation…' : 'Generating your pack…'}
              </h2>
              <p className="text-sm text-gray-500">
                {jobStatus === 'polling'
                  ? 'This may take a moment. AI is creating your personalized pack.'
                  : 'Sending request…'}
              </p>
            </>
          )}

          {jobStatus === 'success' && (
            <>
              <div className="text-4xl mb-4">✅</div>
              <h2 className="text-lg font-semibold mb-2">Pack Generated!</h2>
              <p className="text-sm text-gray-500 mb-6">
                Your AI-generated pack is ready.
              </p>
              {packId ? (
                <Link
                  to={`/packs/${packId}`}
                  className="inline-block bg-primary-600 text-white rounded-lg px-6 py-2.5 font-medium hover:bg-primary-700"
                >
                  View Pack
                </Link>
              ) : (
                <Link
                  to="/packs"
                  className="inline-block bg-primary-600 text-white rounded-lg px-6 py-2.5 font-medium hover:bg-primary-700"
                >
                  Browse Packs
                </Link>
              )}
            </>
          )}

          {(jobStatus === 'failed' || jobStatus === 'rate_limited') && (
            <>
              <div className="text-4xl mb-4">❌</div>
              <h2 className="text-lg font-semibold mb-2">
                {jobStatus === 'rate_limited' ? 'Rate Limited' : 'Generation Failed'}
              </h2>
              <p className="text-sm text-gray-500 mb-6">{errorMsg}</p>
              <button
                onClick={handleRetry}
                className="inline-block bg-primary-600 text-white rounded-lg px-6 py-2.5 font-medium hover:bg-primary-700"
              >
                Try Again
              </button>
            </>
          )}
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen flex flex-col items-center justify-center px-4">
      <div className="w-full max-w-md">
        {/* Back nav */}
        <button
          onClick={() => navigate(-1)}
          className="text-sm text-gray-500 hover:text-gray-700 mb-4 inline-flex items-center gap-1"
        >
          &larr; Back
        </button>

        <h1 className="text-xl font-bold mb-2">Generate AI Pack</h1>
        <p className="text-sm text-gray-500 mb-6">
          Create a personalized learning pack with AI.
        </p>

        {/* Progress dots */}
        <div className="flex justify-center gap-2 mb-8">
          {Array.from({ length: TOTAL_STEPS }, (_, i) => (
            <div
              key={i}
              className={`h-2 w-2 rounded-full ${i <= step ? 'bg-primary-600' : 'bg-gray-300'}`}
            />
          ))}
        </div>

        {/* Step 1: Level */}
        {step === 0 && (
          <div>
            <h2 className="text-lg font-semibold mb-2">Target Level</h2>
            <p className="text-sm text-gray-500 mb-4">Select the CEFR level for this pack</p>
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
            <h2 className="text-lg font-semibold mb-2">Domain</h2>
            <p className="text-sm text-gray-500 mb-4">What topic should the pack focus on?</p>
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

        {/* Step 3: Pack Template */}
        {step === 2 && (
          <div>
            <h2 className="text-lg font-semibold mb-2">Pack Template</h2>
            <p className="text-sm text-gray-500 mb-4">What type of learning pack do you want?</p>
            <div className="grid grid-cols-2 gap-2 mb-4">
              {TEMPLATES.map((t) => (
                <button
                  key={t.value}
                  onClick={() => setPackTemplate(t.value)}
                  className={`rounded-lg border-2 px-3 py-3 text-sm font-medium text-left transition-colors ${
                    packTemplate === t.value
                      ? 'border-primary-600 bg-primary-50 text-primary-700'
                      : 'border-gray-200 hover:border-gray-300'
                  }`}
                >
                  {t.label}
                </button>
              ))}
            </div>
            <div className="flex gap-3">
              <button
                onClick={() => setStep(1)}
                className="flex-1 border border-gray-300 rounded-lg py-2.5 font-medium hover:bg-gray-50"
              >
                Back
              </button>
              <button
                disabled={!packTemplate}
                onClick={() => setStep(3)}
                className="flex-1 bg-primary-600 text-white rounded-lg py-2.5 font-medium hover:bg-primary-700 disabled:opacity-50"
              >
                Next
              </button>
            </div>
          </div>
        )}

        {/* Step 4: Daily minutes */}
        {step === 3 && (
          <div>
            <h2 className="text-lg font-semibold mb-2">Daily Minutes</h2>
            <p className="text-sm text-gray-500 mb-4">How much time per day for this pack?</p>
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
                onClick={() => setStep(2)}
                className="flex-1 border border-gray-300 rounded-lg py-2.5 font-medium hover:bg-gray-50"
              >
                Back
              </button>
              <button
                onClick={() => setStep(4)}
                className="flex-1 bg-primary-600 text-white rounded-lg py-2.5 font-medium hover:bg-primary-700"
              >
                Next
              </button>
            </div>
          </div>
        )}

        {/* Step 5: Days + Focus Skills */}
        {step === 4 && (
          <div>
            <h2 className="text-lg font-semibold mb-2">Duration & Skills</h2>
            <p className="text-sm text-gray-500 mb-4">How many days, and which skills to focus on?</p>

            <p className="text-sm font-medium text-gray-700 mb-2">Days</p>
            <div className="flex flex-wrap gap-2 mb-6">
              {DAYS.map((d) => (
                <button
                  key={d}
                  onClick={() => setDays(d)}
                  className={`rounded-full px-4 py-2 text-sm font-medium border transition-colors ${
                    days === d
                      ? 'border-primary-600 bg-primary-50 text-primary-700'
                      : 'border-gray-200 hover:border-gray-300'
                  }`}
                >
                  {d} days
                </button>
              ))}
            </div>

            <p className="text-sm font-medium text-gray-700 mb-2">Focus Skills (optional)</p>
            <div className="flex flex-wrap gap-2 mb-6">
              {SKILLS.map((s) => (
                <button
                  key={s}
                  onClick={() => toggleSkill(s)}
                  className={`rounded-full px-4 py-2 text-sm font-medium border capitalize transition-colors ${
                    focusSkills.includes(s)
                      ? 'border-primary-600 bg-primary-50 text-primary-700'
                      : 'border-gray-200 hover:border-gray-300'
                  }`}
                >
                  {s}
                </button>
              ))}
            </div>

            <div className="flex gap-3">
              <button
                onClick={() => setStep(3)}
                className="flex-1 border border-gray-300 rounded-lg py-2.5 font-medium hover:bg-gray-50"
              >
                Back
              </button>
              <button
                onClick={handleSubmit}
                className="flex-1 bg-primary-600 text-white rounded-lg py-2.5 font-medium hover:bg-primary-700"
              >
                Generate Pack
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
