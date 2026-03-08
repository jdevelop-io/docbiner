"""Asynchronous Docbiner client."""

from __future__ import annotations

import asyncio
from typing import Any, Dict, List, Optional

import httpx

from .errors import (
    AuthenticationError,
    DocbinerError,
    RateLimitError,
    ServerError,
)
from .types import ConvertOptions

_DEFAULT_BASE_URL = "https://api.docbiner.com"
_DEFAULT_TIMEOUT = 60.0
_MAX_RETRIES = 3
_RETRY_BASE_DELAY = 0.5  # seconds


def _build_options_payload(options: Optional[Dict[str, Any]]) -> Optional[Dict[str, Any]]:
    """Normalize options: accept dict or ConvertOptions dataclass."""
    if options is None:
        return None
    if isinstance(options, ConvertOptions):
        return options.to_dict()
    return options


class AsyncDocbiner:
    """Asynchronous client for the Docbiner API.

    Usage::

        async with AsyncDocbiner("dbk_live_...") as client:
            pdf = await client.convert("<h1>Hello</h1>")
    """

    def __init__(
        self,
        api_key: str,
        *,
        base_url: str = _DEFAULT_BASE_URL,
        timeout: float = _DEFAULT_TIMEOUT,
        max_retries: int = _MAX_RETRIES,
    ) -> None:
        self.api_key = api_key
        self.base_url = base_url
        self.max_retries = max_retries
        self._client = httpx.AsyncClient(
            base_url=base_url,
            headers={
                "Authorization": f"Bearer {api_key}",
                "User-Agent": "docbiner-python/0.1.0",
            },
            timeout=timeout,
        )
        self._jobs = self._Jobs(self)
        self._templates = self._Templates(self)

    # ------------------------------------------------------------------
    # Internal helpers
    # ------------------------------------------------------------------

    async def _request(
        self,
        method: str,
        path: str,
        *,
        json: Optional[Dict[str, Any]] = None,
        params: Optional[Dict[str, Any]] = None,
        raw_response: bool = False,
    ) -> Any:
        """Send an HTTP request with retry on 5xx errors."""
        last_exc: Optional[Exception] = None

        for attempt in range(self.max_retries):
            try:
                response = await self._client.request(
                    method,
                    path,
                    json=json,
                    params=params,
                )
            except httpx.HTTPError as exc:
                last_exc = exc
                if attempt < self.max_retries - 1:
                    await asyncio.sleep(_RETRY_BASE_DELAY * (2 ** attempt))
                    continue
                raise DocbinerError(
                    message=f"HTTP request failed: {exc}",
                    status=0,
                    code="network_error",
                ) from exc

            if response.status_code == 204:
                return None

            if response.status_code == 307:
                location = response.headers.get("location", "")
                if raw_response:
                    async with httpx.AsyncClient() as tmp:
                        r = await tmp.get(location)
                        return r.content
                return {"url": location}

            if 200 <= response.status_code < 300:
                content_type = response.headers.get("content-type", "")
                if raw_response or "application/pdf" in content_type or "image/" in content_type:
                    return response.content
                return response.json()

            await self._handle_error(response, attempt)

        raise ServerError(
            message="Request failed after all retries",
            status=500,
            code="max_retries_exceeded",
        )

    async def _handle_error(self, response: httpx.Response, attempt: int) -> None:
        """Handle non-2xx responses, raising or retrying as appropriate."""
        status = response.status_code

        error_code: Optional[str] = None
        message = f"API error (HTTP {status})"
        try:
            body = response.json()
            error_code = body.get("error")
            message = body.get("message", message)
        except Exception:
            pass

        if status == 401:
            raise AuthenticationError(message=message, status=status, code=error_code)

        if status == 429:
            retry_after = None
            ra_header = response.headers.get("retry-after")
            if ra_header:
                try:
                    retry_after = float(ra_header)
                except ValueError:
                    pass
            raise RateLimitError(
                message=message,
                status=status,
                code=error_code,
                retry_after=retry_after,
            )

        if status >= 500:
            if attempt < self.max_retries - 1:
                await asyncio.sleep(_RETRY_BASE_DELAY * (2 ** attempt))
                return  # retry
            raise ServerError(message=message, status=status, code=error_code)

        raise DocbinerError(message=message, status=status, code=error_code)

    # ------------------------------------------------------------------
    # Convert
    # ------------------------------------------------------------------

    async def convert(
        self,
        source: str,
        format: str = "pdf",
        options: Optional[Dict[str, Any]] = None,
    ) -> bytes:
        """Convert HTML or URL to PDF/image asynchronously.

        Args:
            source: HTML string or URL to convert.
            format: Output format — "pdf", "png", "jpeg", or "webp".
            options: Conversion options (margins, page size, etc.).

        Returns:
            Raw bytes of the generated file.
        """
        payload: Dict[str, Any] = {"source": source, "format": format}
        opts = _build_options_payload(options)
        if opts:
            payload["options"] = opts

        return await self._request("POST", "/v1/convert", json=payload, raw_response=True)

    async def convert_async(
        self,
        source: str,
        format: str = "pdf",
        options: Optional[Dict[str, Any]] = None,
        delivery: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, Any]:
        """Submit an asynchronous conversion job.

        Args:
            source: HTML string or URL to convert.
            format: Output format.
            options: Conversion options.
            delivery: Delivery configuration (webhook or s3).

        Returns:
            Job information dict with id, status, created_at.
        """
        payload: Dict[str, Any] = {"source": source, "format": format}
        opts = _build_options_payload(options)
        if opts:
            payload["options"] = opts
        if delivery:
            payload["delivery"] = delivery

        return await self._request("POST", "/v1/convert/async", json=payload)

    # ------------------------------------------------------------------
    # Jobs
    # ------------------------------------------------------------------

    class _Jobs:
        """Job management operations (async)."""

        def __init__(self, client: AsyncDocbiner) -> None:
            self._client = client

        async def get(self, job_id: str) -> Dict[str, Any]:
            """Get job details by ID."""
            return await self._client._request("GET", f"/v1/jobs/{job_id}")

        async def list(
            self,
            *,
            page: int = 1,
            per_page: int = 20,
            status: Optional[str] = None,
            format: Optional[str] = None,
        ) -> Dict[str, Any]:
            """List jobs with pagination and optional filters."""
            params: Dict[str, Any] = {"page": page, "per_page": per_page}
            if status:
                params["status"] = status
            if format:
                params["format"] = format
            return await self._client._request("GET", "/v1/jobs", params=params)

        async def download(self, job_id: str) -> bytes:
            """Download the result file of a completed job."""
            return await self._client._request(
                "GET", f"/v1/jobs/{job_id}/download", raw_response=True
            )

        async def delete(self, job_id: str) -> None:
            """Delete a job."""
            await self._client._request("DELETE", f"/v1/jobs/{job_id}")

    @property
    def jobs(self) -> _Jobs:
        """Access job management operations."""
        return self._jobs

    # ------------------------------------------------------------------
    # Templates
    # ------------------------------------------------------------------

    class _Templates:
        """Template management operations (async)."""

        def __init__(self, client: AsyncDocbiner) -> None:
            self._client = client

        async def create(
            self,
            name: str,
            engine: str,
            html_content: str,
            *,
            css_content: str = "",
            sample_data: Optional[Dict[str, Any]] = None,
        ) -> Dict[str, Any]:
            """Create a new template."""
            payload: Dict[str, Any] = {
                "name": name,
                "engine": engine,
                "html_content": html_content,
            }
            if css_content:
                payload["css_content"] = css_content
            if sample_data:
                payload["sample_data"] = sample_data
            return await self._client._request("POST", "/v1/templates", json=payload)

        async def get(self, template_id: str) -> Dict[str, Any]:
            """Get a template by ID."""
            return await self._client._request("GET", f"/v1/templates/{template_id}")

        async def list(self) -> List[Dict[str, Any]]:
            """List all templates."""
            return await self._client._request("GET", "/v1/templates")

        async def update(
            self,
            template_id: str,
            **kwargs: Any,
        ) -> Dict[str, Any]:
            """Update a template.

            Accepts keyword arguments: name, engine, html_content,
            css_content, sample_data.
            """
            return await self._client._request(
                "PUT", f"/v1/templates/{template_id}", json=kwargs
            )

        async def delete(self, template_id: str) -> None:
            """Delete a template."""
            await self._client._request("DELETE", f"/v1/templates/{template_id}")

        async def preview(
            self,
            template_id: str,
            data: Optional[Dict[str, Any]] = None,
        ) -> str:
            """Preview a rendered template, returning the HTML string."""
            payload: Dict[str, Any] = {}
            if data:
                payload["data"] = data
            result = await self._client._request(
                "POST", f"/v1/templates/{template_id}/preview", json=payload
            )
            return result.get("html", "")

    @property
    def templates(self) -> _Templates:
        """Access template management operations."""
        return self._templates

    # ------------------------------------------------------------------
    # Merge
    # ------------------------------------------------------------------

    async def merge(
        self,
        sources: List[Dict[str, str]],
        options: Optional[Dict[str, Any]] = None,
    ) -> bytes:
        """Merge multiple HTML/URL sources into a single PDF.

        Args:
            sources: List of dicts with a "source" key (HTML or URL).
            options: Conversion options applied to each source.

        Returns:
            Raw bytes of the merged PDF.
        """
        payload: Dict[str, Any] = {"sources": sources}
        opts = _build_options_payload(options)
        if opts:
            payload["options"] = opts
        return await self._request("POST", "/v1/merge", json=payload, raw_response=True)

    # ------------------------------------------------------------------
    # Usage
    # ------------------------------------------------------------------

    async def usage(self) -> Dict[str, Any]:
        """Get current month usage and quota status."""
        return await self._request("GET", "/v1/usage")

    async def usage_history(self) -> List[Dict[str, Any]]:
        """Get usage history for the last 12 months."""
        return await self._request("GET", "/v1/usage/history")

    # ------------------------------------------------------------------
    # Lifecycle
    # ------------------------------------------------------------------

    async def close(self) -> None:
        """Close the underlying HTTP client."""
        await self._client.aclose()

    async def __aenter__(self) -> AsyncDocbiner:
        return self

    async def __aexit__(self, *args: Any) -> None:
        await self.close()
