"""Workflow wrapper for the m9m Python SDK."""

from __future__ import annotations

import json as _json
from typing import Any


class Workflow:
    """Represents an m9m workflow."""

    def __init__(
        self,
        name: str = "",
        id: str | None = None,
        active: bool = True,
        nodes: list[dict[str, Any]] | None = None,
        connections: dict[str, Any] | None = None,
        settings: dict[str, Any] | None = None,
    ):
        self.name = name
        self.id = id or ""
        self.active = active
        self.nodes = nodes or []
        self.connections = connections or {}
        self.settings = settings or {}

    @classmethod
    def from_file(cls, path: str) -> "Workflow":
        with open(path, "r") as f:
            return cls.from_json(f.read())

    @classmethod
    def from_json(cls, json_str: str) -> "Workflow":
        data = _json.loads(json_str)
        return cls.from_dict(data)

    @classmethod
    def from_dict(cls, data: dict[str, Any]) -> "Workflow":
        return cls(
            name=data.get("name", ""),
            id=data.get("id"),
            active=data.get("active", True),
            nodes=data.get("nodes"),
            connections=data.get("connections"),
            settings=data.get("settings"),
        )

    def to_json(self) -> str:
        return _json.dumps(self.to_dict())

    def to_dict(self) -> dict[str, Any]:
        result: dict[str, Any] = {
            "name": self.name,
            "active": self.active,
            "nodes": self.nodes,
            "connections": self.connections,
        }
        if self.id:
            result["id"] = self.id
        if self.settings:
            result["settings"] = self.settings
        return result

    def __enter__(self) -> "Workflow":
        return self

    def __exit__(self, *args: Any) -> None:
        pass


def create_workflow(
    name: str = "",
    nodes: list[dict[str, Any]] | None = None,
    connections: dict[str, Any] | None = None,
    active: bool = True,
) -> Workflow:
    """Convenience factory for creating a Workflow."""
    return Workflow(
        name=name,
        nodes=nodes,
        connections=connections,
        active=active,
    )
