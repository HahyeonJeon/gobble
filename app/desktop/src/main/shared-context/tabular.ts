import {
  filterTable,
  sortTable,
  type ReferenceView,
  type RenderAcknowledgment,
  type Selection,
  type Surface,
  type SurfaceData,
  type ViewLink,
} from '@gobble/contracts';
import { AppProblem } from '../problem';
import { textObservation } from './observation';

type View = {
  surface: Surface;
  link: ViewLink | undefined;
  data: SurfaceData;
  acknowledgment: RenderAcknowledgment;
  referenceView?: ReferenceView;
};

/** CSV observation is table data. Legacy presentation is retained only on an explicit archive reveal. */
export function observeTabular(
  view: View,
  selection: Extract<Selection, { kind: 'text' | 'image' | 'table' }> | undefined,
  scope: 'view' | 'source-preview',
) {
  if (view.surface.view === 'scatter')
    throw new AppProblem('unsupported', 'Chart views have been retired. Open the source table.');
  if (view.data.kind !== 'file' || view.data.value.content.kind !== 'table') return undefined;
  const content = view.data.value.content;
  const presentation = view.referenceView?.presentation;
  const displayed = filterTable(
    content,
    presentation?.filter ?? view.link?.filter ?? { kind: 'all' },
  );
  let data = view.data;
  if (!selection && scope === 'view')
    data = {
      ...data,
      value: {
        ...data.value,
        content: {
          ...content,
          rows: sortTable(
            content,
            displayed,
            view.surface.view === 'table' ? (view.surface.table?.sort ?? null) : null,
          ),
        },
      },
    };
  const observed = textObservation(data, selection);
  return {
    content: observed,
    selection,
    presentation,
    scope: {
      kind: selection
        ? 'requested-source-rows'
        : scope === 'view'
          ? 'displayed-preview'
          : 'source-preview',
      previewRows: content.rows.length,
      displayedRows: displayed.length,
      returnedRows: observed.kind === 'table' ? observed.rows.length : 0,
      sourceTruncated: content.truncated,
      excludedCoordinates: 0,
    },
  };
}
