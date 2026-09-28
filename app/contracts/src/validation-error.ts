export class ContractValidationError extends Error {
  constructor(message = 'Data does not match the contract.') {
    super(message);
    this.name = 'ContractValidationError';
  }
}
