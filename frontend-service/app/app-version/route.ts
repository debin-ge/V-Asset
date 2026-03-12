import { NextResponse } from "next/server";

export const dynamic = "force-dynamic";
export const revalidate = 0;

const appVersion = process.env.NEXT_PUBLIC_APP_VERSION ?? "unknown";

export async function GET() {
  return NextResponse.json(
    { version: appVersion },
    {
      headers: {
        "Cache-Control": "no-cache, no-store, must-revalidate",
        Pragma: "no-cache",
        Expires: "0",
      },
    }
  );
}
