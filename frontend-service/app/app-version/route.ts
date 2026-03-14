import { NextResponse } from "next/server";
import { getPublicRuntimeConfig } from "@/lib/runtime-config.server";

export const dynamic = "force-dynamic";
export const revalidate = 0;

export async function GET() {
  return NextResponse.json(
    { version: getPublicRuntimeConfig().appVersion },
    {
      headers: {
        "Cache-Control": "no-cache, no-store, must-revalidate",
        Pragma: "no-cache",
        Expires: "0",
      },
    }
  );
}
