import { reportRepresentation } from '../evidence/report';
import { isDeepStrictEqual } from 'node:util';
import { reportReading, validateManifest } from '@gobble/contracts';
import type { WorkspaceController } from '../workspace/controller';
import type { WorkspaceResources } from '../workspace/service';
import { AppProblem } from '../problem';
import { textResult, type ToolContext, type ToolResult } from './ports';

/** Addressed evidence reads require this active submission, never a guessed saved hash. */
export async function readReport(
  context: ToolContext,
  input: { attachmentId: string; imageId?: string },
  workspace: Pick<WorkspaceController, 'readSentEvidence' | 'sentEvidenceGuard'>,
  resources: Pick<WorkspaceResources, 'read'>,
  supportsImages: (model: string) => boolean,
  guard: () => void,
): Promise<ToolResult> {
  const { agent, submission } = context;
  const assertProject = workspace.sentEvidenceGuard(agent.projectId);
  const manifest = submission.evidence?.find((item) => item.attachmentId === input.attachmentId);
  const assert = () => {
    guard();
    assertProject();
    if (
      agent.access !== 'sharedViews' ||
      submission.agentId !== agent.agentId ||
      !manifest ||
      manifest.evidence.schemaVersion !== 8 ||
      manifest.evidence.projectId !== agent.projectId
    )
      throw new AppProblem(
        'forbidden',
        'Attach this report to the current message for this agent.',
      );
    validateManifest(manifest);
    if (
      manifest.representation.kind !== 'report' ||
      (manifest.representation.imageCount > 0 && !supportsImages(submission.model))
    )
      throw new AppProblem('unsupported', 'This report requires an image-capable model.');
  };
  assert();
  const verify = async () => {
    const saved = await workspace.readSentEvidence(
      agent.projectId,
      submission.requestId,
      input.attachmentId,
    );
    assert();
    if (!isDeepStrictEqual(saved, manifest))
      throw new AppProblem('stale_revision', 'The attached report no longer matches this message.');
  };
  await verify();
  const data = await resources.read(agent.projectId, manifest!.evidence.resource);
  assert();
  if (
    data.kind !== 'report' ||
    manifest!.evidence.schemaVersion !== 8 ||
    !isDeepStrictEqual(data.saved, manifest!.evidence.resource.saved) ||
    !isDeepStrictEqual(reportRepresentation(data.value), manifest!.representation)
  )
    throw new AppProblem('internal', 'The saved report identity does not match this attachment.');
  await verify();
  const identity = {
    attachmentId: input.attachmentId,
    savedHash: data.saved.asset.hash,
    producer: data.saved.producer,
    pointable: false,
  };
  if (!input.imageId)
    return {
      ...textResult({
        ...identity,
        report: reportReading(data.value),
        coverage: {
          imagesReturned: [],
          meaning: 'Text, tables and chart inventory only; use imageId to read original pixels.',
        },
      }),
      assertCurrent: assert,
    };
  for (const module of data.value.content.modules) {
    const image = module.blocks.find(
      (block) => block.kind === 'image' && block.id === input.imageId,
    );
    if (image?.kind !== 'image') continue;
    return {
      success: true,
      assertCurrent: assert,
      content: [
        {
          type: 'text',
          text: JSON.stringify({
            ...identity,
            module: { id: module.id, title: module.title, status: module.status },
            image: { id: image.id, alt: image.alt, width: image.width, height: image.height },
            coverage: {
              imagesReturned: [image.id],
              meaning: 'Only this original chart is returned by this call.',
            },
          }),
        },
        { type: 'image', url: 'data:image/png;base64,' + image.base64 },
      ],
    };
  }
  throw new AppProblem(
    'not_found',
    'Choose an imageId from this attached report’s chart inventory.',
  );
}
