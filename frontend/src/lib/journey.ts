/**
 * Pre-disbursement journey for salaried first-time homebuyers.
 * HomeGauge ends at lender-ready / submitted — not post-loan servicing.
 * Fund & settle collects pre-disbursement fees via Paystack DVA.
 */

export const JOURNEY_PHASES = [
  {
    id: "qualify",
    title: "Qualify",
    summary: "Check eligibility and pick a salary-fit product.",
  },
  {
    id: "get_ready",
    title: "Get ready",
    summary: "Documents, equity gap, and known processing costs.",
  },
  {
    id: "fund_settle",
    title: "Fund & settle",
    summary: "Pay valuation, legal, and other pre-disbursement costs.",
  },
  {
    id: "submit",
    title: "Submit",
    summary: "Advisor hands a lender-ready file for underwriting and disbursement.",
  },
] as const;

export type JourneyPhaseId = (typeof JOURNEY_PHASES)[number]["id"];

export type JourneyInput = {
  assessmentCompleted: boolean;
  hasLikelyProduct: boolean;
  preferredProductId?: string | null;
  requiredDocs: number;
  acceptedRequiredDocs: number;
  sentBackDocs: number;
  caseStatus?: string | null;
  fundingSettled?: boolean;
  fundingEnabled?: boolean;
};

export type JourneyPhaseState = {
  id: JourneyPhaseId;
  title: string;
  summary: string;
  locked?: boolean;
  state: "done" | "current" | "upcoming" | "locked";
};

export function deriveJourney(input: JourneyInput): {
  phases: JourneyPhaseState[];
  currentId: JourneyPhaseId;
  nextHint: string;
} {
  const docsReady =
    input.requiredDocs > 0 &&
    input.acceptedRequiredDocs === input.requiredDocs &&
    input.sentBackDocs === 0;

  const qualifyDone = input.assessmentCompleted && (input.hasLikelyProduct || Boolean(input.preferredProductId));
  const readyDone = qualifyDone && docsReady && Boolean(input.preferredProductId);
  const fundingUnlocked = Boolean(input.preferredProductId);
  const fundingDone = Boolean(input.fundingSettled);

  const status = input.caseStatus || "";
  const submitDone = ["SUBMITTED_TO_LENDER", "LENDER_REVIEW", "APPROVED", "REJECTED", "COMPLETED"].includes(status);

  let currentId: JourneyPhaseId = "qualify";
  if (submitDone || status === "READY_FOR_SUBMISSION") currentId = "submit";
  else if (readyDone && fundingUnlocked && !fundingDone) currentId = "fund_settle";
  else if (readyDone && (fundingDone || !fundingUnlocked)) currentId = "submit";
  else if (qualifyDone) currentId = "get_ready";
  else currentId = "qualify";

  const phases: JourneyPhaseState[] = JOURNEY_PHASES.map((p) => {
    if (p.id === "qualify") {
      return { ...p, state: qualifyDone ? "done" : currentId === "qualify" ? "current" : "upcoming" };
    }
    if (p.id === "get_ready") {
      return {
        ...p,
        state: readyDone || submitDone || currentId === "fund_settle" || currentId === "submit" ? "done" : currentId === "get_ready" ? "current" : "upcoming",
      };
    }
    if (p.id === "fund_settle") {
      if (!fundingUnlocked) {
        return { ...p, locked: true, state: "locked" };
      }
      return {
        ...p,
        state: fundingDone || submitDone ? "done" : currentId === "fund_settle" ? "current" : "upcoming",
      };
    }
    return {
      ...p,
      state: submitDone ? "done" : currentId === "submit" ? "current" : "upcoming",
    };
  });

  let nextHint = "Start with eligibility for salaried first-time buyers.";
  if (currentId === "qualify" && !input.assessmentCompleted) {
    nextHint = "Complete eligibility so we can match salary-fit products.";
  } else if (currentId === "qualify") {
    nextHint = "Choose a product that fits your salary profile.";
  } else if (currentId === "get_ready" && input.sentBackDocs > 0) {
    nextHint = "Replace the documents your advisor sent back.";
  } else if (currentId === "get_ready" && input.acceptedRequiredDocs < input.requiredDocs) {
    nextHint = "Upload remaining required documents for this product.";
  } else if (currentId === "get_ready") {
    nextHint = "Confirm product, equity readiness, and documents with your advisor.";
  } else if (currentId === "fund_settle") {
    nextHint = "Open your case collection account and fund outstanding pre-disbursement fees.";
  } else if (currentId === "submit" && status === "READY_FOR_SUBMISSION") {
    nextHint = "Your file is ready — admin will release it to a lender for underwriting.";
  } else if (currentId === "submit" && !submitDone) {
    nextHint = "Your advisor is preparing the file for lender submission (pre-disbursement).";
  } else if (submitDone) {
    nextHint = "With the lender for underwriting. Watch for anything still needed before disbursement.";
  }

  return { phases, currentId, nextHint };
}

export type ReadinessCost = {
  key: string;
  label: string;
  amount: number | null;
  note: string;
  when: "before_approval" | "at_offer" | "before_disbursement";
};

/** Build display-only pre-disbursement cost list from product fees (fallback when funding API not loaded). */
export function readinessCostsFromProduct(p: {
  processing_fee?: number | null;
  valuation_fee?: number | null;
  legal_fee?: number | null;
  min_equity_pct?: number | null;
}): ReadinessCost[] {
  const items: ReadinessCost[] = [
    {
      key: "equity",
      label: "Equity / deposit",
      amount: null,
      note:
        p.min_equity_pct != null
          ? `Plan for about ${p.min_equity_pct}% of the property price before disbursement (vendor/lender path — not through HomeGauge yet).`
          : "Confirm the equity % with the lender for this product.",
      when: "before_disbursement",
    },
  ];
  if (p.valuation_fee != null && p.valuation_fee > 0) {
    items.push({
      key: "valuation",
      label: "Valuation fee",
      amount: p.valuation_fee,
      note: "Usually paid to an approved valuer during underwriting, before disbursement.",
      when: "before_approval",
    });
  } else {
    items.push({
      key: "valuation",
      label: "Valuation fee",
      amount: null,
      note: "Amount is set by the lender’s approved valuers — confirm before booking.",
      when: "before_approval",
    });
  }
  if (p.legal_fee != null && p.legal_fee > 0) {
    items.push({
      key: "legal",
      label: "Legal / search fees",
      amount: p.legal_fee,
      note: "Title search, perfection, and related legal costs — pre-disbursement.",
      when: "before_approval",
    });
  }
  if (p.processing_fee != null && p.processing_fee > 0) {
    items.push({
      key: "processing",
      label: "Processing / originating fee",
      amount: p.processing_fee,
      note: "Often due at offer acceptance — still before disbursement. Confirm with the lender.",
      when: "at_offer",
    });
  }
  items.push({
    key: "repayment_setup",
    label: "Repayment mandate setup",
    amount: null,
    note: "Salary domicile and/or direct debit / GSI is usually a condition precedent to disbursement — arranged with the mortgage bank, not via HomeGauge.",
    when: "before_disbursement",
  });
  return items;
}

export type FundingSnapshot = {
  enabled: boolean;
  paystack_public_key?: string;
  account: {
    account_number: string;
    account_name: string;
    bank_name: string;
    bank_slug: string;
    currency_code: string;
    status: string;
  } | null;
  obligations: {
    id: string;
    obligation_key: string;
    label: string;
    amount: number | null;
    amount_received: number;
    due_phase: string;
    collectable: boolean;
    status: string;
    note: string;
  }[];
  movements: {
    id: string;
    amount: number;
    paystack_reference: string;
    created_at: string;
  }[];
  total_due: number;
  total_received: number;
  total_outstanding: number;
  all_settled: boolean;
  preferred_product_name?: string;
};
