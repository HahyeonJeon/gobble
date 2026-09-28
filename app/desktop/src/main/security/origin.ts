export const PACKAGED_URL = 'app://gobble/index.html';
export const DEVELOPMENT_URL = 'http://127.0.0.1:5173/';

export function rendererURL(packaged: boolean, developmentURL: string | undefined): string {
  if (packaged || developmentURL === undefined) return PACKAGED_URL;
  if (new URL(developmentURL).href !== DEVELOPMENT_URL)
    throw new Error('Unsupported development renderer origin.');
  return DEVELOPMENT_URL;
}

export function isTrustedSender(
  sender: { isMainFrame: boolean; isWorkspaceWindow: boolean; url: string },
  expectedURL: string,
): boolean {
  return sender.isMainFrame && sender.isWorkspaceWindow && sender.url === expectedURL;
}
