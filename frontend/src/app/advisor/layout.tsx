"use client";

import { RequireRole } from "@/components/require-role";
import type { Role } from "@/lib/auth";

const ROLES: Role[] = ["ADVISOR"];

export default function AdvisorLayout({ children }: { children: React.ReactNode }) {
  return <RequireRole roles={ROLES}>{children}</RequireRole>;
}
