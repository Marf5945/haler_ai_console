

export default function FloatingCandidateActions({candidates = [], onSelect}) {
  if (!candidates.length) return null;
  return (
    <div className="floating-candidates">
      {candidates.slice(0, 3).map((candidate) => (
        <button
          key={candidate.id}
          type="button"
          className="floating-candidate-btn"
          onClick={() => onSelect(candidate.id)}
        >
          {candidate.label}
        </button>
      ))}
    </div>
  );
}
