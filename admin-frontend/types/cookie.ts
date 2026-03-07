export interface CookieInfo {
  id: number;
  platform: string;
  name: string;
  content: string;
  status: number;
  expire_at?: string;
  frozen_until?: string;
  freeze_seconds: number;
  last_used_at?: string;
  use_count: number;
  success_count: number;
  fail_count: number;
  created_at: string;
  updated_at: string;
}

export interface CookieListResponse {
  total: number;
  page: number;
  page_size: number;
  items: CookieInfo[];
}

