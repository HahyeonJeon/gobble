import {
  parse,
  RunFollowUpInputSchema,
  type EvidenceManifest,
  type RunFollowUpInput,
} from '@gobble/contracts';
import type { ToolContext } from '../shared-context/ports';
import type { ProjectService } from '../service/project-service';
import type { WorkspaceController } from '../workspace/controller';
import { AppProblem } from '../problem';

/** Derive provenance from this sent message, never from model-supplied arguments. */
export async function buildRunFollowUp(
  context: ToolContext,
  service: ProjectService,
  workspace: Pick<WorkspaceController, 'readSentEvidence'>,
  verify: ((projectId: string, manifest: EvidenceManifest) => Promise<void>) | undefined,
  guard: () => void,
): Promise<RunFollowUpInput | undefined> {
  const projectId = context.agent.projectId;
  const items = (context.submission.evidence ?? []).filter(
    (m) => m.capture?.kind === 'observed' && ['run', 'log'].includes(m.evidence.resource.kind),
  );
  if (!items.length) return undefined;
  const refs = new Set(
    items.map((m) => ('runRef' in m.evidence.resource ? m.evidence.resource.runRef : '')),
  );
  if (refs.size !== 1)
    throw new AppProblem(
      'invalid_request',
      'Use evidence from one analysis when requesting a linked Pipeline change.',
    );
  const reviews = await service.launches.list({ projectId });
  guard();
  const origin = reviews.find((r) => r.operation?.runRef === [...refs][0]);
  if (!origin || origin.pipelineId !== context.submission.pipelineReview?.pipelineId)
    throw new AppProblem(
      'invalid_request',
      'Choose the Current design belonging to this analysis before requesting a change.',
    );
  if (!verify)
    throw new AppProblem('unsupported', 'Captured evidence verification is unavailable.');
  const evidence: EvidenceManifest[] = [];
  for (const item of items) {
    const saved = await workspace.readSentEvidence(
      projectId,
      context.submission.requestId,
      item.attachmentId,
    );
    guard();
    if (JSON.stringify(saved) !== JSON.stringify(item))
      throw new AppProblem(
        'stale_revision',
        'The sent evidence does not match this proposal context.',
      );
    await verify(projectId, saved);
    guard();
    evidence.push(saved);
  }
  return parse(RunFollowUpInputSchema, {
    launchReviewId: origin.requestId,
    submissionId: context.submission.requestId,
    evidence,
  });
}
