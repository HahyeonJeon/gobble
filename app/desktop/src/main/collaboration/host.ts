import type { PipelineReviewHost } from '../pipeline-review/host';
import type { QuestionTools, QuestionReplies } from '../questions/ports';
import { join } from 'node:path';
import type { AccountAction, CollaborationStatus } from '@gobble/contracts';
import { CodexAccount } from '../codex/account';
import { CodexConversations } from '../codex/conversations';
import { CodexTransport } from '../codex/transport';
import { AppProblem } from '../problem';
import { CollaborationCoordinator } from './coordinator';
import type { SharedToolExecutor } from '../shared-context/ports';
import type { CollaborationStore } from './store';
import type { AddressedEvidence, EvidenceAccount } from '../evidence/ports';

export class CollaborationHost {
  readonly coordinator: CollaborationCoordinator;
  readonly evidence: AddressedEvidence | undefined;
  private readonly account: CodexAccount;
  private readonly rpc: CodexTransport;
  private readonly listeners = new Set<(value: CollaborationStatus) => void>();
  private sequence = 0;
  private problem: string | null = null;
  private accountBusy = false;
  private stopped = false;
  private notificationTimer: NodeJS.Timeout | undefined;
  constructor(
    store: CollaborationStore,
    executable: string,
    profile: string,
    openBrowser: (url: string) => Promise<void>,
    createSharedTools?: (
      supportsImages: (model: string) => boolean,
      questions?: QuestionTools,
    ) => SharedToolExecutor,
    createEvidence?: (account: EvidenceAccount) => AddressedEvidence,
    createQuestions?: (account: EvidenceAccount) => QuestionTools & QuestionReplies,
    pipelineReview?: PipelineReviewHost,
  ) {
    const directory = join(profile, 'codex');
    this.rpc = new CodexTransport(
      executable,
      join(directory, 'home'),
      join(directory, 'discussion'),
    );
    this.account = new CodexAccount(this.rpc, join(directory, 'session.json'), openBrowser, () =>
      this.publish(),
    );
    const account: EvidenceAccount = {
      require: (model, effort) => {
        if (this.accountBusy)
          throw new AppProblem('runtime_unavailable', 'Wait for the account action to finish.');
        return this.account.require(model, effort);
      },
      supportsImages: (model) =>
        this.account
          .snapshot()
          .models.some((item) => item.id === model && item.inputModalities?.includes('image')),
    };
    this.evidence = createEvidence?.(account);
    const questions = createQuestions?.(account);
    this.coordinator = new CollaborationCoordinator(
      store,
      new CodexConversations(this.rpc, join(directory, 'discussion')),
      account,
      () => this.publish(),
      (message) => {
        this.problem = message;
        this.publish();
      },
      createSharedTools?.(account.supportsImages, questions),
      this.evidence,
      questions,
      pipelineReview,
    );
  }
  status(): CollaborationStatus {
    return {
      sequence: this.sequence,
      ...this.account.snapshot(),
      streams: this.coordinator.streams(),
      problem: this.problem,
    };
  }
  onChanged(listener: (value: CollaborationStatus) => void): () => void {
    this.listeners.add(listener);
    return () => this.listeners.delete(listener);
  }
  private publish(): void {
    this.sequence += 1;
    if (this.notificationTimer) return;
    this.notificationTimer = setTimeout(() => {
      this.notificationTimer = undefined;
      for (const listener of this.listeners) {
        try {
          listener(this.status());
        } catch {
          /* Window closure preserves host state. */
        }
      }
    }, 40);
  }
  async accountAction(input: AccountAction): Promise<CollaborationStatus> {
    if (this.stopped)
      throw new AppProblem(
        'runtime_unavailable',
        'Agents have stopped. Restart the app to reconnect.',
      );
    if (this.accountBusy)
      throw new AppProblem('request_conflict', 'An account action is already in progress.');
    if (this.coordinator.hasActive() && input.action !== 'connect')
      throw new AppProblem('request_conflict', 'Stop active agents before changing the account.');
    this.accountBusy = true;
    try {
      await this.account.action(input.action);
      return this.status();
    } finally {
      this.accountBusy = false;
    }
  }
  async stop(): Promise<void> {
    this.stopped = true;
    this.evidence?.stop();
    try {
      await this.coordinator.stop();
      await this.account.stop();
    } finally {
      await this.rpc.stop();
      clearTimeout(this.notificationTimer);
      this.listeners.clear();
    }
  }
}
