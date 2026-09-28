import { randomUUID } from 'node:crypto';
import {
  EVIDENCE_TEXT_BYTES,
  evidenceTextBytes,
  type AgentAttachment,
  type PrepareEvidence,
  type PreparedEvidence,
  type PreviewEvidence,
  type SendMessage,
} from '@gobble/contracts';
import { AppProblem } from '../problem';
import { capturedManifest } from './capture';
import type { WorkspaceResources } from '../workspace/service';
import { checkSelection } from '../workspace/selection';
import type { ImageRenderer } from '../shared-context/observation';
import {
  evidenceInput,
  evidencePreview,
  materializeEvidence,
  type EvidenceAsset,
} from './materialize';
import type { EvidenceStorage } from './storage';
import type {
  AddressedEvidence,
  EvidenceAccount,
  EvidenceDelivery,
  EvidenceDraft,
  EvidenceWorkspace,
} from './ports';

type Prepared = {
  receipt: PreparedEvidence;
  draft: EvidenceDraft;
  sessionId: string;
  assets: EvidenceAsset[];
};
const expired = () =>
  new AppProblem(
    'stale_revision',
    'The attachments need to be prepared again. Review the current recipient and preview.',
  );
const binding = (draft: EvidenceDraft) =>
  JSON.stringify({
    rendererSessionId: draft.rendererSessionId,
    revision: draft.attachmentRevision,
    attachments: draft.attachments,
    agentId: draft.agent?.agentId,
    configuration: draft.agent?.configuration,
    access: draft.agent?.access,
  });

/** Session-bounded preparation; durable asset ownership is separate from the Project aggregate. */
export class EvidenceService implements AddressedEvidence {
  private prepared = new Map<string, Prepared>();
  private sequences = new Map<string, number>();
  private pending = 0;
  private stopped = false;
  constructor(
    private readonly workspace: EvidenceWorkspace,
    private readonly resources: WorkspaceResources,
    private readonly storage: EvidenceStorage,
    private readonly renderImage: ImageRenderer,
    private readonly account: EvidenceAccount,
    private readonly now = Date.now,
  ) {}
  private checkOpen(): void {
    if (this.stopped)
      throw new AppProblem(
        'runtime_unavailable',
        'Attachment preparation has stopped. Reopen the app to continue.',
      );
  }
  private prune(): void {
    for (const [id, value] of this.prepared)
      if (value.receipt.expiresAt <= this.now()) this.prepared.delete(id);
  }
  private session(draft: EvidenceDraft): string {
    const config = draft.agent?.configuration;
    if (!config)
      throw new AppProblem(
        'invalid_request',
        'Choose a configured recipient before preparing attachments.',
      );
    if (
      draft.attachments.some((item) => item.evidence.schemaVersion === 8) &&
      draft.agent?.access !== 'sharedViews'
    )
      throw new AppProblem(
        'unsupported',
        'Report attachments require Shared views access so the agent can read original charts.',
      );
    return this.account.require(config.model, config.effort);
  }
  private requireImages(assets: EvidenceAsset[], model: string): void {
    const images = assets.filter((item) => item.manifest.representation.kind === 'image');
    if (images.length > 2)
      throw new AppProblem('unsupported', 'A message can include at most two images.');
    if (
      (images.length ||
        assets.some(
          (item) =>
            item.manifest.representation.kind === 'report' &&
            item.manifest.representation.imageCount > 0,
        )) &&
      !this.account.supportsImages(model)
    )
      throw new AppProblem(
        'unsupported',
        'This model cannot receive images. Choose a supported model.',
      );
  }
  async prepare(input: PrepareEvidence): Promise<PreparedEvidence> {
    this.checkOpen();
    this.prune();
    if (this.pending >= 2)
      throw new AppProblem(
        'runtime_unavailable',
        'Other attachments are still being prepared. Try again shortly.',
      );
    this.pending++;
    try {
      const draft = await this.workspace.readEvidenceDraft(input.projectId);
      if (
        !draft.attachments.length ||
        draft.agent?.agentId !== input.agentId ||
        draft.attachmentRevision !== input.attachmentRevision
      )
        throw expired();
      const sequence = (this.sequences.get(input.projectId) ?? 0) + 1;
      this.sequences.set(input.projectId, sequence);
      const sessionId = this.session(draft);
      const assets: EvidenceAsset[] = [];
      for (const attachment of draft.attachments) {
        this.checkOpen();
        if (attachment.capture) {
          const asset = await this.storage.read(input.projectId, capturedManifest(attachment));
          evidencePreview(asset);
          assets.push(asset);
        } else {
          const data = await this.resources.read(input.projectId, attachment.evidence.resource);
          if (
            data.kind === 'file' &&
            data.value.content.kind === 'image' &&
            !this.account.supportsImages(draft.agent.configuration!.model)
          )
            throw new AppProblem(
              'unsupported',
              'This model cannot receive images. Choose a supported model.',
            );
          assets.push(await materializeEvidence(attachment, data, this.renderImage));
        }
        this.requireImages(assets, draft.agent.configuration!.model);
        if (
          assets.reduce((sum, item) => sum + evidenceTextBytes(item.manifest), 0) >
          EVIDENCE_TEXT_BYTES
        )
          throw new AppProblem(
            'unsupported',
            'The combined text attachments exceed 64 KiB. Remove an attachment or select less content.',
          );
      }
      await this.storage.checkCapacity(input.projectId, assets);
      const latest = await this.workspace.readEvidenceDraft(input.projectId);
      this.checkOpen();
      if (
        this.sequences.get(input.projectId) !== sequence ||
        binding(latest) !== binding(draft) ||
        this.session(latest) !== sessionId
      )
        throw expired();
      for (const [id, previous] of this.prepared)
        if (previous.receipt.projectId === input.projectId) this.prepared.delete(id);
      if (this.prepared.size >= 4)
        throw new AppProblem(
          'runtime_unavailable',
          'Too many attachment previews are prepared. Reopen the app or wait for previews to expire.',
        );
      const receipt: PreparedEvidence = {
        ...input,
        preparedId: 'prep_' + randomUUID(),
        model: draft.agent.configuration!.model,
        expiresAt: this.now() + 10 * 60 * 1000,
        items: assets.map((item) => item.manifest),
      };
      this.prepared.set(receipt.preparedId, { receipt, draft, sessionId, assets });
      return structuredClone(receipt);
    } finally {
      this.pending--;
    }
  }
  private async current(projectId: string, preparedId: string): Promise<Prepared> {
    this.checkOpen();
    this.prune();
    const value = this.prepared.get(preparedId);
    if (!value || value.receipt.projectId !== projectId) throw expired();
    const latest = await this.workspace.readEvidenceDraft(projectId);
    this.checkOpen();
    if (
      this.prepared.get(preparedId) !== value ||
      value.receipt.expiresAt <= this.now() ||
      binding(latest) !== binding(value.draft) ||
      this.session(latest) !== value.sessionId
    )
      throw expired();
    this.requireImages(value.assets, value.receipt.model);
    return value;
  }
  async preview(input: PreviewEvidence) {
    this.checkOpen();
    if (input.kind === 'draft') {
      const draft = await this.workspace.readEvidenceDraft(input.projectId);
      const attachment = draft.attachments.find((item) => item.attachmentId === input.attachmentId);
      if (!attachment?.capture)
        throw new AppProblem('not_found', 'This draft has no captured attachment.');
      const manifest = capturedManifest(attachment);
      const asset = await this.storage.read(input.projectId, manifest);
      const current = await this.workspace.readEvidenceDraft(input.projectId);
      if (
        current.rendererSessionId !== draft.rendererSessionId ||
        !current.attachments.some(
          (item) =>
            item.attachmentId === attachment.attachmentId &&
            item.capture?.asset.hash === manifest.asset.hash,
        )
      )
        throw new AppProblem('stale_revision', 'This draft attachment changed.');
      this.checkOpen();
      return evidencePreview(asset);
    }
    if (input.kind === 'question') {
      if (!this.workspace.readQuestionEvidence)
        throw new AppProblem('unsupported', 'Question evidence is unavailable.');
      const read = () =>
        this.workspace.readQuestionEvidence!(input.projectId, input.questionId, input.attachmentId);
      const asset = await this.storage.read(input.projectId, await read());
      await read();
      return evidencePreview(asset);
    }
    if (input.kind === 'sent') {
      const manifest = await this.workspace.readSentEvidence(
        input.projectId,
        input.requestId,
        input.attachmentId,
      );
      const asset = await this.storage.read(input.projectId, manifest);
      // Historical content is authorized through its addressed message, never an arbitrary hash/path.
      await this.workspace.readSentEvidence(input.projectId, input.requestId, input.attachmentId);
      return evidencePreview(asset);
    }
    const value = await this.current(input.projectId, input.preparedId);
    const asset = value.assets.find((item) => item.manifest.attachmentId === input.attachmentId);
    if (!asset) throw new AppProblem('not_found', 'This preparation has no such attachment.');
    return evidencePreview(asset);
  }
  async accept(
    input: SendMessage,
    agent: AgentAttachment,
    sessionId: string,
    assertCurrent: () => void,
  ): Promise<EvidenceDelivery> {
    if (!input.preparedEvidenceId) throw expired();
    const value = await this.current(input.projectId, input.preparedEvidenceId);
    if (
      value.receipt.agentId !== agent.agentId ||
      value.sessionId !== sessionId ||
      JSON.stringify(value.draft.agent?.configuration) !== JSON.stringify(agent.configuration)
    )
      throw expired();
    const assert = () => {
      this.checkOpen();
      assertCurrent();
      this.workspace.assertEvidenceSession(input.projectId, value.draft.rendererSessionId);
      if (
        value.receipt.expiresAt <= this.now() ||
        this.prepared.get(value.receipt.preparedId) !== value ||
        this.session(value.draft) !== sessionId
      )
        throw expired();
      this.requireImages(value.assets, value.receipt.model);
    };
    assert();
    const content = evidenceInput(value.assets);
    if (
      Buffer.byteLength(
        JSON.stringify({ input: [{ type: 'text', text: input.text }, ...content] }),
      ) +
        65536 >
      4 * 1024 * 1024
    )
      throw new AppProblem(
        'unsupported',
        'This message exceeds the delivery limit. Remove an attachment or shorten its text.',
      );
    await this.storage.put(input.projectId, value.assets);
    // Freshness is a point-in-time read just before acceptance, outside the Project write queue.
    for (const item of value.assets) {
      assert();
      if (item.manifest.capture) {
        evidencePreview(item);
        continue;
      }
      checkSelection(
        item.manifest.evidence,
        await this.resources.read(input.projectId, item.manifest.evidence.resource),
      );
    }
    await this.current(input.projectId, value.receipt.preparedId);
    assert();
    return {
      manifests: structuredClone(value.receipt.items),
      input: content,
      consumption: {
        text: input.text,
        agentId: agent.agentId,
        attachmentRevision: value.draft.attachmentRevision,
        attachmentIds: value.draft.attachments.map((item) => item.attachmentId),
        assertCurrent: assert,
      },
    };
  }
  consume(preparedId: string): void {
    this.prepared.delete(preparedId);
  }
  stop(): void {
    this.stopped = true;
    this.prepared.clear();
  }
}
