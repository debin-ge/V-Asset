"use client"

import * as React from "react"
import { AccountOverview } from "./AccountOverview"
import { BillingStatements } from "./BillingStatements"

export function Stats() {
    return (
        <div className="space-y-6">
            <AccountOverview />
            <BillingStatements />
        </div>
    )
}
