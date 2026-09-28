import type { NotebookCapture } from '../evidence/notebook-evidence';
import type { PdfRasterCapture } from '../evidence/pdf-evidence';
import { materializeObserved } from '../evidence/capture';
import { createHash, randomUUID } from 'node:crypto';
import {
  parse,
  QuestionInputSchema,
  type AgentAttachment,
  type Question,
  type SendMessage,
  type SurfaceData,
  type EvidenceRef,
} from '@gobble/contracts';
import { AppProblem } from '../problem';
import type { ToolCall, ToolContext } from '../shared-context/ports';
import type { ImageRenderer } from '../shared-context/observation';
import { evidenceInput, materializeEvidence, type EvidenceAsset } from '../evidence/materialize';
import type { EvidenceStorage } from '../evidence/storage';
import type { EvidenceAccount } from '../evidence/ports';
import type { WorkspaceResources } from '../workspace/service';
import { checkSelection } from '../workspace/selection';
import { requirePending, requireQuestion } from './model';
import type { QuestionReplies, QuestionWorkspace, QuestionTools } from './ports';

const hash = (value: unknown) => createHash('sha256').update(JSON.stringify(value)).digest('hex');
const turnKey = (projectId: string, agentId: string, submissionId: string) =>
  projectId + '/' + agentId + '/' + submissionId;
const contextKey = (context: ToolContext) =>
  turnKey(context.agent.projectId, context.agent.agentId, context.submission.requestId);
function bounds(assets: EvidenceAsset[]): void {
  if (assets.some((item) => item.manifest.evidence.schemaVersion === 8))
    throw new AppProblem(
      'unsupported',
      'Discuss reports in Chat and attach the report directly to the current message. Report-linked questions are not supported.',
    );
  if (
    assets.length > 16 ||
    assets.filter((item) => item.manifest.representation.kind === 'image').length > 2 ||
    assets
      .filter((item) => item.manifest.asset.mediaType === 'application/json')
      .reduce((sum, item) => sum + item.bytes.length, 0) > 65536
  )
    throw new AppProblem(
      'unsupported',
      'Question evidence is limited to 16 items, two images and 64 KiB of text.',
    );
}

/** Owns question rules and turn-scoped observation authority; never writes a second database. */
export class QuestionService implements QuestionReplies, QuestionTools {
  private readonly observations = new Map<string, Map<string, EvidenceAsset>>();
  constructor(
    private readonly workspace: QuestionWorkspace,
    private readonly resources: WorkspaceResources,
    private readonly storage: EvidenceStorage,
    private readonly account: Pick<EvidenceAccount, 'supportsImages'>,
    private readonly pdfCapture?: PdfRasterCapture,
    private readonly notebookCapture?: NotebookCapture,
  ) {}
  begin(projectId: string, agentId: string, submissionId: string): void {
    this.observations.set(turnKey(projectId, agentId, submissionId), new Map());
  }
  release(projectId: string, agentId: string, submissionId: string): void {
    this.observations.delete(turnKey(projectId, agentId, submissionId));
  }
  async observe(
    context: ToolContext,
    evidence: EvidenceRef,
    data: SurfaceData,
    renderImage: ImageRenderer,
    assertCurrent: () => void,
    presentation?: import('@gobble/contracts').ReferencePresentation,
  ): Promise<string> {
    if (evidence.projectId !== context.agent.projectId)
      throw new AppProblem('forbidden', 'Observation evidence belongs to another Project.');
    const observations = this.observations.get(contextKey(context));
    if (!observations || observations.size >= 16)
      throw new AppProblem('unsupported', 'This turn has reached its question evidence limit.');
    const attachment = {
      attachmentId: 'att_' + randomUUID(),
      evidence,
      ...(presentation ? { presentation } : {}),
      createdAt: Date.now(),
      label:
        data.kind === 'file'
          ? data.value.name
          : data.kind === 'run'
            ? 'Run preview'
            : 'Attempt log',
    };
    if (evidence.schemaVersion === 5 && !this.pdfCapture)
      throw new AppProblem('unsupported', 'PDF question capture is unavailable.');
    if (evidence.schemaVersion === 6 && !this.notebookCapture)
      throw new AppProblem('unsupported', 'Notebook question capture is unavailable.');
    const asset =
      evidence.schemaVersion === 6
        ? await this.notebookCapture!(attachment, data)
        : evidence.schemaVersion === 5
          ? await this.pdfCapture!(attachment, data)
          : data.kind !== 'file' && evidence.schemaVersion >= 3
            ? materializeObserved(attachment, data)
            : await materializeEvidence(attachment, data, renderImage);
    assertCurrent();
    if (this.observations.get(contextKey(context)) !== observations)
      throw new AppProblem('forbidden', 'The observation session ended.');
    if (
      [...observations.values()].reduce(
        (sum, item) => sum + item.bytes.length,
        asset.bytes.length,
      ) >
      2 * 1024 * 1024
    )
      throw new AppProblem(
        'unsupported',
        'This turn has reached its question evidence memory limit.',
      );
    if (observations.size >= 16)
      throw new AppProblem('unsupported', 'This turn has reached its question evidence limit.');
    const id = 'obs_' + randomUUID();
    observations.set(id, asset);
    return id;
  }
  async create(
    context: ToolContext,
    call: ToolCall,
    guard: () => void,
  ): Promise<{ questionId: string; state: Question['state']['kind'] }> {
    const input = parse(QuestionInputSchema, call.arguments);
    if (!input.question.trim() || input.options?.some((item) => !item.trim()))
      throw new AppProblem('invalid_request', 'Question and option text must be nonempty.');
    const identity = hash([
      context.agent.projectId,
      context.agent.agentId,
      context.agent.provider.accountSessionId,
      context.agent.provider.toolsetVersion,
      call.threadId,
      call.turnId,
      call.callId,
    ]);
    const digest = hash([input.question, input.options ?? [], input.evidenceIds]);
    const projectId = context.agent.projectId;
    const existing = await this.workspace.readQuestions(projectId);
    guard();
    const prior = existing.questions.find((item) => item.invocation.identity === identity);
    if (prior) {
      if (prior.invocation.digest !== digest)
        throw new AppProblem(
          'request_conflict',
          'This question call was used with different arguments.',
        );
      return { questionId: prior.decisionId, state: prior.state.kind };
    }
    if (existing.questions.length >= 100)
      throw new AppProblem('unsupported', 'This Project has reached its limit of 100 questions.');
    const assets: EvidenceAsset[] = [];
    for (const id of input.evidenceIds) {
      guard();
      const observed = this.observations.get(contextKey(context))?.get(id);
      if (observed) {
        assets.push(observed);
        continue;
      }
      // Accepted-but-unconfirmed history does not prove evidence was received. The active
      // callback proves this turn's input was received; older evidence requires completion.
      const manifest = existing.submissions
        .filter(
          (item) =>
            item.agentId === context.agent.agentId &&
            item.threadId === call.threadId &&
            (item.requestId === context.submission.requestId || item.state === 'completed'),
        )
        .flatMap((item) => [
          ...(item.evidence ?? []),
          ...(existing.questions.find(
            (question) =>
              question.decisionId === item.replyToQuestionId &&
              question.requestedBy === context.agent.agentId,
          )?.evidence ?? []),
        ])
        .find((item) => item.attachmentId === id);
      if (!manifest)
        throw new AppProblem(
          'forbidden',
          'This evidence ID was not observed or received by this Agent in this conversation.',
        );
      assets.push(await this.storage.read(projectId, manifest));
    }
    bounds(assets);
    await this.storage.put(projectId, assets);
    await this.checkCurrent(projectId, assets, guard);
    guard();
    const question: Question = {
      kind: 'question',
      projectId,
      decisionId: 'dec_' + identity.slice(0, 48),
      requestedBy: context.agent.agentId,
      question: input.question,
      options: input.options ?? [],
      evidence: assets.map((item) => structuredClone(item.manifest)),
      originSubmissionId: context.submission.requestId,
      createdAt: Date.now(),
      invocation: { identity, digest },
      state: { kind: 'pending' },
    };
    await this.workspace.changeQuestions(projectId, guard, (questions) => {
      const duplicate = questions.find((item) => item.invocation.identity === identity);
      if (duplicate) {
        if (duplicate.invocation.digest !== digest)
          throw new AppProblem(
            'request_conflict',
            'This question call was used with different arguments.',
          );
        return questions;
      }
      if (questions.length >= 100)
        throw new AppProblem('unsupported', 'This Project has reached its question limit.');
      return [...questions, question];
    });
    return { questionId: question.decisionId, state: 'pending' };
  }
  private async checkCurrent(
    projectId: string,
    assets: EvidenceAsset[],
    guard: () => void,
  ): Promise<void> {
    for (const item of assets) {
      guard();
      if (item.manifest.evidence.schemaVersion === 5) {
        if (!this.resources.checkPdfSource)
          throw new AppProblem('unsupported', 'PDF source verification is unavailable.');
        await this.resources.checkPdfSource(item.manifest.evidence);
      } else if (item.manifest.evidence.schemaVersion === 6) {
        if (!this.resources.checkNotebookSource)
          throw new AppProblem('unsupported', 'Notebook source verification is unavailable.');
        await this.resources.checkNotebookSource(item.manifest.evidence);
      } else
        checkSelection(
          item.manifest.evidence,
          await this.resources.read(projectId, item.manifest.evidence.resource),
        );
      guard();
    }
  }
  async prepareReply(input: SendMessage, agent: AgentAttachment, guard: () => void) {
    const snapshot = await this.workspace.readQuestions(input.projectId);
    const question = requireQuestion(snapshot.questions, input.replyToQuestionId ?? '');
    requirePending(question);
    if (question.requestedBy !== input.agentId || agent.agentId !== input.agentId)
      throw new AppProblem('forbidden', 'Reply to the Agent that asked this question.');
    const draft = await this.workspace.readEvidenceDraft(input.projectId);
    const assertCurrent = () => {
      guard();
      this.workspace.assertEvidenceSession(input.projectId, draft.rendererSessionId);
    };
    assertCurrent();
    if (
      question.evidence.some((item) => item.representation.kind === 'image') &&
      !this.account.supportsImages(agent.configuration!.model)
    )
      throw new AppProblem(
        'unsupported',
        'This model cannot receive the question’s images. Choose a supported model.',
      );
    const assets = await Promise.all(
      question.evidence.map((item) => this.storage.read(input.projectId, item)),
    );
    try {
      await this.checkCurrent(input.projectId, assets, assertCurrent);
    } catch (error) {
      assertCurrent();
      if (error instanceof AppProblem && error.code === 'stale_revision') {
        await this.workspace.changeQuestions(input.projectId, assertCurrent, (questions) => {
          const current = requireQuestion(questions, question.decisionId);
          if (current.state.kind === 'pending')
            current.state = {
              kind: 'invalidated',
              reason: 'evidence_changed',
              invalidatedAt: Date.now(),
            };
          return questions;
        });
        throw new AppProblem(
          'stale_revision',
          'The question’s evidence changed. Your draft is preserved; cancel the reply and ask for a new question.',
        );
      }
      throw error;
    }
    return {
      evidence: question.evidence,
      commit: {
        questionId: question.decisionId,
        requestId: input.requestId,
        agentId: input.agentId,
        text: input.text,
        attachmentRevision: draft.attachmentRevision,
        assertCurrent,
      },
      input: [
        {
          type: 'text' as const,
          text:
            'Explicit user reply to this question:\n' +
            JSON.stringify({
              questionId: question.decisionId,
              question: question.question,
              options: question.options,
            }),
        },
        ...evidenceInput(assets, 'Question evidence'),
      ],
    };
  }
}
