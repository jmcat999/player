const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || ''
const AUTH_TOKEN_KEY = 'player-stats-admin-token'

export function getAuthToken() {
  return localStorage.getItem(AUTH_TOKEN_KEY) || ''
}

export function setAuthToken(token) {
  localStorage.setItem(AUTH_TOKEN_KEY, token)
}

export function clearAuthToken() {
  localStorage.removeItem(AUTH_TOKEN_KEY)
}

export async function apiGet(path, params = {}, options = {}) {
  const url = new URL(`${API_BASE_URL}${path}`, window.location.origin)
  Object.entries(params)
    .filter(([, value]) => value !== null && value !== undefined && value !== '')
    .forEach(([key, value]) => url.searchParams.set(key, value))

  const response = await fetch(url, {
    headers: authHeaders({}, options)
  })
  return parseResponse(response)
}

export async function apiPost(path, params = {}, options = {}) {
  const url = new URL(`${API_BASE_URL}${path}`, window.location.origin)
  Object.entries(params)
    .filter(([, value]) => value !== null && value !== undefined && value !== '')
    .forEach(([key, value]) => url.searchParams.set(key, value))

  const response = await fetch(url, {
    method: 'POST',
    headers: authHeaders({
      Accept: 'application/json'
    }, options)
  })
  return parseResponse(response)
}

export async function apiPostJson(path, body = {}, options = {}) {
  const url = new URL(`${API_BASE_URL}${path}`, window.location.origin)
  const response = await fetch(url, {
    method: 'POST',
    headers: authHeaders({
      Accept: 'application/json',
      'Content-Type': 'application/json'
    }, options),
    body: JSON.stringify(body)
  })
  return parseResponse(response)
}

export async function apiPutJson(path, body = {}, options = {}) {
  const url = new URL(`${API_BASE_URL}${path}`, window.location.origin)
  const response = await fetch(url, {
    method: 'PUT',
    headers: authHeaders({
      Accept: 'application/json',
      'Content-Type': 'application/json'
    }, options),
    body: JSON.stringify(body)
  })
  return parseResponse(response)
}

export async function apiDelete(path, params = {}, options = {}) {
  const url = new URL(`${API_BASE_URL}${path}`, window.location.origin)
  Object.entries(params)
    .filter(([, value]) => value !== null && value !== undefined && value !== '')
    .forEach(([key, value]) => url.searchParams.set(key, value))

  const response = await fetch(url, {
    method: 'DELETE',
    headers: authHeaders({
      Accept: 'application/json'
    }, options)
  })
  return parseResponse(response)
}

function authHeaders(headers = {}, options = {}) {
  if (options.auth === false) {
    return headers
  }
  const token = getAuthToken()
  if (!token) {
    return headers
  }
  return {
    ...headers,
    Authorization: `Bearer ${token}`
  }
}

async function parseResponse(response) {
  const payload = await response.json().catch(() => ({}))
  if (!response.ok) {
    if (response.status === 401) {
      clearAuthToken()
    }
    throw new Error(payload.message || `请求失败: ${response.status}`)
  }
  return payload
}
