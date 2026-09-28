import { useState } from 'react';
import type { AccountAction } from '@gobble/contracts';
import { ModalDialog } from '../ui/ModalDialog';
import type { AgentsModel } from './useAgents';

export function AccountPanel({ model }: { model: AgentsModel }) {
  const [open, setOpen] = useState(false);
  const [busy, setBusy] = useState(false);
  const account = model.status?.account;
  const action = async (action: AccountAction['action']) => {
    setBusy(true);
    try {
      await model.account({ action });
    } finally {
      setBusy(false);
    }
  };
  return (
    <div className="account-control">
      <button
        onClick={() => setOpen(true)}
        aria-label="Account"
        title={
          account?.status === 'signedIn' && account.runtime === 'ready'
            ? 'Account · Connected'
            : 'Account · Check connection'
        }
      >
        <span
          className={
            'account-dot' +
            (account?.status === 'signedIn' && account.runtime === 'ready' ? ' connected' : '')
          }
          aria-hidden="true"
        />
        Account
      </button>
      {open && (
        <ModalDialog title="Codex account" onClose={() => setOpen(false)}>
          <p role="status">
            {account?.status === 'signedIn' && account.runtime === 'ready'
              ? 'ChatGPT connected'
              : account?.status === 'signingIn'
                ? 'Waiting for sign-in'
                : 'Not connected'}
          </p>
          <p>Use your ChatGPT account to collaborate with agents in Gobble.</p>
          <dl>
            <div>
              <dt>Runtime</dt>
              <dd>
                {account?.version ?? '0.153.4'} · {account?.runtime ?? 'disconnected'}
              </dd>
            </div>
            {account?.email && (
              <div>
                <dt>Account</dt>
                <dd>{account.email}</dd>
              </div>
            )}
            {account?.plan && (
              <div>
                <dt>Plan</dt>
                <dd>{account.plan}</dd>
              </div>
            )}
          </dl>
          {account?.problem && (
            <p role="alert" className="inline-error">
              {account.problem}
            </p>
          )}
          {account?.status === 'signingIn' ? (
            <>
              <p role="status">Complete sign-in in your browser.</p>
              <button disabled={busy} onClick={() => void action('cancelSignIn')}>
                Cancel sign-in
              </button>
            </>
          ) : account?.status === 'signedIn' ? (
            <button disabled={busy} onClick={() => void action('signOut')}>
              Sign out
            </button>
          ) : (
            <>
              <button
                className="primary-button"
                disabled={busy}
                onClick={() => void action('signIn')}
              >
                Sign in with ChatGPT
              </button>
              <p className="muted">
                Uses your browser. Credentials stay with the official Codex runtime in Gobble’s
                profile.
              </p>
            </>
          )}
          <button disabled={busy} onClick={() => void action('connect')}>
            {busy ? 'Connecting…' : 'Refresh connection'}
          </button>
        </ModalDialog>
      )}
    </div>
  );
}
