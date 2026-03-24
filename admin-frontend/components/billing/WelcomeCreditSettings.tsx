import * as React from "react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { billingApi } from "@/lib/api/billing";

export function WelcomeCreditSettings() {
  const [enabled, setEnabled] = React.useState(false);
  const [amountYuan, setAmountYuan] = React.useState("0.00");
  const [currencyCode, setCurrencyCode] = React.useState("CNY");
  const [showSavedBanner, setShowSavedBanner] = React.useState(false);
  const [isLoading, setIsLoading] = React.useState(true);
  const [isSaving, setIsSaving] = React.useState(false);

  const loadSettings = React.useCallback(async () => {
    setIsLoading(true);
    try {
      const data = await billingApi.getWelcomeCreditSettings();
      const loadedCurrencyCode = (data as { currency_code?: string }).currency_code ?? "CNY";
      setEnabled(data.enabled);
      setAmountYuan(Number(data.amount_yuan).toFixed(2));
      setCurrencyCode(loadedCurrencyCode);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to load welcome credit settings");
    } finally {
      setIsLoading(false);
    }
  }, []);

  React.useEffect(() => {
    void loadSettings();
  }, [loadSettings]);

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSaving(true);
    setShowSavedBanner(false);

    try {
      const parsedYuan = parseFloat(amountYuan);
      if (isNaN(parsedYuan) || parsedYuan < 0) {
        throw new Error("Invalid amount");
      }
      
      const updated = await billingApi.updateWelcomeCreditSettings({
        enabled,
        amount_yuan: parsedYuan.toFixed(2),
        currency_code: currencyCode || "CNY",
      });
      const updatedCurrencyCode = (updated as { currency_code?: string }).currency_code ?? currencyCode;
      
      setEnabled(updated.enabled);
      setAmountYuan(Number(updated.amount_yuan).toFixed(2));
      setCurrencyCode(updatedCurrencyCode);
      setShowSavedBanner(true);
      toast.success("Welcome credit settings saved");
      
      setTimeout(() => setShowSavedBanner(false), 3000);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to save welcome credit settings");
    } finally {
      setIsSaving(false);
    }
  };

  if (isLoading) {
    return (
      <Card className="rounded-[28px] border-border/60 bg-white/85 shadow-sm">
        <CardHeader>
          <CardTitle>Welcome Credit</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="py-4 text-center text-sm text-slate-500">Loading settings...</div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card className="rounded-[28px] border-border/60 bg-white/85 shadow-sm">
      <CardHeader>
        <CardTitle>Welcome Credit</CardTitle>
      </CardHeader>
      <CardContent>
        {showSavedBanner && (
          <div 
            data-testid="welcome-credit-saved-banner"
            className="mb-4 rounded-lg bg-green-50 p-3 text-sm text-green-700 border border-green-200"
          >
            Settings saved successfully.
          </div>
        )}
        
        <form data-testid="welcome-credit-form" onSubmit={handleSave} className="space-y-4">
          <div className="flex items-center space-x-2">
            <input
              type="checkbox"
              id="welcome-credit-enabled"
              data-testid="welcome-credit-enabled"
              checked={enabled}
              onChange={(e) => setEnabled(e.target.checked)}
              className="h-4 w-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
            />
            <label htmlFor="welcome-credit-enabled" className="text-sm font-medium text-slate-700">
              Enable Welcome Credit
            </label>
          </div>
          
          <div className="space-y-2">
            <label htmlFor="welcome-credit-amount" className="text-sm font-medium text-slate-700">
              Credit Amount (Yuan)
            </label>
            <Input
              id="welcome-credit-amount"
              data-testid="welcome-credit-amount"
              value={amountYuan}
              onChange={(e) => setAmountYuan(e.target.value)}
              disabled={!enabled}
              placeholder="e.g. 1.00"
            />
          </div>
          
          <Button 
            type="submit" 
            data-testid="welcome-credit-save" 
            disabled={isSaving}
          >
            {isSaving ? "Saving..." : "Save Settings"}
          </Button>
        </form>
      </CardContent>
    </Card>
  );
}
