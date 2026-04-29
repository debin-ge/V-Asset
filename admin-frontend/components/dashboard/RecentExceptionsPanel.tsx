import Link from "next/link";

import { buttonVariants } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { DashboardStatusBadge } from "@/components/dashboard/DashboardStatusBadge";
import type { DashboardException } from "@/types/stats";

export function RecentExceptionsPanel({
  exceptions,
  loading,
}: {
  exceptions: DashboardException[];
  loading: boolean;
}) {
  return (
    <Card className="rounded-lg border-border/70 bg-white/90 shadow-sm">
      <CardHeader>
        <CardTitle>Recent Exceptions</CardTitle>
        <CardDescription>由当前统计数据派生的待关注事项。</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="overflow-hidden rounded-lg border border-border/70">
          <Table>
            <TableHeader>
              <TableRow className="bg-muted/40 hover:bg-muted/40">
                <TableHead>Area</TableHead>
                <TableHead>Severity</TableHead>
                <TableHead>Message</TableHead>
                <TableHead className="text-right">Action</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {exceptions.map((item, index) => (
                <TableRow key={`${item.area}-${index}`}>
                  <TableCell className="font-medium text-foreground">{item.area}</TableCell>
                  <TableCell>
                    <DashboardStatusBadge status={item.severity} />
                  </TableCell>
                  <TableCell className="max-w-[360px] truncate text-muted-foreground">{item.message}</TableCell>
                  <TableCell className="text-right">
                    {item.action_href ? (
                      <Link href={item.action_href} className={buttonVariants({ variant: "ghost", size: "sm" })}>
                        {item.action_label}
                      </Link>
                    ) : (
                      <span className="text-sm text-muted-foreground">{item.action_label}</span>
                    )}
                  </TableCell>
                </TableRow>
              ))}
              {exceptions.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={4} className="py-8 text-center text-sm text-muted-foreground">
                    {loading ? "Checking exceptions..." : "No active exceptions reported by the health service."}
                  </TableCell>
                </TableRow>
              ) : null}
            </TableBody>
          </Table>
        </div>
      </CardContent>
    </Card>
  );
}
