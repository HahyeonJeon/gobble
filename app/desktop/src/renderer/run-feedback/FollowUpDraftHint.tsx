import type { DraftAttachment, PipelineReviewContext } from '@gobble/contracts';
import { useLaunchReviews } from '../run-launch/useLaunchReviews';
export function FollowUpDraftHint({
  projectId,
  attachments,
  context,
}: {
  projectId: string;
  attachments: DraftAttachment[];
  context: PipelineReviewContext;
}) {
  const { values } = useLaunchReviews(projectId);
  const refs = new Set(
    attachments.flatMap((a) =>
      a.capture?.kind === 'observed' && 'runRef' in a.evidence.resource
        ? [a.evidence.resource.runRef]
        : [],
    ),
  );
  if (!refs.size || context.proposalId) return null;
  const origin = values.find((r) => r.operation?.runRef === [...refs][0]);
  return (
    <p className="follow-up-draft" role="status">
      {refs.size !== 1
        ? 'Use evidence from one analysis for a linked change.'
        : !origin || origin.pipelineId !== context.pipelineId
          ? 'The attached analysis does not belong to this Current design. Choose its Pipeline before requesting a change.'
          : origin.preparation.artifactId !== context.baseArtifactId
            ? 'This evidence is from an earlier design. Your proposal will target the explicitly selected Current; the earlier Run stays unchanged.'
            : 'The proposal will retain a link to this analysis and the evidence you send. Adopting a change does not start work.'}
    </p>
  );
}
