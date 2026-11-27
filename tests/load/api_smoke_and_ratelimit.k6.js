import http from 'k6/http'
import { check } from 'k6'

export const options = {
  scenarios: {
    health_smoke: {
      executor: 'constant-vus',
      vus: 10,
      duration: '5s',
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<200'],
  },
}

const BASE_URL = __ENV.API_BASE_URL || 'http://127.0.0.1:8080'

export default function () {
  const res = http.get(`${BASE_URL}/api/v1/health`)
  check(res, {
    'status is 200 or 429 under rate limit': (r) => r.status === 200 || r.status === 429,
  })
}
