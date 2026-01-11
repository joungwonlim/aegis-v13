/**
 * API Client
 * SSOT: 모든 API 호출은 이 클라이언트를 통해서만 수행
 */

const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8089'

export interface ApiError {
  message: string
  code?: string
  status: number
}

export class ApiClientError extends Error {
  status: number
  code?: string

  constructor(error: ApiError) {
    super(error.message)
    this.status = error.status
    this.code = error.code
    this.name = 'ApiClientError'
  }
}

export async function apiClient<T>(
  endpoint: string,
  options?: RequestInit
): Promise<T> {
  const url = `${API_BASE}${endpoint}`

  const res = await fetch(url, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options?.headers,
    },
  })

  if (!res.ok) {
    let errorMessage = `API Error: ${res.status}`
    try {
      const errorData = await res.json()
      errorMessage = errorData.message || errorMessage
    } catch {
      // JSON 파싱 실패 시 기본 에러 메시지 사용
    }

    throw new ApiClientError({
      message: errorMessage,
      status: res.status,
    })
  }

  return res.json()
}

// HTTP Methods Helpers
export const api = {
  get: <T>(endpoint: string, options?: RequestInit) =>
    apiClient<T>(endpoint, { ...options, method: 'GET' }),

  post: <T>(endpoint: string, body?: unknown, options?: RequestInit) =>
    apiClient<T>(endpoint, {
      ...options,
      method: 'POST',
      body: body ? JSON.stringify(body) : undefined,
    }),

  put: <T>(endpoint: string, body?: unknown, options?: RequestInit) =>
    apiClient<T>(endpoint, {
      ...options,
      method: 'PUT',
      body: body ? JSON.stringify(body) : undefined,
    }),

  delete: <T>(endpoint: string, options?: RequestInit) =>
    apiClient<T>(endpoint, { ...options, method: 'DELETE' }),
}
