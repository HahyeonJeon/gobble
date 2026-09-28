import { Type, type Static } from '@sinclair/typebox';
import { closed, LabelSchema, ProjectIdSchema, ResourceIdSchema } from './identity';

/** Stable work scope. Its local root/resource mapping is owned by the native Project service. */
export const ProjectSchema = Type.Object(
  {
    projectId: ProjectIdSchema,
    name: LabelSchema,
  },
  closed,
);

export const ProjectInfoSchema = Type.Object(
  { ...ProjectSchema.properties, rootResourceId: ResourceIdSchema },
  closed,
);
export type Project = Static<typeof ProjectSchema>;
export type ProjectInfo = Static<typeof ProjectInfoSchema>;
