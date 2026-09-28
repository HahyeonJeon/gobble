import { Type, type Static } from '@sinclair/typebox';
import { closed, SurfaceIdSchema } from './identity';
import type { DependencyGroup, DependencyObservation, TaskDependency } from './run-dependencies';

export const RunModeSchema = Type.Union([Type.Literal('tasks'), Type.Literal('dependencies')]);
export const DependencyCameraSchema = Type.Object(
  {
    zoom: Type.Number({ minimum: 0.25, maximum: 2 }),
    x: Type.Number({ minimum: 0, maximum: 100000 }),
    y: Type.Number({ minimum: 0, maximum: 100000 }),
  },
  closed,
);
export const DependencyNavigationSchema = Type.Object(
  {
    representation: Type.Union([Type.Literal('graph'), Type.Literal('list')]),
    query: Type.String({ maxLength: 200 }),
    camera: DependencyCameraSchema,
  },
  closed,
);
export const DependencyNavigationIntentSchema = Type.Union([
  Type.Object(
    { kind: Type.Literal('runMode'), surfaceId: SurfaceIdSchema, mode: RunModeSchema },
    closed,
  ),
  Type.Object(
    {
      kind: Type.Literal('dependencyCamera'),
      surfaceId: SurfaceIdSchema,
      camera: DependencyCameraSchema,
    },
    closed,
  ),
  Type.Object(
    {
      kind: Type.Literal('dependencyNavigation'),
      surfaceId: SurfaceIdSchema,
      navigation: Type.Omit(DependencyNavigationSchema, ['camera']),
    },
    closed,
  ),
]);
export type DependencyBrowse = Pick<DependencyNavigation, 'representation' | 'query'>;
export type DependencyNavigation = Static<typeof DependencyNavigationSchema>;
export type DependencyCamera = Static<typeof DependencyCameraSchema>;
export const defaultDependencyNavigation = (): DependencyNavigation => ({
  representation: 'graph',
  query: '',
  camera: { zoom: 1, x: 0, y: 0 },
});
const matches = (id: string, query: string) =>
  id.toLocaleLowerCase('en').includes(query.trim().toLocaleLowerCase('en'));
export const matchingDependencyGroups = (
  value: DependencyObservation,
  query: string,
): DependencyGroup[] => value.groups.filter((g) => matches(g.taskId, query));
export const matchingDependencies = (
  value: DependencyObservation,
  query: string,
): TaskDependency[] =>
  value.edges.filter((e) => matches(e.fromTaskId, query) || matches(e.toTaskId, query));
