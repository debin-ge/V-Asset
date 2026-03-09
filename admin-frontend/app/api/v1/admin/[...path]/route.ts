import type { NextRequest } from "next/server";

const apiGatewayOrigin =
  process.env.API_GATEWAY_INTERNAL_URL ||
  process.env.NEXT_PUBLIC_API_BASE_URL ||
  "http://localhost:8080";

async function proxy(request: NextRequest, path: string[]) {
  const targetUrl = new URL(`/api/v1/admin/${path.join("/")}`, apiGatewayOrigin);
  targetUrl.search = request.nextUrl.search;

  const headers = new Headers();
  const contentType = request.headers.get("content-type");
  if (contentType) {
    headers.set("content-type", contentType);
  }

  const cookie = request.headers.get("cookie");
  if (cookie) {
    headers.set("cookie", cookie);
  }

  const userAgent = request.headers.get("user-agent");
  if (userAgent) {
    headers.set("user-agent", userAgent);
  }

  const body =
    request.method === "GET" || request.method === "HEAD"
      ? undefined
      : await request.arrayBuffer();

  const upstream = await fetch(targetUrl, {
    method: request.method,
    headers,
    body,
    redirect: "manual",
    cache: "no-store",
  });

  const responseHeaders = new Headers();
  copyHeaderIfPresent(upstream.headers, responseHeaders, "content-type");
  copyHeaderIfPresent(upstream.headers, responseHeaders, "location");
  copyHeaderIfPresent(upstream.headers, responseHeaders, "vary");
  copyHeaderIfPresent(upstream.headers, responseHeaders, "access-control-allow-origin");
  copyHeaderIfPresent(upstream.headers, responseHeaders, "access-control-allow-methods");
  copyHeaderIfPresent(upstream.headers, responseHeaders, "access-control-allow-headers");
  copyHeaderIfPresent(upstream.headers, responseHeaders, "access-control-allow-credentials");
  copyHeaderIfPresent(upstream.headers, responseHeaders, "access-control-max-age");
  copyHeaderIfPresent(upstream.headers, responseHeaders, "access-control-expose-headers");

  // Node fetch may expose Set-Cookie via getSetCookie() instead of get("set-cookie").
  const getSetCookie = (
    upstream.headers as Headers & { getSetCookie?: () => string[] }
  ).getSetCookie;
  const setCookies = typeof getSetCookie === "function"
    ? getSetCookie.call(upstream.headers)
    : [];

  if (setCookies.length > 0) {
    for (const value of setCookies) {
      responseHeaders.append("set-cookie", value);
    }
  } else {
    const setCookie = upstream.headers.get("set-cookie");
    if (setCookie) {
      responseHeaders.append("set-cookie", setCookie);
    }
  }

  return new Response(upstream.body, {
    status: upstream.status,
    headers: responseHeaders,
  });
}

function copyHeaderIfPresent(source: Headers, target: Headers, name: string) {
  const value = source.get(name);
  if (value) {
    target.set(name, value);
  }
}

export async function GET(request: NextRequest, context: { params: Promise<{ path: string[] }> }) {
  const { path } = await context.params;
  return proxy(request, path);
}

export async function POST(request: NextRequest, context: { params: Promise<{ path: string[] }> }) {
  const { path } = await context.params;
  return proxy(request, path);
}

export async function PUT(request: NextRequest, context: { params: Promise<{ path: string[] }> }) {
  const { path } = await context.params;
  return proxy(request, path);
}

export async function PATCH(request: NextRequest, context: { params: Promise<{ path: string[] }> }) {
  const { path } = await context.params;
  return proxy(request, path);
}

export async function DELETE(request: NextRequest, context: { params: Promise<{ path: string[] }> }) {
  const { path } = await context.params;
  return proxy(request, path);
}

export async function OPTIONS(request: NextRequest, context: { params: Promise<{ path: string[] }> }) {
  const { path } = await context.params;
  return proxy(request, path);
}
