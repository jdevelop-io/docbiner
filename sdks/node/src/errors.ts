/**
 * Error thrown when the Docbiner API returns an error response.
 */
export class DocbinerError extends Error {
  /** HTTP status code from the API. */
  public readonly status: number;
  /** Machine-readable error code from the API (e.g. "validation_error"). */
  public readonly code?: string;

  constructor(message: string, status: number, code?: string) {
    super(message);
    this.name = 'DocbinerError';
    this.status = status;
    this.code = code;

    // Restore prototype chain (required when extending built-ins in TS).
    Object.setPrototypeOf(this, new.target.prototype);
  }
}
