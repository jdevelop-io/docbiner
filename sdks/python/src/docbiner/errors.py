"""Error types for the Docbiner SDK."""

from __future__ import annotations

from typing import Optional


class DocbinerError(Exception):
    """Base exception for Docbiner API errors.

    Attributes:
        message: Human-readable error description.
        status: HTTP status code from the API.
        code: Machine-readable error code (e.g. "validation_error").
    """

    def __init__(
        self,
        message: str,
        status: int,
        code: Optional[str] = None,
    ) -> None:
        super().__init__(message)
        self.message = message
        self.status = status
        self.code = code

    def __repr__(self) -> str:
        return f"DocbinerError(message={self.message!r}, status={self.status}, code={self.code!r})"


class AuthenticationError(DocbinerError):
    """Raised on 401 Unauthorized responses."""


class RateLimitError(DocbinerError):
    """Raised on 429 Too Many Requests responses.

    Attributes:
        retry_after: Seconds to wait before retrying, if provided by the API.
    """

    def __init__(
        self,
        message: str,
        status: int = 429,
        code: Optional[str] = None,
        retry_after: Optional[float] = None,
    ) -> None:
        super().__init__(message, status, code)
        self.retry_after = retry_after


class ServerError(DocbinerError):
    """Raised on 5xx server errors after all retries are exhausted."""
