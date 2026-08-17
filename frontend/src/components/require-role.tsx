"use client";

import { useRouter } from "next/navigation";
import { useEffect, type ReactNode } from "react";
import { homeForRole, useAuth, type Role } from "@/lib/auth";

export function RequireRole({ roles, children }: { roles: Role[]; children: ReactNode }) {
  const { user, loading } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (loading) return;
    if (!user) {
      router.replace("/login");
      return;
    }
    if (!roles.includes(user.role)) {
      router.replace(homeForRole(user.role));
    }
  }, [loading, user, roles, router]);

  if (loading || !user || !roles.includes(user.role)) {
    return <p className="px-5 py-12 text-sm text-muted">Opening your workspace…</p>;
  }

  return <>{children}</>;
}
