"use client";

import { RequireRole } from "@/components/require-role";
import type { Role } from "@/lib/auth";

const ROLES: Role[] = ["ADMIN"];

export default function AdminLayout({ children }: { children: React.ReactNode }) {
  return <RequireRole roles={ROLES}>{children}</RequireRole>;
}
