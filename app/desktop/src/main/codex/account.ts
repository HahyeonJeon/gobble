import { randomUUID } from 'node:crypto';
import { parse, CodexModelSchema, type AccountStatus, type CodexModel } from '@gobble/contracts';
import { AtomicStateFile } from '../workspace/storage';
import { AppProblem } from '../problem';
import type { RpcTransport } from './transport';
import { array, id, object, string, malformed } from './protocol';
import { CODEX_VERSION, loginURL } from './policy';

type Session = { schemaVersion: 1; sessionId: string | null };
export class CodexAccount {
  private state: AccountStatus = {
    runtime: 'disconnected',
    version: CODEX_VERSION,
    status: 'signedOut',
    email: null,
    plan: null,
    sessionId: null,
    problem: null,
  };
  private models: CodexModel[] = [];
  private loginId: string | null = null;
  private queue: Promise<unknown> = Promise.resolve();
  private session: Session | undefined;
  private readonly file: AtomicStateFile<Session>;
  private readonly unsubscribe: (() => void)[];
  constructor(
    private readonly rpc: RpcTransport,
    sessionPath: string,
    private readonly openBrowser: (url: string) => Promise<void>,
    private readonly changed: () => void,
  ) {
    this.file = new AtomicStateFile(sessionPath, (raw) => {
      const value = object(raw);
      if (value.schemaVersion !== 1 || Object.keys(value).length !== 2) throw malformed();
      return { schemaVersion: 1, sessionId: value.sessionId === null ? null : id(value.sessionId) };
    });
    this.unsubscribe = [
      rpc.onDisconnected(() => {
        this.loginId = null;
        this.state = {
          ...this.state,
          runtime: 'disconnected',
          status: this.state.status === 'signingIn' ? 'signedOut' : this.state.status,
          problem: 'Codex is disconnected. Reconnect to check pending messages.',
        };
        this.models = [];
        this.changed();
      }),
      rpc.onNotification((event) => {
        if (event.method !== 'account/login/completed' && event.method !== 'account/updated')
          return;
        this.schedule(async () => {
          if (event.method === 'account/login/completed') {
            const params = object(event.params);
            if (params.loginId !== this.loginId) return;
            this.loginId = null;
          }
          await this.refresh();
        }).catch(() => this.problem());
      }),
    ];
  }
  snapshot(): { account: AccountStatus; models: CodexModel[] } {
    return structuredClone({ account: this.state, models: this.models });
  }
  private schedule<T>(work: () => Promise<T>): Promise<T> {
    const result = this.queue.then(work);
    this.queue = result.catch(() => {});
    return result;
  }
  private problem(): void {
    this.state = {
      ...this.state,
      problem: 'Account status could not be checked. Reconnect and try again.',
    };
    this.changed();
  }
  async action(action: 'connect' | 'signIn' | 'cancelSignIn' | 'signOut'): Promise<void> {
    return this.schedule(async () => {
      try {
        this.session ??= (await this.file.read()) ?? { schemaVersion: 1, sessionId: null };
        if (this.state.runtime !== 'ready') {
          this.state = { ...this.state, runtime: 'connecting', problem: null };
          this.changed();
          await this.rpc.start();
          this.state = { ...this.state, runtime: 'ready' };
          await this.refresh();
        }
        if (action === 'connect') {
          await this.refresh();
          return;
        }
        if (action === 'signIn') {
          if (this.loginId || this.state.status === 'signedIn') return;
          // Commit the account boundary before starting a new browser ceremony.
          this.session = { schemaVersion: 1, sessionId: randomUUID() };
          await this.file.write(this.session);
          const value = object(await this.rpc.request('account/login/start', { type: 'chatgpt' }));
          if (value.type !== 'chatgpt') throw malformed();
          this.loginId = id(value.loginId);
          this.state = {
            ...this.state,
            status: 'signingIn',
            sessionId: this.session.sessionId,
            problem: null,
          };
          this.changed();
          try {
            await this.openBrowser(loginURL(string(value.authUrl, 16000)));
          } catch {
            await this.rpc.request('account/login/cancel', { loginId: this.loginId });
            this.loginId = null;
            throw new AppProblem(
              'runtime_unavailable',
              'The sign-in browser could not open. Try signing in again.',
            );
          }
        } else if (action === 'cancelSignIn') {
          if (this.loginId)
            await this.rpc.request('account/login/cancel', { loginId: this.loginId });
          this.loginId = null;
          await this.refresh();
        } else {
          this.session = { schemaVersion: 1, sessionId: null };
          await this.file.write(this.session);
          this.state = {
            ...this.state,
            status: 'signedOut',
            sessionId: null,
            email: null,
            plan: null,
          };
          this.models = [];
          this.changed();
          await this.rpc.request('account/logout');
          this.loginId = null;
          await this.refresh();
        }
      } catch (error) {
        if (this.state.runtime === 'connecting') this.state.runtime = 'disconnected';
        this.state.problem =
          error instanceof AppProblem
            ? error.message
            : 'Account connection could not be completed.';
        this.changed();
        throw error;
      }
    });
  }
  private async refresh(): Promise<void> {
    const response = object(await this.rpc.request('account/read', { refreshToken: false }));
    const account = response.account === null ? null : object(response.account);
    if (response.requiresOpenaiAuth !== true) throw malformed();
    if (account && account.type !== 'chatgpt')
      throw new AppProblem(
        'unsupported',
        'Sign in with ChatGPT to use Gobble agents. Other billing methods are not enabled.',
      );
    if (account && !this.session?.sessionId) {
      this.session = { schemaVersion: 1, sessionId: randomUUID() };
      await this.file.write(this.session);
    }
    const nextState: AccountStatus = {
      ...this.state,
      runtime: 'ready',
      status: account ? 'signedIn' : this.loginId ? 'signingIn' : 'signedOut',
      email: account?.email == null ? null : string(account.email, 320),
      plan: account ? string(account.planType, 80) : null,
      sessionId: account ? (this.session?.sessionId ?? null) : null,
      problem: null,
    };
    const nextModels: CodexModel[] = [];
    if (account) {
      let cursor: string | null = null;
      const seen = new Set<string>();
      do {
        const page = object(
          await this.rpc.request('model/list', { cursor, limit: 50, includeHidden: false }),
        );
        for (const raw of array(page.data, 100)) {
          const model = object(raw);
          if (model.hidden === true) continue;
          nextModels.push(
            parse(CodexModelSchema, {
              id: id(model.model),
              name: string(model.displayName, 200),
              isDefault: model.isDefault,
              inputModalities:
                model.inputModalities === undefined ? [] : array(model.inputModalities, 2),
              efforts: array(model.supportedReasoningEfforts, 16).map((effort) =>
                string(object(effort).reasoningEffort, 32),
              ),
              defaultEffort: string(model.defaultReasoningEffort, 32),
            }),
          );
        }
        cursor = page.nextCursor === null ? null : id(page.nextCursor);
        if (cursor && seen.has(cursor)) throw malformed();
        if (cursor) seen.add(cursor);
        if (nextModels.length > 100 || seen.size > 10) throw malformed();
      } while (cursor);
    }
    this.state = nextState;
    this.models = nextModels;
    this.changed();
  }
  require(model?: string, effort?: string): string {
    if (
      this.state.runtime !== 'ready' ||
      this.state.problem !== null ||
      this.state.status !== 'signedIn' ||
      !this.state.sessionId
    )
      throw new AppProblem('runtime_unavailable', 'Connect your ChatGPT account before sending.');
    if (
      model &&
      !this.models.some((item) => item.id === model && (!effort || item.efforts.includes(effort)))
    )
      throw new AppProblem(
        'invalid_request',
        'Choose a model and reasoning effort currently available to your account.',
      );
    return this.state.sessionId;
  }
  async stop(): Promise<void> {
    for (const unsubscribe of this.unsubscribe) unsubscribe();
    await this.queue;
    if (this.loginId)
      await this.rpc.request('account/login/cancel', { loginId: this.loginId }).catch(() => {});
  }
}
