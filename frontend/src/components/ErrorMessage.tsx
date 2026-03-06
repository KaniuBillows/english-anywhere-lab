export default function ErrorMessage({ message, onRetry }: { message: string; onRetry?: () => void }) {
  return (
    <div className="rounded-lg bg-danger-50 px-4 py-3 text-sm text-danger-700">
      <p>{message}</p>
      {onRetry && (
        <button
          onClick={onRetry}
          className="mt-2 text-danger-600 underline hover:text-danger-700"
        >
          Try again
        </button>
      )}
    </div>
  );
}
