/** Per-Surface navigation, retained by the active Project across presenter remounts.
 * It is not source state, an Agent capability, or a durable Workspace record. */
export type FlowNavigation = {
  reviewing: boolean;
  proposalId: string | null;
  changeId: string | null;
  artifactId: string;
  list: boolean;
  zoom: number | null;
  left: number;
  top: number;
};
export const emptyFlowNavigation = (): FlowNavigation => ({
  reviewing: false,
  proposalId: null,
  changeId: null,
  artifactId: '',
  list: false,
  zoom: null,
  left: 0,
  top: 0,
});
