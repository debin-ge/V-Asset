const ADMIN_BASE_PATH = "/admin-console";
const ADMIN_API_PREFIX = "/api/v1/admin";

export function buildAdminApiPath(path: string) {
  if (!path.startsWith(ADMIN_API_PREFIX)) {
    return path;
  }

  if (path.startsWith(`${ADMIN_BASE_PATH}${ADMIN_API_PREFIX}`)) {
    return path;
  }

  return `${ADMIN_BASE_PATH}${path}`;
}
