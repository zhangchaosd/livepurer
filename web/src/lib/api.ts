export async function api<T>(path: string, init: RequestInit = {}): Promise<T> {
  const response = await fetch(`/api/v1${path}`, {
    ...init,
    signal: init.signal ?? AbortSignal.timeout(20000),
    headers: { "Content-Type": "application/json", ...init.headers },
  });
  let body: { code: number; msg: string; data: T };
  try {
    body = await response.json();
  } catch {
    throw new Error(`服务响应异常（${response.status}），请确认程序正在运行。`);
  }
  if (!response.ok || body.code !== 0)
    throw new Error(body.msg || `请求失败（${response.status}）`);
  return body.data;
}
export const json = (method: string, data: unknown): RequestInit => ({
  method,
  body: JSON.stringify(data),
});
