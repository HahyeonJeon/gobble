import { toolSchemas } from '@gobble/contracts';
import type { ToolDefinition } from './ports';
export { toolSchemas } from '@gobble/contracts';
const descriptions: Record<keyof typeof toolSchemas, string> = {
  read_report:
    'Read a whole saved report attached to your current message. Pass its attachmentId; omit imageId for complete original text/tables and chart inventory, or use one returned imageId for its original PNG. Inventory is not pixel coverage. No earlier-message, arbitrary hash/path or other-recipient access. Does not grant viewport observation or pointing authority.',
  gobble_creation_source:
    'Read the scoped draft source for this message. Requires explicit allowPipelineCreation. Input/setup/runtime stay immutable.',
  gobble_creation_propose:
    'Submit one complete pipeline source file from gobble_creation_source for this message. Starts Gobble checking, never adopts or runs. Poll gobble_creation_status for the result.',
  gobble_creation_status:
    'List candidate status for this message’s draft. Use the returned exact context with gobble_creation_review. Never fabricate a successful check.',
  gobble_creation_review:
    'Read the exact retained first-version flow and input/settings/connection facts. No current base exists. These are retained facts, not viewport observation.',
  gobble_creation_point:
    'Publish a reference to an exact addition in a candidate read during this turn. Requires a target. Does not move the User selection or edit input binding.',
  gobble_pipeline_proposals:
    'List bounded proposal identities and status for one registered Pipeline in this Project. Use returned exact comparison context with gobble_pipeline_review once ready. No source, viewport observation, adoption or execution authority.',
  gobble_pipeline_source:
    'Read only the existing Pipeline source scope explicitly authorized for this message. Requires allowPipelineProposal. Returns retained checked source, never research-folder files. No arbitrary paths or shell.',
  gobble_pipeline_propose:
    'Submit complete replacement bytes for existing files returned by gobble_pipeline_source. Requires this message’s explicit proposal permission. Starts an isolated check and returns a proposal identity. Does not adopt, overwrite imports, execute or start a Run.',
  gobble_pipeline_review:
    'Read an exact retained comparison or run preparation. For a preparation, use the preparationId and optional preparationSection (data, settings, environment) received in Chat. This is source evidence, not proof of the user’s current viewport. Use change identities and both artifact identities, never infer status from colors.',
  gobble_pipeline_point:
    'Publish an Agent reference to an exact comparison change or a run preparation section read in this turn. Preparation pointing requires preparationSection: data, settings or environment. Does not alter User selection, draft or current Pipeline, and does not claim viewport observation.',
  workspace_question:
    'Ask the user a durable question in Project chat, using 1–16 evidenceIds from workspace_observe receipts or attachmentIds explicitly received in this conversation. Optional suggestions are text. Returns immediately with a question ID; finish your turn and await a later addressed reply. Cannot answer for the user.',
  workspace_list:
    'List this Project’s open resource views, layout and published pointers. No private drafts or other Agent messages. Opening is not observation.',
  workspace_resources:
    'List one bounded registered Project directory using an opaque directory ID; omit for the root. Names and IDs only, no source bytes.',
  workspace_open:
    'Open one or two registered resources. CSV files open as tables; PDFs open in the page reader; saved Notebooks open in the read-only cell reader. User-owned views are protected; result distinguishes background from revealed. Does not prove content was observed.',
  workspace_arrange:
    'Move your own unclaimed temporary view into an available Pane. User-owned views and focus are protected.',
  workspace_observe:
    'Read a ready foreground view. Report views return whole-report identity and the selected section heading, not chart pixels or complete report content. Use a current-message attachment and read_report for complete report reading. Report pointing is whole-report only with both returned receipts.  Pipeline views return checked artifact v7 targets for steps, directional ports, exact connections and module-published settings; use returned subject.target, receipt.observedReadId and receipt.observation when pointing. Request an explicit pipeline selector for one subject. This is design metadata, not Run status or editable source. Saved Notebooks return schemaVersion 6 targets with exact revision/profile/cell/part and notebook-display-utf16 text or natural-image-pixels rectangles. Default Notebook observation returns only visible source/text output ranges (up to 4096 units), and advertises painted image regions without pixels or pointing authority. Explicitly observe one returned image selector to receive pixels. A single explicit part selector also provides question evidence; source-preview requires that selector and is never pointable. Scroll, folding or changed viewport invalidates Notebook receipts. Basic Markdown presentation is not an exact source observation; ask the User to reveal Source. Return from a temporary Notebook reference before observing. No execution, widgets or raw file access. PDF observations return PNG pixels with schemaVersion 5 PDF-user-space coordinates, source revision, page/model identity and viewport transform. Default PDF scope is the clipped visible region; optional PDF selection must be inside it. PDF source-preview reads only the current decoded page and cannot authorize pointing. No off-page reads or text targeting. Scrolling invalidates a PDF observation. Runs support representation tasks or dependencies (default: displayed mode). Dependencies return bounded authored groups and directed pairs with exact schemaVersion 4 targets; whole graphs have no single evidence target. Tasks/logs return schemaVersion 3 evidence. A Run may include continuation context: the exact saved review identity and reuse/restart/start plan shown beside its observed tasks. Discuss or point using that same evidence revision; the plan is neither live execution status nor permission to execute. Only the User confirms continuation in Chat. Both return observedReadId; optional stream chooses stdout/stderr. Use scope source-preview explicitly for another stream or filtered task; no User setting changes and no visual pointing permission. Default scope is the displayed preview; scope source-preview explicitly reads the bounded source including filtered-out rows. Optional selection declares coordinateSpace: utf16-line-column (1-based lines, 0-based UTF-16 columns, half-open), revision-row-column-keys, or normalized-original-image (top-left origin). Returns evidence identity, a full observation receipt, explicit scope and content. Display returned image content before describing pixels.',
  workspace_point:
    'Publish a labelled Project pointer to evidence from a current observation. For schemaVersion 3/4 Run/log, 5 PDF, 6 Notebook or 7 Pipeline evidence, observedReadId AND observation are required; the target must be inside the returned task list or stream range, or match an exact returned group/directed pair. A PDF region must be contained in the returned region on the same page/model and still-current visible presentation. Notebook targets must fit inside a returned visible text range or explicitly observed image rectangle, with the same cell/part/profile and current viewport. Source-preview is not pointable. Use the receipt’s evidence Project/resource/revision identity and a valid selection. Origin is optional; a matching ready view is required. Authorship is assigned by Gobble App. Does not change the user’s selection or send a message.',
  workspace_release:
    'Close only your own unpinned, unclaimed temporary view. Does not delete its source.',
  gobble_runs:
    'List already registered Gobble Runs for this Project. Does not execute or attach a Run.',
  gobble_run: 'Read an existing registered Run’s bounded status projection.',
  gobble_logs:
    'Read a bounded existing registered Run attempt’s logs using its exact instance ID and attempt.',
};
export const sharedToolDefinitions: ToolDefinition[] = Object.entries(toolSchemas).map(
  ([name, schema]) => ({
    name,
    schema,
    description: descriptions[name as keyof typeof descriptions],
  }),
);
