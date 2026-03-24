"use client";

import * as React from "react";
import { Search } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import type { BillingAccount } from "@/types/billing";

type BillingUserFilterProps = {
  testIdPrefix: string;
  query: string;
  onQueryChange: (value: string) => void;
  onSearch: () => void | Promise<void>;
  onClear: () => void | Promise<void>;
  selectedUser: BillingAccount | null;
  candidates: BillingAccount[];
  onSelectUser: (user: BillingAccount) => void;
  clearLabel: string;
};

export function BillingUserFilter({
  testIdPrefix,
  query,
  onQueryChange,
  onSearch,
  onClear,
  selectedUser,
  candidates,
  onSelectUser,
  clearLabel,
}: BillingUserFilterProps) {
  const selectedUserId = selectedUser?.user_id ?? "";
  const selectedPrimary = selectedUser ? userPrimaryText(selectedUser) : "All users";
  const selectedSecondary = selectedUser
    ? selectedUser.email && selectedUser.email !== selectedPrimary
      ? selectedUser.email
      : selectedUser.user_id
    : "Showing global records";

  return (
    <div className="space-y-3" data-testid={testIdPrefix}>
      <div className="flex flex-col gap-3 lg:flex-row">
        <div className="relative flex-1">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-slate-400" />
          <Input
            data-testid={`${testIdPrefix}-input`}
            className="pl-9"
            placeholder="Search by email, nickname, or user ID"
            value={query}
            onChange={(e) => onQueryChange(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") {
                void onSearch();
              }
            }}
          />
        </div>
        <Button data-testid={`${testIdPrefix}-search`} onClick={() => void onSearch()}>
          Search User
        </Button>
        <Button
          data-testid={`${testIdPrefix}-clear`}
          variant="outline"
          onClick={() => void onClear()}
        >
          {clearLabel}
        </Button>
      </div>

      <div className="rounded-2xl border border-slate-100 bg-slate-50/70 p-3">
        <p className="text-xs uppercase tracking-wide text-slate-400">Current filter</p>
        <p className="mt-1 text-sm font-medium text-slate-900">{selectedPrimary}</p>
        <p className="text-xs text-slate-500">{selectedSecondary}</p>

        {candidates.length ? (
          <div className="mt-3 flex flex-wrap gap-2" data-testid={`${testIdPrefix}-candidates`}>
            {candidates.map((candidate) => {
              const isSelected = selectedUserId === candidate.user_id;
              return (
                <Button
                  key={candidate.user_id}
                  size="sm"
                  variant={isSelected ? "default" : "outline"}
                  data-testid={`${testIdPrefix}-candidate-${candidate.user_id}`}
                  onClick={() => onSelectUser(candidate)}
                >
                  {userPrimaryText(candidate)}
                </Button>
              );
            })}
          </div>
        ) : null}

        {query.trim() && !candidates.length ? (
          <p className="mt-3 text-xs text-slate-500">No users matched the current query.</p>
        ) : null}
      </div>
    </div>
  );
}

function userPrimaryText(user: Pick<BillingAccount, "user_id" | "nickname" | "email">) {
  return user.nickname || user.email || user.user_id;
}
