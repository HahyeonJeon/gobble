import { spawn, execFile, type ChildProcessWithoutNullStreams } from 'node:child_process';
import { mkdir } from 'node:fs/promises';
import { isAbsolute } from 'node:path';
import { promisify } from 'node:util';
import { AppProblem } from '../problem';
import { CODEX_VERSION, discussionPolicy } from './policy';

export type Notification = { method: string; params: unknown };
export type ServerRequestHandler = (params: unknown, signal: AbortSignal) => Promise<unknown>;
type Pending = {
  resolve: (value: unknown) => void;
  reject: (error: Error) => void;
  timer: NodeJS.Timeout;
};
export interface RpcTransport {
  request(method: string, params?: unknown): Promise<unknown>;
  onRequest?(method: string, handler: ServerRequestHandler): () => void;
  onNotification(listener: (event: Notification) => void): () => void;
  onDisconnected(listener: () => void): () => void;
  start(): Promise<void>;
  stop(): Promise<void>;
}
const unavailable = () =>
  new AppProblem(
    'runtime_unavailable',
    'The Codex connection was lost. Check status before sending another message.',
  );
const frameLimit = 8 * 1024 * 1024;

export class CodexTransport implements RpcTransport {
  private child: ChildProcessWithoutNullStreams | undefined;
  private starting: Promise<void> | undefined;
  private nextId = 0;
  private readonly handlers = new Map<string, ServerRequestHandler>();
  private readonly callbacks = new Set<AbortController>();
  private pending = new Map<number, Pending>();
  private notifications = new Set<(event: Notification) => void>();
  private disconnections = new Set<() => void>();
  constructor(
    private readonly executable: string,
    private readonly home: string,
    private readonly cwd: string,
    private readonly timeoutMs = 20000,
    private readonly callbackTimeoutMs = 15000,
  ) {}

  onRequest(method: string, handler: ServerRequestHandler): () => void {
    if (this.handlers.has(method)) throw new Error('Server request handler already registered.');
    this.handlers.set(method, handler);
    return () => {
      this.handlers.delete(method);
    };
  }
  onNotification(listener: (event: Notification) => void): () => void {
    this.notifications.add(listener);
    return () => this.notifications.delete(listener);
  }
  onDisconnected(listener: () => void): () => void {
    this.disconnections.add(listener);
    return () => this.disconnections.delete(listener);
  }
  start(): Promise<void> {
    if (this.starting) return this.starting;
    if (this.child) return Promise.resolve();
    const started = this.launch();
    this.starting = started.finally(() => {
      this.starting = undefined;
    });
    return this.starting;
  }
  private async launch(): Promise<void> {
    if (!isAbsolute(this.executable))
      throw new AppProblem(
        'runtime_unavailable',
        'Install the supported Codex runtime to connect your account.',
      );
    const env: NodeJS.ProcessEnv = { CODEX_HOME: this.home, LANG: 'en_US.UTF-8' };
    for (const key of ['HOME', 'PATH', 'TMPDIR', 'SystemRoot']) {
      if (process.env[key]) env[key] = process.env[key];
    }
    await mkdir(this.home, { recursive: true, mode: 0o700 });
    await mkdir(this.cwd, { recursive: true, mode: 0o700 });
    let version: string;
    try {
      version = (
        await promisify(execFile)(this.executable, ['--version'], {
          env,
          cwd: this.cwd,
          timeout: 8000,
          maxBuffer: 4096,
        })
      ).stdout.trim();
    } catch {
      throw new AppProblem(
        'runtime_unavailable',
        'The Codex runtime could not start. Check its installation.',
      );
    }
    if (version !== 'codex-cli ' + CODEX_VERSION)
      throw new AppProblem(
        'incompatible_runtime',
        'This app requires Codex ' + CODEX_VERSION + '.',
      );
    const args = [
      'app-server',
      '--listen',
      'stdio://',
      ...Object.entries(discussionPolicy).flatMap(([key, value]) => [
        '-c',
        key + '=' + JSON.stringify(value),
      ]),
    ];
    const child = spawn(this.executable, args, { env, cwd: this.cwd, stdio: 'pipe' });
    this.child = child;
    let buffer = Buffer.alloc(0);
    child.stderr.resume(); // Never forward provider diagnostics or credentials into UI/logs.
    child.stdin.on('error', () => this.fail(child));
    child.on('error', () => this.fail(child));
    child.on('exit', () => this.fail(child));
    child.stdout.on('data', (chunk: Buffer) => {
      if (this.child !== child) return;
      buffer = Buffer.concat([buffer, chunk]);
      let end: number;
      while ((end = buffer.indexOf(10)) >= 0) {
        if (end > frameLimit) {
          this.fail(child);
          return;
        }
        const line = buffer.subarray(0, end);
        buffer = buffer.subarray(end + 1);
        try {
          this.receive(
            JSON.parse(new TextDecoder('utf-8', { fatal: true }).decode(line)) as unknown,
          );
        } catch {
          this.fail(child);
          return;
        }
      }
      if (buffer.length > frameLimit) this.fail(child);
    });
    try {
      await this.request('initialize', {
        clientInfo: { name: 'gobble_app', title: 'Gobble', version: '0.1.0' },
        capabilities: { experimentalApi: true },
      });
      this.write({ method: 'initialized', params: {} });
    } catch (error) {
      this.fail(child);
      throw error;
    }
  }
  private write(value: unknown): void {
    if (!this.child || this.child.stdin.destroyed) throw unavailable();
    const line = JSON.stringify(value) + '\n';
    if (Buffer.byteLength(line) > frameLimit)
      throw new AppProblem('unsupported', 'The Codex request is too large.');
    this.child.stdin.write(line);
  }
  request(method: string, params: unknown = {}): Promise<unknown> {
    if (!this.child) return Promise.reject(unavailable());
    if (this.pending.size >= 32)
      return Promise.reject(
        new AppProblem('runtime_unavailable', 'Codex is busy. Try again shortly.'),
      );
    const id = ++this.nextId;
    return new Promise((resolve, reject) => {
      const child = this.child;
      const timer = setTimeout(() => {
        if (child) this.fail(child);
      }, this.timeoutMs);
      this.pending.set(id, { resolve, reject, timer });
      try {
        this.write({ id, method, params });
      } catch (error) {
        clearTimeout(timer);
        this.pending.delete(id);
        reject(error);
      }
    });
  }
  private receive(raw: unknown): void {
    if (!raw || typeof raw !== 'object' || Array.isArray(raw)) throw new Error('Invalid frame.');
    const message = raw as Record<string, unknown>;
    if (typeof message.method === 'string') {
      if ('id' in message) {
        if (typeof message.id !== 'string' && typeof message.id !== 'number')
          throw new Error('Invalid request ID.');
        this.respond(message.id, message.method, message.params);
      } else {
        for (const listener of this.notifications)
          listener({ method: message.method, params: message.params });
      }
      return;
    }
    if (typeof message.id !== 'number') throw new Error('Missing response ID.');
    const pending = this.pending.get(message.id);
    if (!pending) return;
    this.pending.delete(message.id);
    clearTimeout(pending.timer);
    if ('error' in message)
      pending.reject(
        new AppProblem(
          'runtime_unavailable',
          'Codex could not complete this request. Check the account and connection.',
        ),
      );
    else if ('result' in message) pending.resolve(message.result);
    else {
      pending.reject(unavailable());
      throw new Error('Missing response body.');
    }
  }
  /** Callbacks never occupy an outbound-request slot or block notification delivery. */
  private respond(id: string | number, method: string, params: unknown): void {
    const child = this.child;
    const handler = this.handlers.get(method);
    if (!handler || this.callbacks.size >= 4) {
      this.write({ id, error: { code: -32601, message: 'This operation is unavailable.' } });
      return;
    }
    const cancellation = new AbortController();
    this.callbacks.add(cancellation);
    const timer = setTimeout(() => cancellation.abort(), this.callbackTimeoutMs);
    const deadline = new Promise<never>((_resolve, reject) => {
      cancellation.signal.addEventListener('abort', () => reject(unavailable()), { once: true });
    });
    void Promise.race([
      Promise.resolve().then(() => handler(params, cancellation.signal)),
      deadline,
    ])
      .then((result) => {
        if (this.child !== child || cancellation.signal.aborted) return;
        if (Buffer.byteLength(JSON.stringify(result)) > 4 * 1024 * 1024)
          throw new Error('Oversized tool result.');
        this.write({ id, result });
      })
      .catch(() => {
        if (this.child === child)
          this.write({
            id,
            error: { code: -32603, message: 'The operation expired or could not complete.' },
          });
      })
      .finally(() => {
        clearTimeout(timer);
        this.callbacks.delete(cancellation);
      })
      .catch(() => {
        if (child) this.fail(child);
      });
  }
  private fail(child: ChildProcessWithoutNullStreams): void {
    if (this.child !== child) return;
    this.child = undefined;
    for (const callback of this.callbacks) callback.abort();
    this.callbacks.clear();
    child.kill('SIGKILL');
    for (const pending of this.pending.values()) {
      clearTimeout(pending.timer);
      pending.reject(unavailable());
    }
    this.pending.clear();
    for (const listener of this.disconnections) listener();
  }
  async stop(): Promise<void> {
    if (this.starting) await this.starting.catch(() => {});
    const child = this.child;
    if (!child) return;
    await new Promise<void>((resolve) => {
      const timer = setTimeout(() => {
        this.fail(child);
        resolve();
      }, 1500);
      child.once('exit', () => {
        clearTimeout(timer);
        resolve();
      });
      child.stdin.end();
    });
  }
}
