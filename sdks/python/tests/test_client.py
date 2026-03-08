"""Tests for the synchronous Docbiner client."""

import httpx
import pytest
import respx

from docbiner import (
    AsyncDocbiner,
    AuthenticationError,
    ConvertOptions,
    Docbiner,
    DocbinerError,
    RateLimitError,
    ServerError,
)

BASE_URL = "https://api.docbiner.com"


# ------------------------------------------------------------------
# Client initialization
# ------------------------------------------------------------------


class TestClientInit:
    def test_default_initialization(self):
        client = Docbiner("dbk_live_test123")
        assert client.api_key == "dbk_live_test123"
        assert client.base_url == BASE_URL
        assert client.max_retries == 3
        client.close()

    def test_custom_base_url(self):
        client = Docbiner("key", base_url="https://custom.api.com")
        assert client.base_url == "https://custom.api.com"
        client.close()

    def test_context_manager(self):
        with Docbiner("dbk_live_test123") as client:
            assert client.api_key == "dbk_live_test123"

    def test_has_jobs_property(self):
        with Docbiner("key") as client:
            assert client.jobs is not None

    def test_has_templates_property(self):
        with Docbiner("key") as client:
            assert client.templates is not None


# ------------------------------------------------------------------
# Convert
# ------------------------------------------------------------------


class TestConvert:
    @respx.mock
    def test_convert_returns_bytes(self):
        pdf_bytes = b"%PDF-1.4 fake content"
        respx.post(f"{BASE_URL}/v1/convert").mock(
            return_value=httpx.Response(
                200,
                content=pdf_bytes,
                headers={"content-type": "application/pdf"},
            )
        )

        with Docbiner("dbk_live_test123") as client:
            result = client.convert("<h1>Hello</h1>")

        assert result == pdf_bytes

    @respx.mock
    def test_convert_sends_correct_payload(self):
        route = respx.post(f"{BASE_URL}/v1/convert").mock(
            return_value=httpx.Response(
                200,
                content=b"pdf",
                headers={"content-type": "application/pdf"},
            )
        )

        with Docbiner("dbk_live_test123") as client:
            client.convert("<h1>Hi</h1>", format="png", options={"width": 800})

        request = route.calls.last.request
        import json

        body = json.loads(request.content)
        assert body["source"] == "<h1>Hi</h1>"
        assert body["format"] == "png"
        assert body["options"]["width"] == 800

    @respx.mock
    def test_convert_with_options_dataclass(self):
        route = respx.post(f"{BASE_URL}/v1/convert").mock(
            return_value=httpx.Response(
                200,
                content=b"pdf",
                headers={"content-type": "application/pdf"},
            )
        )

        opts = ConvertOptions(page_size="Letter", landscape=True)
        with Docbiner("dbk_live_test123") as client:
            client.convert("<h1>Hi</h1>", options=opts.to_dict())

        import json

        body = json.loads(route.calls.last.request.content)
        assert body["options"]["page_size"] == "Letter"
        assert body["options"]["landscape"] is True

    @respx.mock
    def test_convert_async_returns_job(self):
        job_response = {
            "id": "550e8400-e29b-41d4-a716-446655440000",
            "status": "pending",
            "created_at": "2026-01-01T00:00:00Z",
        }
        respx.post(f"{BASE_URL}/v1/convert/async").mock(
            return_value=httpx.Response(202, json=job_response)
        )

        with Docbiner("dbk_live_test123") as client:
            result = client.convert_async(
                "<h1>Hi</h1>",
                delivery={"method": "webhook", "config": {"url": "https://example.com/hook"}},
            )

        assert result["id"] == "550e8400-e29b-41d4-a716-446655440000"
        assert result["status"] == "pending"


# ------------------------------------------------------------------
# Error handling
# ------------------------------------------------------------------


class TestErrorHandling:
    @respx.mock
    def test_401_raises_authentication_error(self):
        respx.post(f"{BASE_URL}/v1/convert").mock(
            return_value=httpx.Response(
                401,
                json={"error": "unauthorized", "message": "Invalid API key"},
            )
        )

        with Docbiner("bad_key") as client:
            with pytest.raises(AuthenticationError) as exc_info:
                client.convert("<h1>Hi</h1>")

        assert exc_info.value.status == 401
        assert exc_info.value.code == "unauthorized"

    @respx.mock
    def test_429_raises_rate_limit_error(self):
        respx.post(f"{BASE_URL}/v1/convert").mock(
            return_value=httpx.Response(
                429,
                json={"error": "rate_limited", "message": "Too many requests"},
                headers={"retry-after": "30"},
            )
        )

        with Docbiner("key") as client:
            with pytest.raises(RateLimitError) as exc_info:
                client.convert("<h1>Hi</h1>")

        assert exc_info.value.status == 429
        assert exc_info.value.retry_after == 30.0

    @respx.mock
    def test_400_raises_docbiner_error(self):
        respx.post(f"{BASE_URL}/v1/convert").mock(
            return_value=httpx.Response(
                400,
                json={"error": "validation_error", "message": "source is required"},
            )
        )

        with Docbiner("key") as client:
            with pytest.raises(DocbinerError) as exc_info:
                client.convert("")

        assert exc_info.value.status == 400
        assert exc_info.value.code == "validation_error"


# ------------------------------------------------------------------
# Retry on 5xx
# ------------------------------------------------------------------


class TestRetry:
    @respx.mock
    def test_retry_on_500_then_success(self):
        route = respx.post(f"{BASE_URL}/v1/convert")
        route.side_effect = [
            httpx.Response(
                500,
                json={"error": "internal_error", "message": "Server error"},
            ),
            httpx.Response(
                200,
                content=b"%PDF-ok",
                headers={"content-type": "application/pdf"},
            ),
        ]

        with Docbiner("key", max_retries=3) as client:
            result = client.convert("<h1>Hi</h1>")

        assert result == b"%PDF-ok"
        assert route.call_count == 2

    @respx.mock
    def test_exhausted_retries_raises_server_error(self):
        respx.post(f"{BASE_URL}/v1/convert").mock(
            return_value=httpx.Response(
                502,
                json={"error": "bad_gateway", "message": "Bad gateway"},
            )
        )

        with Docbiner("key", max_retries=2) as client:
            with pytest.raises(ServerError) as exc_info:
                client.convert("<h1>Hi</h1>")

        assert exc_info.value.status == 502

    @respx.mock
    def test_no_retry_on_4xx(self):
        route = respx.post(f"{BASE_URL}/v1/convert").mock(
            return_value=httpx.Response(
                422,
                json={"error": "validation_error", "message": "Invalid"},
            )
        )

        with Docbiner("key", max_retries=3) as client:
            with pytest.raises(DocbinerError):
                client.convert("<h1>Hi</h1>")

        assert route.call_count == 1


# ------------------------------------------------------------------
# Jobs
# ------------------------------------------------------------------


class TestJobs:
    @respx.mock
    def test_get_job(self):
        job_id = "550e8400-e29b-41d4-a716-446655440000"
        job_data = {"id": job_id, "status": "completed"}
        respx.get(f"{BASE_URL}/v1/jobs/{job_id}").mock(
            return_value=httpx.Response(200, json=job_data)
        )

        with Docbiner("key") as client:
            result = client.jobs.get(job_id)

        assert result["id"] == job_id
        assert result["status"] == "completed"

    @respx.mock
    def test_list_jobs(self):
        response_data = {
            "data": [{"id": "abc", "status": "pending"}],
            "pagination": {"page": 1, "per_page": 20, "total": 1, "total_pages": 1},
        }
        respx.get(f"{BASE_URL}/v1/jobs").mock(
            return_value=httpx.Response(200, json=response_data)
        )

        with Docbiner("key") as client:
            result = client.jobs.list(page=1, per_page=20, status="pending")

        assert len(result["data"]) == 1

    @respx.mock
    def test_delete_job(self):
        job_id = "abc123"
        respx.delete(f"{BASE_URL}/v1/jobs/{job_id}").mock(
            return_value=httpx.Response(204)
        )

        with Docbiner("key") as client:
            client.jobs.delete(job_id)  # should not raise


# ------------------------------------------------------------------
# Templates
# ------------------------------------------------------------------


class TestTemplates:
    @respx.mock
    def test_create_template(self):
        tpl_data = {
            "id": "tpl-123",
            "name": "Invoice",
            "engine": "handlebars",
            "html_content": "<h1>{{title}}</h1>",
        }
        respx.post(f"{BASE_URL}/v1/templates").mock(
            return_value=httpx.Response(201, json=tpl_data)
        )

        with Docbiner("key") as client:
            result = client.templates.create(
                name="Invoice",
                engine="handlebars",
                html_content="<h1>{{title}}</h1>",
            )

        assert result["name"] == "Invoice"

    @respx.mock
    def test_list_templates(self):
        respx.get(f"{BASE_URL}/v1/templates").mock(
            return_value=httpx.Response(200, json=[{"id": "t1"}, {"id": "t2"}])
        )

        with Docbiner("key") as client:
            result = client.templates.list()

        assert len(result) == 2

    @respx.mock
    def test_preview_template(self):
        tpl_id = "tpl-123"
        respx.post(f"{BASE_URL}/v1/templates/{tpl_id}/preview").mock(
            return_value=httpx.Response(200, json={"html": "<h1>Hello World</h1>"})
        )

        with Docbiner("key") as client:
            html = client.templates.preview(tpl_id, data={"title": "Hello World"})

        assert html == "<h1>Hello World</h1>"


# ------------------------------------------------------------------
# Merge
# ------------------------------------------------------------------


class TestMerge:
    @respx.mock
    def test_merge_returns_bytes(self):
        pdf_bytes = b"%PDF-merged"
        respx.post(f"{BASE_URL}/v1/merge").mock(
            return_value=httpx.Response(
                200,
                content=pdf_bytes,
                headers={"content-type": "application/pdf"},
            )
        )

        with Docbiner("key") as client:
            result = client.merge(
                sources=[{"source": "<h1>Page 1</h1>"}, {"source": "<h1>Page 2</h1>"}]
            )

        assert result == pdf_bytes


# ------------------------------------------------------------------
# Usage
# ------------------------------------------------------------------


class TestUsage:
    @respx.mock
    def test_get_usage(self):
        usage_data = {
            "month": "2026-03",
            "conversions": 42,
            "test_conversions": 5,
            "quota": {"allowed": True, "used": 42, "limit": 1000, "remaining": 958},
        }
        respx.get(f"{BASE_URL}/v1/usage").mock(
            return_value=httpx.Response(200, json=usage_data)
        )

        with Docbiner("key") as client:
            result = client.usage()

        assert result["conversions"] == 42
        assert result["quota"]["remaining"] == 958

    @respx.mock
    def test_get_usage_history(self):
        history = [
            {"month": "2026-03", "conversions": 42, "test_conversions": 5, "overage_amount": 0},
            {"month": "2026-02", "conversions": 100, "test_conversions": 10, "overage_amount": 0},
        ]
        respx.get(f"{BASE_URL}/v1/usage/history").mock(
            return_value=httpx.Response(200, json=history)
        )

        with Docbiner("key") as client:
            result = client.usage_history()

        assert len(result) == 2


# ------------------------------------------------------------------
# Async client
# ------------------------------------------------------------------


class TestAsyncClient:
    @respx.mock
    @pytest.mark.asyncio
    async def test_async_convert(self):
        pdf_bytes = b"%PDF-async"
        respx.post(f"{BASE_URL}/v1/convert").mock(
            return_value=httpx.Response(
                200,
                content=pdf_bytes,
                headers={"content-type": "application/pdf"},
            )
        )

        async with AsyncDocbiner("dbk_live_test123") as client:
            result = await client.convert("<h1>Hello</h1>")

        assert result == pdf_bytes

    @respx.mock
    @pytest.mark.asyncio
    async def test_async_401_error(self):
        respx.post(f"{BASE_URL}/v1/convert").mock(
            return_value=httpx.Response(
                401,
                json={"error": "unauthorized", "message": "Invalid API key"},
            )
        )

        async with AsyncDocbiner("bad_key") as client:
            with pytest.raises(AuthenticationError):
                await client.convert("<h1>Hi</h1>")

    @respx.mock
    @pytest.mark.asyncio
    async def test_async_context_manager(self):
        async with AsyncDocbiner("key") as client:
            assert client.api_key == "key"
