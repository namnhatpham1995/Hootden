type MascotProps = {
  size?: number;
  className?: string;
};

// Chibi owl, eight primitive shapes: two ear tufts, the body ellipse, two
// eye circles with pupil dots, and a beak triangle. Same currentColor line
// art convention as Bear.
export function Owl({ size = 96, className }: MascotProps) {
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
      aria-label="Owl"
    >
      <path d="M20 26 L28 12 L34 24 Z" />
      <path d="M80 26 L72 12 L66 24 Z" />
      <ellipse cx="50" cy="56" rx="32" ry="34" />
      <circle cx="38" cy="48" r="10" />
      <circle cx="62" cy="48" r="10" />
      <circle cx="38" cy="48" r="3" fill="currentColor" stroke="none" />
      <circle cx="62" cy="48" r="3" fill="currentColor" stroke="none" />
      <path d="M46 60 L54 60 L50 68 Z" fill="currentColor" stroke="none" />
    </svg>
  );
}
