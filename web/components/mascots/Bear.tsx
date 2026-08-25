type MascotProps = {
  size?: number;
  className?: string;
  sleeping?: boolean;
};

// Chibi bear head, six primitive shapes: two ear circles, the head circle,
// a snout ellipse, two eye dots, and a nose dot with a short mouth line.
// Line art in currentColor so it stays legible at 24px and follows
// whichever theme fill token the caller sets via CSS `color`. `sleeping`
// swaps the eye dots for closed-eye arcs, for the empty-state mascot spot
// (see design.md's mascot-placement restriction).
export function Bear({ size = 96, className, sleeping = false }: MascotProps) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 100 100"
      fill="none"
      stroke="currentColor"
      strokeWidth={4}
      strokeLinecap="round"
      strokeLinejoin="round"
      className={className}
      role="img"
      aria-label={sleeping ? "Sleeping bear" : "Bear"}
    >
      <circle cx="28" cy="28" r="12" />
      <circle cx="72" cy="28" r="12" />
      <circle cx="50" cy="55" r="30" />
      <ellipse cx="50" cy="63" rx="14" ry="10" />
      {sleeping ? (
        <>
          <path d="M35 50 q5 4 10 0" />
          <path d="M55 50 q5 4 10 0" />
        </>
      ) : (
        <>
          <circle cx="40" cy="50" r="3" fill="currentColor" stroke="none" />
          <circle cx="60" cy="50" r="3" fill="currentColor" stroke="none" />
        </>
      )}
      <circle cx="50" cy="58" r="3" fill="currentColor" stroke="none" />
      <line x1="50" y1="61" x2="50" y2="67" />
      {sleeping && (
        <text x="66" y="30" fontSize="14" fontFamily="var(--font-ui)" stroke="none" fill="currentColor">
          z
        </text>
      )}
    </svg>
  );
}
