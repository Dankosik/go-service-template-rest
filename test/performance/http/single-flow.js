import { check } from 'k6';
import http from 'k6/http';
import { Trend } from 'k6/metrics';

function required(name) {
  const value = __ENV[name];
  if (value === undefined || value === '') {
    throw new Error(`${name} is required`);
  }
  return value;
}

function positiveInteger(name) {
  const raw = required(name);
  const value = Number(raw);
  if (!Number.isInteger(value) || value <= 0) {
    throw new Error(`${name} must be a positive integer`);
  }
  return value;
}

function nonNegativeInteger(name) {
  const raw = required(name);
  const value = Number(raw);
  if (!Number.isInteger(value) || value < 0) {
    throw new Error(`${name} must be a non-negative integer`);
  }
  return value;
}

function positiveNumber(name) {
  const raw = required(name);
  const value = Number(raw);
  if (!Number.isFinite(value) || value <= 0) {
    throw new Error(`${name} must be a positive number`);
  }
  return value;
}

function boundedRate(name) {
  const raw = required(name);
  const value = Number(raw);
  if (!Number.isFinite(value) || value < 0 || value >= 1) {
    throw new Error(`${name} must be greater than or equal to 0 and less than 1`);
  }
  return value;
}

function positiveDuration(name) {
  const value = required(name);
  if (!/^[1-9][0-9]*(ms|s|m|h)$/.test(value)) {
    throw new Error(`${name} must be a positive k6 duration such as 30s or 2m`);
  }
  return value;
}

function stageDuration(value, index) {
  if (value === '0') {
    return value;
  }
  if (typeof value !== 'string' || !/^[1-9][0-9]*(ms|s|m|h)$/.test(value)) {
    throw new Error(`HTTP_BENCH_STAGES_JSON[${index}].duration must be 0 or a positive k6 duration`);
  }
  return value;
}

function identifier(name) {
  const value = required(name);
  if (!/^[A-Za-z0-9._-]+$/.test(value)) {
    throw new Error(`${name} may contain only letters, digits, dot, underscore, and dash`);
  }
  return value;
}

function percentile(name) {
  const value = positiveNumber(name);
  if (value >= 100) {
    throw new Error(`${name} must be greater than 0 and less than 100`);
  }
  return value;
}

function stages() {
  const raw = __ENV.HTTP_BENCH_STAGES_JSON;
  if (raw === undefined || raw === '') {
    return null;
  }

  let value;
  try {
    value = JSON.parse(raw);
  } catch (error) {
    throw new Error(`HTTP_BENCH_STAGES_JSON must be a JSON array: ${error.message}`);
  }
  if (!Array.isArray(value) || value.length === 0) {
    throw new Error('HTTP_BENCH_STAGES_JSON must be a non-empty JSON array');
  }

  return value.map((stage, index) => {
    if (stage === null || Array.isArray(stage) || typeof stage !== 'object') {
      throw new Error(`HTTP_BENCH_STAGES_JSON[${index}] must be an object`);
    }
    if (!Number.isInteger(stage.target) || stage.target < 0) {
      throw new Error(`HTTP_BENCH_STAGES_JSON[${index}].target must be a non-negative integer`);
    }
    return {
      target: stage.target,
      duration: stageDuration(stage.duration, index),
    };
  });
}

function method() {
  const value = (__ENV.HTTP_BENCH_METHOD || 'GET').toUpperCase();
  if (!['GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'HEAD', 'OPTIONS'].includes(value)) {
    throw new Error('HTTP_BENCH_METHOD must be a standard HTTP method');
  }
  return value;
}

function headers() {
  if (!__ENV.HTTP_BENCH_HEADERS_JSON) {
    return {};
  }

  let value;
  try {
    value = JSON.parse(__ENV.HTTP_BENCH_HEADERS_JSON);
  } catch (error) {
    throw new Error(`HTTP_BENCH_HEADERS_JSON must be a JSON object: ${error.message}`);
  }
  if (value === null || Array.isArray(value) || typeof value !== 'object') {
    throw new Error('HTTP_BENCH_HEADERS_JSON must be a JSON object');
  }
  for (const [name, headerValue] of Object.entries(value)) {
    if (typeof headerValue !== 'string') {
      throw new Error(`HTTP header ${name} must have a string value`);
    }
  }
  return value;
}

function targetURL() {
  const baseURL = required('HTTP_BENCH_BASE_URL').replace(/\/+$/, '');
  const path = required('HTTP_BENCH_PATH').replace(/^\/+/, '');
  if (!/^https?:\/\//.test(baseURL)) {
    throw new Error('HTTP_BENCH_BASE_URL must start with http:// or https://');
  }
  if (/^https?:\/\/[^/]*@/.test(baseURL)) {
    throw new Error('HTTP_BENCH_BASE_URL must not contain credentials');
  }
  return `${baseURL}/${path}`;
}

const config = {
  workloadId: identifier('HTTP_BENCH_WORKLOAD_ID'),
  targetURL: targetURL(),
  method: method(),
  headers: headers(),
  body: __ENV.HTTP_BENCH_BODY || null,
  stages: stages(),
  preAllocatedVUs: positiveInteger('HTTP_BENCH_PRE_ALLOCATED_VUS'),
  timeout: positiveDuration('HTTP_BENCH_TIMEOUT'),
  latencyPercentile: percentile('HTTP_BENCH_LATENCY_PERCENTILE'),
  latencyMilliseconds: positiveNumber('HTTP_BENCH_LATENCY_MS'),
  maxErrorRate: boundedRate('HTTP_BENCH_MAX_ERROR_RATE'),
  expectedStatus: positiveInteger('HTTP_BENCH_EXPECTED_STATUS'),
};

config.executor = config.stages === null ? 'constant-arrival-rate' : 'ramping-arrival-rate';
config.rate = config.stages === null ? positiveInteger('HTTP_BENCH_RATE') : null;
config.duration = config.stages === null ? positiveDuration('HTTP_BENCH_DURATION') : null;
config.startRate = config.stages === null ? null : nonNegativeInteger('HTTP_BENCH_START_RATE');

if (config.expectedStatus < 100 || config.expectedStatus > 599) {
  throw new Error('HTTP_BENCH_EXPECTED_STATUS must be between 100 and 599');
}

http.setResponseCallback(http.expectedStatuses(config.expectedStatus));

const flowDuration = new Trend('flow_duration', true);

const scenario = {
  executor: config.executor,
  timeUnit: '1s',
  preAllocatedVUs: config.preAllocatedVUs,
  gracefulStop: '5s',
  tags: {
    flow: 'single_flow',
    workload: config.workloadId,
  },
};
if (config.stages === null) {
  scenario.rate = config.rate;
  scenario.duration = config.duration;
} else {
  scenario.startRate = config.startRate;
  scenario.stages = config.stages;
}

export const options = {
  discardResponseBodies: true,
  summaryTrendStats: ['avg', 'min', 'med', 'max', 'p(90)', 'p(95)', 'p(99)', 'count'],
  scenarios: {
    single_flow: scenario,
  },
  thresholds: {
    'flow_duration{flow:single_flow}': [
      `p(${config.latencyPercentile})<${config.latencyMilliseconds}`,
    ],
    'http_req_failed{scenario:single_flow}': [`rate<=${config.maxErrorRate}`],
    'checks{scenario:single_flow}': [`rate>=${1 - config.maxErrorRate}`],
    dropped_iterations: ['count==0'],
  },
};

export default function () {
  const startedAt = Date.now();
  const response = http.request(config.method, config.targetURL, config.body, {
    headers: config.headers,
    timeout: config.timeout,
    tags: {
      name: 'configured_http_flow',
      workload: config.workloadId,
    },
  });
  flowDuration.add(Date.now() - startedAt, {
    flow: 'single_flow',
    workload: config.workloadId,
  });

  check(response, {
    [`status is ${config.expectedStatus}`]: (result) => result.status === config.expectedStatus,
  });
}

export function handleSummary(data) {
  const summaryPath = __ENV.K6_SUMMARY_PATH || 'summary.json';
  return {
    [summaryPath]: JSON.stringify(
      {
        configuration: {
          workloadId: config.workloadId,
          targetURL: config.targetURL,
          method: config.method,
          headerNames: Object.keys(config.headers).sort(),
          bodyConfigured: config.body !== null,
          executor: config.executor,
          ratePerSecond: config.rate,
          duration: config.duration,
          startRatePerSecond: config.startRate,
          stages: config.stages,
          preAllocatedVUs: config.preAllocatedVUs,
          timeout: config.timeout,
          latencyPercentile: config.latencyPercentile,
          latencyMilliseconds: config.latencyMilliseconds,
          maxErrorRate: config.maxErrorRate,
          expectedStatus: config.expectedStatus,
        },
        summary: data,
      },
      null,
      2,
    ),
  };
}
