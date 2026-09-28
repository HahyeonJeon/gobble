import type { ResourceRef } from '@gobble/contracts';
export type OpenResource = (resource: ResourceRef, destination: 'active' | 'other') => void;
export type ReportError = (message: string) => void;
