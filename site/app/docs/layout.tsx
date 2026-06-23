import type { Metadata } from "next";
import DocsShell from "@/components/docs/DocsShell";

export const metadata: Metadata = {
  title: { default: "Docs · DevRites", template: "%s · DevRites docs" },
  description:
    "How DevRites works: the command map, the feature lifecycle, the Spec Drift Guard, run modes, and the architecture.",
};

export default function DocsLayout({ children }: { children: React.ReactNode }) {
  return <DocsShell>{children}</DocsShell>;
}
