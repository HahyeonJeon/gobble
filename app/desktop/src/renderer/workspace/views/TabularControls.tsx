import { useState } from 'react';
import {
  type RenderAcknowledgment,
  type Surface,
  type TableContent,
  type ViewLink,
} from '@gobble/contracts';
import type { Command } from '../useWorkspace';
import { FilterSettings, TableSettings } from './TabularSettings';

/** These controls send explicit intents; drafts inside dialogs are never workspace state. */
export function TabularControls({
  surface,
  content,
  link,
  acknowledgment,
  ready,
  command,
}: {
  surface: Surface;
  content: TableContent;
  link: ViewLink | undefined;
  acknowledgment: RenderAcknowledgment;
  ready: boolean;
  command: Command;
}) {
  const [dialog, setDialog] = useState<'filter' | 'table' | null>(null);
  const observed = { surfaceId: surface.surfaceId, acknowledgment };
  const close = () => setDialog(null);
  return (
    <>
      <button disabled={!ready} onClick={() => setDialog('table')}>
        Table settings
      </button>
      {link && (
        <button
          disabled={!ready}
          aria-label="Filter linked views"
          aria-pressed={link.filter.kind !== 'all'}
          onClick={() => setDialog('filter')}
        >
          {link.filter.kind === 'all' ? 'Filter' : 'Filter applied'}
        </button>
      )}
      {dialog === 'table' && surface.view === 'table' && (
        <TableSettings
          content={content}
          initial={surface.table ?? { sort: null, columns: null }}
          onClose={close}
          onApply={(settings) => command({ kind: 'tableSettings', ...observed, settings })}
        />
      )}
      {dialog === 'filter' && link && (
        <FilterSettings
          content={content}
          initial={link.filter}
          onClose={close}
          onApply={(filter) => command({ kind: 'linkedFilter', ...observed, filter })}
        />
      )}
    </>
  );
}
