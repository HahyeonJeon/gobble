import { spawn, type ChildProcessWithoutNullStreams } from 'node:child_process';
import { randomBytes } from 'node:crypto';
import { parse, ServiceHandshakeSchema } from '@gobble/contracts';

type Connection = { port: number; token: string; child: ChildProcessWithoutNullStreams };

export class ProjectServiceClient {
  private connection: Promise<Connection> | undefined;
  private child: ChildProcessWithoutNullStreams | undefined;
  private stopping = false;

  constructor(
    private readonly executable: string,
    private readonly profilePath: string,
  ) {}

  private start(): Promise<Connection> {
    if (this.stopping) return Promise.reject(new Error('Project service is stopping.'));
    if (this.connection) return this.connection;
    const token = randomBytes(32).toString('hex');
    const child = spawn(this.executable, [], {
      stdio: ['pipe', 'pipe', 'pipe'],
      windowsHide: true,
    });
    this.child = child;
    const pending = new Promise<Connection>((resolve, reject) => {
      let buffer = '';
      let settled = false;
      const timeout = setTimeout(() => fail(), 8000);
      const fail = () => {
        if (settled) return;
        settled = true;
        clearTimeout(timeout);
        child.kill();
        reject(new Error('Project service could not start.'));
      };
      child.on('error', fail);
      child.on('close', () => {
        fail();
        if (this.child === child) {
          this.child = undefined;
          this.connection = undefined;
        }
      });
      child.stdin.on('error', fail);
      // Drain diagnostics without returning native paths or process details to the renderer.
      child.stderr.on('data', () => {});
      child.stdout.on('data', (data: Buffer) => {
        if (settled) return;
        buffer += data.toString('utf8');
        if (buffer.length > 16 * 1024) {
          fail();
          return;
        }
        const newline = buffer.indexOf('\n');
        if (newline < 0) return;
        try {
          const handshake = parse(ServiceHandshakeSchema, JSON.parse(buffer.slice(0, newline)));
          settled = true;
          clearTimeout(timeout);
          resolve({ port: handshake.port, token, child });
        } catch {
          fail();
        }
      });
      child.stdin.write(
        `${JSON.stringify({ schemaVersion: 1, token, profilePath: this.profilePath })}\n`,
      );
    });
    this.connection = pending;
    return pending;
  }

  async request(path: string, method: 'GET' | 'POST' = 'GET', body?: unknown): Promise<unknown> {
    const connection = await this.start();
    const response = await fetch(`http://127.0.0.1:${connection.port}${path}`, {
      method,
      redirect: 'error',
      signal: AbortSignal.timeout(25_000),
      headers: { Authorization: `Bearer ${connection.token}`, 'Content-Type': 'application/json' },
      ...(body === undefined ? {} : { body: JSON.stringify(body) }),
    });
    const reader = response.body?.getReader();
    if (!reader) throw new Error('Project service returned no response.');
    const chunks: Uint8Array[] = [];
    let size = 0;
    try {
      while (true) {
        const part = await reader.read();
        if (part.done) break;
        size += part.value.byteLength;
        if (size > 16 * 1024 * 1024) throw new Error('Project service response is too large.');
        chunks.push(part.value);
      }
      return JSON.parse(Buffer.concat(chunks).toString('utf8')) as unknown;
    } finally {
      await reader.cancel();
    }
  }

  async stop(): Promise<void> {
    this.stopping = true;
    const child = this.child;
    if (!child || child.pid === undefined || child.exitCode !== null || child.signalCode !== null)
      return;
    await new Promise<void>((resolve) => {
      const timeout = setTimeout(() => {
        child.kill('SIGKILL');
        resolve();
      }, 10_000);
      child.once('close', () => {
        clearTimeout(timeout);
        resolve();
      });
      child.stdin.end();
    });
  }
}
