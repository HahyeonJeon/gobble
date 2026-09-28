import type { ContractError } from '@gobble/contracts';

export class AppProblem extends Error {
  constructor(
    readonly code: ContractError['code'],
    message: string,
  ) {
    super(message);
  }
}
