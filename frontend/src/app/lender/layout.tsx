"use client";

import { RequireRole } from "@/components/require-role";
import type { Role } from "@/lib/auth";

const ROLES: Role[] = ["LENDER_USER"];

export default function LenderLayout({ children }: { children: React.ReactNode }) {
  return <RequireRole roles={ROLES}>{children}</RequireRole>;
}
