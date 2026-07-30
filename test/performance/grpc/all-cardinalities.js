import { Counter, Trend } from 'k6/metrics';
import { Client, StatusOK, Stream } from 'k6/net/grpc';
import exec from 'k6/execution';

const PROTO_PATH = '../../../examples/grpc-reference-service/api/proto';
const PROTO_FILE = 'reference/v1/echo.proto';
const SERVICE = 'reference.v1.EchoService';

const mode = stringEnv('GRPC_BENCH_MODE', 'synthetic');
if (mode !== 'smoke' && mode !== 'synthetic') {
  throw new Error(`GRPC_BENCH_MODE must be smoke or synthetic, got ${mode}`);
}

const target = stringEnv('GRPC_BENCH_TARGET', '127.0.0.1:50051');
const workloadID = stringEnv('GRPC_BENCH_WORKLOAD_ID', 'grpc-reference-all-cardinalities');
const payloadBytes = integerEnv('GRPC_BENCH_PAYLOAD_BYTES', 64, 1, 1 << 20);
const streamMessages = integerEnv('GRPC_BENCH_STREAM_MESSAGES', 4, 1, 1024);
const vus = integerEnv('GRPC_BENCH_VUS', 1, 1, 256);
const warmupDuration = durationEnv('GRPC_BENCH_WARMUP_DURATION', '3s');
const measuredDuration = durationEnv('GRPC_BENCH_DURATION', '10s');
const rpcTimeout = durationEnv('GRPC_BENCH_RPC_TIMEOUT', '10s');
const measuredStartTime = addDurations(warmupDuration, rpcTimeout);

const operationsOffered = new Counter('grpc_bench_operations_offered');
const operationSuccesses = new Counter('grpc_bench_operation_successes');
const unarySuccesses = new Counter('grpc_bench_unary_successes');
const streamSuccesses = new Counter('grpc_bench_stream_successes');
const messagesSent = new Counter('grpc_bench_messages_sent');
const messagesReceived = new Counter('grpc_bench_messages_received');
const messagesProcessed = new Counter('grpc_bench_messages_processed');
const correctnessFailures = new Counter('grpc_bench_correctness_failures');
const terminalFailures = new Counter('grpc_bench_terminal_failures');
const timeoutFailures = new Counter('grpc_bench_timeouts');
const unaryDuration = new Trend('grpc_bench_unary_duration', true);
const streamDuration = new Trend('grpc_bench_stream_duration', true);
const messageLag = new Trend('grpc_bench_message_lag', true);

const client = new Client();
client.load([PROTO_PATH], PROTO_FILE);
let connected = false;

const cardinalities = ['unary', 'server_stream', 'client_stream', 'bidi'];
const thresholds = {
  grpc_bench_correctness_failures: ['count==0'],
  grpc_bench_terminal_failures: ['count==0'],
  grpc_bench_timeouts: ['count==0'],
};
for (const cardinality of cardinalities) {
  thresholds[`grpc_bench_operation_successes{cardinality:${cardinality}}`] = ['count>0'];
  thresholds[`grpc_bench_messages_sent{cardinality:${cardinality}}`] = ['count>0'];
  thresholds[`grpc_bench_messages_received{cardinality:${cardinality}}`] = ['count>0'];
  thresholds[`grpc_bench_messages_processed{cardinality:${cardinality}}`] = ['count>0'];
}
thresholds['grpc_bench_unary_successes{cardinality:unary}'] = ['count>0'];
for (const cardinality of ['server_stream', 'client_stream', 'bidi']) {
  thresholds[`grpc_bench_stream_successes{cardinality:${cardinality}}`] = ['count>0'];
}
if (mode === 'synthetic') {
  thresholds.dropped_iterations = ['count==0'];
}

export const options =
  mode === 'smoke'
    ? {
        scenarios: {
          smoke: {
            executor: 'per-vu-iterations',
            exec: 'smoke',
            vus: 1,
            iterations: 1,
            maxDuration: '45s',
          },
        },
        thresholds,
        summaryTrendStats: ['min', 'med', 'p(95)', 'p(99)', 'max', 'count'],
      }
    : {
        scenarios: {
          warmup: {
            executor: 'constant-vus',
            exec: 'warmup',
            vus,
            duration: warmupDuration,
            gracefulStop: rpcTimeout,
          },
          measured: {
            executor: 'constant-vus',
            exec: 'measured',
            vus,
            duration: measuredDuration,
            startTime: measuredStartTime,
            gracefulStop: rpcTimeout,
          },
        },
        thresholds,
        summaryTrendStats: ['min', 'med', 'p(95)', 'p(99)', 'max', 'count'],
      };

export async function smoke() {
  ensureConnected();
  await unary(true);
  await serverStream(true);
  await clientStream(true);
  await bidiStream(true);
}

export async function warmup() {
  ensureConnected();
  await operationFor(exec.scenario.iterationInTest, false);
}

export async function measured() {
  ensureConnected();
  await operationFor(exec.scenario.iterationInTest, true);
}

export function handleSummary(data) {
  const summaryPath = stringEnv('K6_SUMMARY_PATH', '/artifacts/summary.json');
  const evidence = {
    evidence_level: mode,
    workload_id: workloadID,
    target,
    plaintext: true,
    payload_bytes: payloadBytes,
    stream_messages: streamMessages,
    virtual_users: mode === 'smoke' ? 1 : vus,
    connections_per_vu: 1,
    warmup_duration: mode === 'smoke' ? 'none' : warmupDuration,
    warmup_completion_budget: mode === 'smoke' ? 'none' : rpcTimeout,
    measured_start_time: mode === 'smoke' ? 'immediate' : measuredStartTime,
    measured_duration: mode === 'smoke' ? 'one iteration' : measuredDuration,
    rpc_timeout: rpcTimeout,
    decision_grade_observables: {
      server_cpu: 'unavailable',
      server_resident_memory: 'unavailable',
      server_heap_and_gc: 'unavailable',
      admission_utilization_and_rejection: 'unavailable',
      server_connection_and_stream_counts: 'unavailable',
      server_network_utilization: 'unavailable',
      load_generator_cpu_and_network_headroom: 'unavailable',
    },
    k6: data,
  };
  return {
    [summaryPath]: JSON.stringify(evidence, null, 2),
    stdout: `gRPC ${mode} summary written to ${summaryPath}\n`,
  };
}

async function operationFor(iteration, record) {
  switch (iteration % cardinalities.length) {
    case 0:
      return unary(record);
    case 1:
      return serverStream(record);
    case 2:
      return clientStream(record);
    default:
      return bidiStream(record);
  }
}

function ensureConnected() {
  if (connected) {
    return;
  }
  client.connect(target, {
    plaintext: true,
    timeout: rpcTimeout,
  });
  connected = true;
}

async function unary(record) {
  const tags = { cardinality: 'unary' };
  const value = payload('unary');
  beginOperation(record, tags, 1);
  const startedAt = Date.now();
  let response;
  try {
    response = client.invoke(`${SERVICE}/Unary`, { value }, { timeout: rpcTimeout });
  } catch (error) {
    finishFailure(record, tags, error, false);
    return;
  } finally {
    if (record) {
      unaryDuration.add(Date.now() - startedAt, tags);
    }
  }

  if (!response || response.status !== StatusOK) {
    finishFailure(record, tags, response && response.error, false);
    return;
  }
  if (record) {
    messagesReceived.add(1, tags);
  }
  if (!response.message || response.message.value !== value) {
    finishFailure(record, tags, new Error('unary payload mismatch'), true);
    return;
  }
  if (record) {
    messagesProcessed.add(1, tags);
    unarySuccesses.add(1, tags);
    operationSuccesses.add(1, tags);
  }
}

async function serverStream(record) {
  const tags = { cardinality: 'server_stream' };
  const value = payload('server-stream');
  beginOperation(record, tags, 1);
  const startedAt = Date.now();
  const result = await runStream(
    `${SERVICE}/ServerStream`,
    (stream) => {
      stream.write({ value, count: streamMessages });
      stream.end();
    },
    (message, index) => message.value === value && message.sequence === index + 1,
    record,
    tags,
  );
  finishStream(record, tags, startedAt, result, streamMessages);
}

async function clientStream(record) {
  const tags = { cardinality: 'client_stream' };
  const expected = [];
  for (let index = 0; index < streamMessages; index += 1) {
    expected.push(payload(`client-${index}`));
  }
  beginOperation(record, tags, streamMessages);
  const startedAt = Date.now();
  const result = await runStream(
    `${SERVICE}/ClientStream`,
    (stream) => {
      for (const value of expected) {
        stream.write({ value });
      }
      stream.end();
    },
    (message) =>
      Array.isArray(message.values) &&
      message.values.length === expected.length &&
      message.values.every((value, index) => value === expected[index]),
    record,
    tags,
  );
  finishStream(record, tags, startedAt, result, 1);
}

async function bidiStream(record) {
  const tags = { cardinality: 'bidi' };
  const expected = [];
  for (let index = 0; index < streamMessages; index += 1) {
    expected.push({
      sequence: index + 1,
      sent_at: Date.now(),
      payload: payload(`bidi-${index}`),
    });
  }
  beginOperation(record, tags, streamMessages);
  const startedAt = Date.now();
  const result = await runStream(
    `${SERVICE}/BidiStream`,
    (stream) => {
      for (const item of expected) {
        item.sent_at = Date.now();
        stream.write({ value: JSON.stringify(item) });
      }
      stream.end();
    },
    (message, index) => {
      let actual;
      try {
        actual = JSON.parse(message.value);
      } catch (_) {
        return false;
      }
      const want = expected[index];
      const correct =
        want &&
        actual.sequence === want.sequence &&
        actual.sent_at === want.sent_at &&
        actual.payload === want.payload;
      if (correct && record) {
        messageLag.add(Date.now() - actual.sent_at, tags);
      }
      return correct;
    },
    record,
    tags,
  );
  finishStream(record, tags, startedAt, result, streamMessages);
}

function beginOperation(record, tags, sentCount) {
  if (!record) {
    return;
  }
  operationsOffered.add(1, tags);
  messagesSent.add(sentCount, tags);
  correctnessFailures.add(0, tags);
  terminalFailures.add(0, tags);
  timeoutFailures.add(0, tags);
}

function runStream(method, writeRequests, validateMessage, record, tags) {
  return new Promise((resolve) => {
    const stream = new Stream(client, method, { timeout: rpcTimeout });
    const result = {
      ended: false,
      error: null,
      correctnessFailure: false,
      received: 0,
      processed: 0,
    };
    let resolved = false;
    const finish = () => {
      if (resolved) {
        return;
      }
      resolved = true;
      resolve(result);
    };

    stream.on('data', (message) => {
      const index = result.received;
      result.received += 1;
      if (record) {
        messagesReceived.add(1, tags);
      }
      if (validateMessage(message, index)) {
        result.processed += 1;
        if (record) {
          messagesProcessed.add(1, tags);
        }
      } else {
        result.correctnessFailure = true;
      }
    });
    stream.on('error', (error) => {
      result.error = error || new Error(`${method} failed without an error detail`);
      finish();
    });
    stream.on('end', () => {
      result.ended = true;
      finish();
    });

    try {
      writeRequests(stream);
    } catch (error) {
      result.error = error;
      finish();
    }
  });
}

function finishStream(record, tags, startedAt, result, expectedResponses) {
  if (!record) {
    return;
  }
  streamDuration.add(Date.now() - startedAt, tags);
  if (result.error || !result.ended) {
    finishFailure(record, tags, result.error, false);
    return;
  }
  if (
    result.correctnessFailure ||
    result.received !== expectedResponses ||
    result.processed !== expectedResponses
  ) {
    correctnessFailures.add(1, tags);
    return;
  }
  streamSuccesses.add(1, tags);
  operationSuccesses.add(1, tags);
}

function finishFailure(record, tags, error, correctness) {
  if (!record) {
    return;
  }
  if (correctness) {
    correctnessFailures.add(1, tags);
    return;
  }
  terminalFailures.add(1, tags);
  if (String(error || '').toLowerCase().includes('deadline')) {
    timeoutFailures.add(1, tags);
  }
}

function payload(prefix) {
  const base = `${prefix}:`;
  if (base.length >= payloadBytes) {
    return base.slice(0, payloadBytes);
  }
  return base + 'x'.repeat(payloadBytes - base.length);
}

function stringEnv(name, fallback) {
  const value = __ENV[name] || fallback;
  if (!value || value.includes('\n') || value.includes('\r')) {
    throw new Error(`${name} must be a non-empty single-line value`);
  }
  return value;
}

function integerEnv(name, fallback, minimum, maximum) {
  const raw = __ENV[name] || String(fallback);
  if (!/^[1-9][0-9]*$/.test(raw)) {
    throw new Error(`${name} must be a positive integer, got ${raw}`);
  }
  const value = Number(raw);
  if (!Number.isSafeInteger(value) || value < minimum || value > maximum) {
    throw new Error(`${name} must be in range [${minimum},${maximum}], got ${raw}`);
  }
  return value;
}

function durationEnv(name, fallback) {
  const value = __ENV[name] || fallback;
  if (!/^[1-9][0-9]*(ms|s|m)$/.test(value)) {
    throw new Error(`${name} must be a positive k6 duration using ms, s, or m, got ${value}`);
  }
  return value;
}

function addDurations(left, right) {
  return `${durationMilliseconds(left) + durationMilliseconds(right)}ms`;
}

function durationMilliseconds(value) {
  const match = /^([1-9][0-9]*)(ms|s|m)$/.exec(value);
  const amount = Number(match[1]);
  switch (match[2]) {
    case 'ms':
      return amount;
    case 's':
      return amount * 1000;
    default:
      return amount * 60 * 1000;
  }
}
