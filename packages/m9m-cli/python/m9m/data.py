"""Data types for the m9m Python SDK."""

from __future__ import annotations

import json as _json
from dataclasses import dataclass, field
from typing import Any


@dataclass
class DataItem:
    json: dict[str, Any] = field(default_factory=dict)
    binary: dict[str, Any] | None = None
    pairedItem: dict[str, Any] | None = None
    error: dict[str, Any] | None = None

    @classmethod
    def from_dict(cls, data: dict[str, Any]) -> "DataItem":
        return cls(
            json=data.get("json", {}),
            binary=data.get("binary"),
            pairedItem=data.get("pairedItem"),
            error=data.get("error"),
        )

    def to_dict(self) -> dict[str, Any]:
        result: dict[str, Any] = {"json": self.json}
        if self.binary is not None:
            result["binary"] = self.binary
        if self.pairedItem is not None:
            result["pairedItem"] = self.pairedItem
        if self.error is not None:
            result["error"] = self.error
        return result


@dataclass
class ExecutionResult:
    data: list[DataItem] = field(default_factory=list)
    error: str | None = None

    @property
    def success(self) -> bool:
        return self.error is None

    @classmethod
    def from_json(cls, json_str: str) -> "ExecutionResult":
        raw = _json.loads(json_str)
        items = [DataItem.from_dict(item) for item in raw.get("data", [])]
        return cls(data=items, error=raw.get("error"))

    def to_json(self) -> str:
        return _json.dumps(
            {
                "data": [item.to_dict() for item in self.data],
                "error": self.error,
            }
        )
