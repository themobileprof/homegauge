import { ADVISOR_STATUSES } from "@/lib/cases";

export type DocItem = {
  document_type_code: string;
  label: string;
  category: string;
  required: boolean;
  instructions: string;
  status: string;
  document_id?: string;
  uploaded_at?: string;
  review_notes?: string;
};

export type ProductOutcome = {
  product_id: string;
  product_name: string;
  lender_name: string;
  outcome: string;
  explanation: string;
  estimated_monthly_repayment?: number | null;
  interest_rate?: number | null;
  interest_rate_min?: number | null;
  interest_rate_max?: number | null;
  interest_rate_type?: string;
  min_equity_pct?: number | null;
};

export type Assessment = {
  id: string;
  status: string;
  input_snapshot: {
    country_code?: string;
    full_name?: string;
    date_of_birth?: string;
    age?: number;
    state_of_residence?: string;
    residency_type?: string;
    marital_status?: string;
    employment_type?: string;
    employer_name?: string;
    years_employed?: number;
    monthly_net_income?: number;
    other_monthly_income?: number;
    monthly_expenses?: number;
    existing_debt_repayments?: number;
    available_deposit?: number;
    desired_property_price?: number;
    desired_loan_amount?: number;
    preferred_tenor_years?: number;
    salary_months?: number;
    nhf_contributor_months?: number;
    willing_to_domicile_salary?: boolean;
  };
  results?: ProductOutcome[];
  readiness?: {
    total: number;
    narrative: string;
    components: { key: string; label: string; score: number; max: number; note: string }[];
  };
  best_fit_product_id?: string;
  best_fit_why?: string;
};

export type Suggestion = {
  id: string;
  suggestion_type: string;
  payload: {
    message?: string;
    actions?: string[];
    priority?: string;
    rationale?: string;
  };
  status: string;
  confidence?: number;
  created_at: string;
};

export type Note = {
  id: string;
  author_email: string;
  body: string;
  visibility: string;
  created_at: string;
};

export type CaseRow = {
  id: string;
  user_id: string;
  customer_name: string;
  customer_email: string;
  status: string;
  next_action_text: string;
  preferred_product_id?: string | null;
  preferred_product_name?: string;
  lender_id?: string | null;
  lender_name?: string;
  lender_has_account?: boolean;
  assigned_advisor_name?: string;
  assigned_advisor_email?: string;
  assessment_id?: string | null;
  created_at: string;
  updated_at: string;
};

export type CaseFile = {
  case: CaseRow;
  notes: Note[];
  suggestions: Suggestion[];
  documents: DocItem[];
  assessment: Assessment | null;
};

export const STAGES = [
  { id: "situation", n: "01", title: "Situation" },
  { id: "documents", n: "02", title: "Documents" },
  { id: "products", n: "03", title: "Product fit" },
  { id: "lender", n: "04", title: "Lender" },
  { id: "work", n: "05", title: "Work list" },
  { id: "notes", n: "06", title: "Notes" },
  { id: "advance", n: "07", title: "Advance the file" },
] as const;

const ACTION_LABELS: Record<string, string> = {
  request_assessment: "Ask the buyer to finish eligibility",
  confirm_best_fit_product: "Confirm the best-fit product with the buyer",
  review_eligibility_inputs: "Recheck income, deposit, and tenor",
  verify_salary_pattern: "Verify ~6 months of salary credits",
  discuss_deposit_gap: "Talk through the deposit gap",
  chase_missing_documents: "Chase missing documents",
  collect_salary_statements: "Collect 6-month salary statements",
  review_uploaded_documents: "Review uploads on this file",
  request_document_replacements: "Ask for replacement documents",
};

export function actionLabel(code: string) {
  return ACTION_LABELS[code] || code.replaceAll("_", " ");
}

export function isWorkingStatus(status: string) {
  return (ADVISOR_STATUSES as readonly string[]).includes(status);
}

export function needsReview(status: string) {
  return status === "uploaded" || status === "under_review";
}

export function isSentBack(status: string) {
  return status === "rejected" || status === "requires_replacement";
}

export function suggestedWorkingStatus(docs: DocItem[]) {
  const required = docs.filter((d) => d.required);
  const pool = required.length ? required : docs;
  if (pool.some((d) => d.status === "not_started")) return "DOCUMENTS_PENDING";
  if (pool.some((d) => isSentBack(d.status))) return "ADDITIONAL_INFORMATION_REQUIRED";
  if (pool.some((d) => needsReview(d.status))) return "DOCUMENTS_UNDER_REVIEW";
  if (pool.length && pool.every((d) => d.status === "accepted")) return "READY_FOR_SUBMISSION";
  return "ADDITIONAL_INFORMATION_REQUIRED";
}

export function deriveFile(data: CaseFile) {
  const docs = data.documents || [];
  const required = docs.filter((d) => d.required);
  const missingRequired = required.filter((d) => d.status === "not_started");
  const pendingReview = docs.filter((d) => needsReview(d.status) && d.document_id);
  const sentBack = docs.filter((d) => isSentBack(d.status));
  const acceptedRequired = required.filter((d) => d.status === "accepted");
  const allRequiredAccepted = required.length > 0 && acceptedRequired.length === required.length;
  const assessment = data.assessment;
  const in_ = assessment?.input_snapshot;
  const results = assessment?.results || [];
  const likely = results.filter((r) => r.outcome === "likely_eligible" || r.outcome === "potentially_eligible");
  const pendingWork = (data.suggestions || []).filter((s) => s.status === "pending");
  const blockers: string[] = [];
  if (!assessment || assessment.status !== "completed") {
    blockers.push("No completed eligibility assessment.");
  }
  if (missingRequired.length) {
    blockers.push(`${missingRequired.length} required document${missingRequired.length === 1 ? "" : "s"} still missing.`);
  }
  if (pendingReview.length) {
    blockers.push(`${pendingReview.length} upload${pendingReview.length === 1 ? "" : "s"} waiting for your review.`);
  }
  if (sentBack.length) {
    blockers.push(`${sentBack.length} document${sentBack.length === 1 ? "" : "s"} sent back for replacement.`);
  }
  if (assessment && assessment.status === "completed" && results.length === 0) {
    blockers.push("No product outcomes on the assessment.");
  }
  if (assessment && likely.length === 0 && results.length > 0) {
    blockers.push("No likely product fit yet — confirm figures before handing up.");
  }
  if (!data.case.preferred_product_id) {
    blockers.push("Choose which product this file is going to.");
  }

  return {
    required,
    missingRequired,
    pendingReview,
    sentBack,
    acceptedRequired,
    allRequiredAccepted,
    assessment,
    input: in_,
    results,
    likely,
    pendingWork,
    blockers,
    canMarkReady: blockers.length === 0 && isWorkingStatus(data.case.status),
    suggestedStatus: suggestedWorkingStatus(docs),
    readyCopy: "File is complete. Waiting for admin to release to a lender.",
    stageDone: {
      situation: Boolean(assessment && in_?.monthly_net_income),
      documents: allRequiredAccepted,
      products: likely.length > 0,
      lender: Boolean(data.case.preferred_product_id),
      work: pendingWork.length === 0,
      notes: (data.notes || []).length > 0,
      advance: data.case.status === "READY_FOR_SUBMISSION" || data.case.status === "SUBMITTED_TO_LENDER" || data.case.status === "LENDER_REVIEW",
    },
  };
}

export type FileState = ReturnType<typeof deriveFile>;
