/** Keeps freshness next to observed data, including the last successful observation after an error. */
export function ObservationNotice({
  observedAt,
  problem,
  ready,
}: {
  observedAt: number;
  problem: string | undefined;
  ready: boolean;
}) {
  return (
    <div className="observation-notice" role="status">
      <span>
        {!ready
          ? 'Updating view…'
          : problem
            ? 'Refresh failed · Showing previous observation'
            : 'Observed'}{' '}
        · {new Intl.DateTimeFormat('en', { timeStyle: 'medium' }).format(observedAt)}
      </span>
      {problem && <span className="observation-problem">{problem}</span>}
    </div>
  );
}
