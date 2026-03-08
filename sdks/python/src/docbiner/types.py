"""Type definitions for the Docbiner SDK."""

from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any, Dict, List, Optional


@dataclass
class EncryptOptions:
    """PDF encryption options."""

    user_password: str = ""
    owner_password: str = ""


@dataclass
class ConvertOptions:
    """Options for HTML-to-PDF/image conversion."""

    # PDF options
    page_size: str = "A4"
    landscape: bool = False
    margin_top: str = "20mm"
    margin_right: str = "20mm"
    margin_bottom: str = "20mm"
    margin_left: str = "20mm"
    header_html: str = ""
    footer_html: str = ""
    scale: float = 1.0
    print_background: bool = True

    # Screenshot options
    width: int = 0
    height: int = 0
    quality: int = 0
    full_page: bool = False

    # Shared options
    css: str = ""
    js: str = ""
    wait_for: str = ""
    delay_ms: int = 0

    # PDF encryption
    encrypt: Optional[EncryptOptions] = None

    def to_dict(self) -> Dict[str, Any]:
        """Convert to API-compatible dictionary, omitting defaults."""
        result: Dict[str, Any] = {}
        defaults = ConvertOptions()
        for fld in self.__dataclass_fields__:
            val = getattr(self, fld)
            default_val = getattr(defaults, fld)
            if val != default_val and val:
                if isinstance(val, EncryptOptions):
                    result[fld] = {
                        "user_password": val.user_password,
                        "owner_password": val.owner_password,
                    }
                else:
                    result[fld] = val
        return result


@dataclass
class DeliveryConfig:
    """Delivery configuration for async conversions."""

    method: str = ""
    config: Optional[Dict[str, Any]] = None

    def to_dict(self) -> Dict[str, Any]:
        d: Dict[str, Any] = {"method": self.method}
        if self.config:
            d["config"] = self.config
        return d


@dataclass
class MergeSource:
    """A single source for PDF merging."""

    source: str = ""

    def to_dict(self) -> Dict[str, str]:
        return {"source": self.source}


@dataclass
class PaginationMeta:
    """Pagination metadata from list responses."""

    page: int = 1
    per_page: int = 20
    total: int = 0
    total_pages: int = 0


@dataclass
class JobListResponse:
    """Paginated list of jobs."""

    data: List[Dict[str, Any]] = field(default_factory=list)
    pagination: Optional[PaginationMeta] = None


@dataclass
class QuotaStatus:
    """Current quota state for the organization."""

    allowed: bool = True
    used: int = 0
    limit: int = 0
    remaining: int = 0


@dataclass
class UsageResponse:
    """Current month usage with quota."""

    month: str = ""
    conversions: int = 0
    test_conversions: int = 0
    quota: Optional[QuotaStatus] = None


@dataclass
class MonthlyUsage:
    """Usage data for a single month."""

    month: str = ""
    conversions: int = 0
    test_conversions: int = 0
    overage_amount: float = 0.0
