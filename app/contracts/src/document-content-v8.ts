// Frozen nested v7/v8 storage components; live evidence and Surface fields must not leak in.
import { Type } from '@sinclair/typebox';
import { closed } from './identity';
import { WorkspaceSchema } from './workspace';
import { QuestionSchema } from './question';
import { LegacyDecisionSchema } from './decision';
import { SubmissionSchema } from './collaboration';
import { SurfaceV8Schema } from './surface-v8';
import { EvidenceManifestV8Schema } from './evidence-v8';
const question = Type.Object(
  {
    ...QuestionSchema.properties,
    evidence: Type.Array(EvidenceManifestV8Schema, { minItems: 1, maxItems: 16 }),
  },
  closed,
);
const submission = Type.Object(
  {
    ...Type.Omit(SubmissionSchema, [
      'pipelineReview',
      'allowPipelineProposal',
      'pipelineCreation',
      'allowPipelineCreation',
    ]).properties,
    evidence: Type.Optional(Type.Array(EvidenceManifestV8Schema, { minItems: 1, maxItems: 16 })),
  },
  closed,
);
export const CollaborationHistoryV8Schema = Type.Object(
  { submissions: Type.Array(submission, { maxItems: 40 }) },
  closed,
);
export const WorkspaceV8Schema = Type.Object(
  {
    ...WorkspaceSchema.properties,
    surfaces: Type.Array(SurfaceV8Schema, { maxItems: 64 }),
    decisions: Type.Array(Type.Union([LegacyDecisionSchema, question]), { maxItems: 1000 }),
  },
  closed,
);
