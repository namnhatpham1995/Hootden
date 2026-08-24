import { Bear } from "@/components/mascots/Bear";
import { Owl } from "@/components/mascots/Owl";

// Visual QA route for the design tokens, type scale, and mascots (tasks
// 7.2-7.4). Not part of the product -- toggle the OS colour scheme to
// check both themes until the in-app toggle (task 8.5) lands.
const swatches = [
  ["surface", "var(--surface)"],
  ["surface-raised", "var(--surface-raised)"],
  ["foreground", "var(--foreground)"],
  ["foreground-muted", "var(--foreground-muted)"],
  ["accent", "var(--accent)"],
  ["secondary", "var(--secondary)"],
  ["border", "var(--border)"],
] as const;

const radii = ["sm", "md", "lg", "pill"] as const;
const spacing = ["1", "2", "3", "4", "6", "8", "12"] as const;

export default function StyleGuide() {
  return (
    <main style={{ padding: "var(--space-8)", display: "flex", flexDirection: "column", gap: "var(--space-8)" }}>
      <section>
        <h2 style={{ fontFamily: "var(--font-display)" }}>Colour tokens</h2>
        <div style={{ display: "flex", gap: "var(--space-4)", flexWrap: "wrap", marginTop: "var(--space-4)" }}>
          {swatches.map(([name, value]) => (
            <div key={name} style={{ textAlign: "center" }}>
              <div
                style={{
                  width: 72,
                  height: 72,
                  borderRadius: "var(--radius-md)",
                  background: value,
                  border: "1px solid var(--border)",
                }}
              />
              <span style={{ fontFamily: "var(--font-ui)", fontSize: "0.75rem" }}>{name}</span>
            </div>
          ))}
        </div>
      </section>

      <section>
        <h2 style={{ fontFamily: "var(--font-display)" }}>Type scale</h2>
        <p style={{ fontFamily: "var(--font-display)", fontSize: "2rem", marginTop: "var(--space-4)" }}>
          Display -- Fredoka
        </p>
        <p style={{ fontFamily: "var(--font-ui)", fontSize: "1rem" }}>UI -- Nunito Sans</p>
        <p style={{ fontFamily: "var(--font-body)", fontSize: "1.125rem" }}>
          Document body -- Lora, for the reading and writing surface.
        </p>
      </section>

      <section>
        <h2 style={{ fontFamily: "var(--font-display)" }}>Radius scale</h2>
        <div style={{ display: "flex", gap: "var(--space-4)", marginTop: "var(--space-4)" }}>
          {radii.map((r) => (
            <div
              key={r}
              style={{
                width: 64,
                height: 64,
                borderRadius: `var(--radius-${r})`,
                background: "var(--surface-raised)",
                border: "1px solid var(--border)",
              }}
            />
          ))}
        </div>
      </section>

      <section>
        <h2 style={{ fontFamily: "var(--font-display)" }}>Spacing scale</h2>
        <div style={{ display: "flex", alignItems: "flex-end", gap: "var(--space-2)", marginTop: "var(--space-4)" }}>
          {spacing.map((s) => (
            <div key={s} style={{ width: `var(--space-${s})`, height: 24, background: "var(--accent)" }} />
          ))}
        </div>
      </section>

      <section>
        <h2 style={{ fontFamily: "var(--font-display)" }}>Mascots</h2>
        <div style={{ display: "flex", alignItems: "center", gap: "var(--space-6)", marginTop: "var(--space-4)" }}>
          <Bear size={24} />
          <Bear size={96} />
          <Owl size={24} />
          <Owl size={96} />
          <span style={{ color: "var(--accent)" }}>
            <Bear size={64} />
          </span>
        </div>
      </section>
    </main>
  );
}
