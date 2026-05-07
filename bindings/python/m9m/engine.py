"""WorkflowEngine wrapper for the m9m Python SDK."""

from __future__ import annotations

import json as _json
import subprocess
import tempfile
from typing import Any

from .binary import get_binary_path
from .data import DataItem, ExecutionResult
from .workflow import Workflow


class WorkflowEngine:
    """Spawns the m9m binary to execute workflows."""

    def __init__(self, credential_manager: Any | None = None):
        self._binary = get_binary_path()
        self._credential_manager = credential_manager
        self._custom_nodes: dict[str, Any] = {}

    def execute(
        self,
        workflow: Workflow,
        input_data: list[dict[str, Any]] | None = None,
    ) -> ExecutionResult:
        """Execute a workflow with optional input data."""
        # Write workflow to a temp file
        with tempfile.NamedTemporaryFile(mode="w", suffix=".json", delete=False) as f:
            f.write(workflow.to_json())
            workflow_path = f.name

        input_data = input_data or []
        input_json = _json.dumps(input_data)

        try:
            result = subprocess.run(
                [self._binary, "exec", workflow_path, "--input", input_json],
                capture_output=True,
                text=True,
                check=False,
            )
            if result.returncode != 0:
                return ExecutionResult(error=result.stderr.strip() or "execution failed")
            # Try to parse JSON output
            try:
                return ExecutionResult.from_json(result.stdout)
            except Exception:
                # If output is not JSON, wrap it
                return ExecutionResult(
                    data=[DataItem(json={"output": result.stdout})]
                )
        except FileNotFoundError:
            return ExecutionResult(error="m9m binary not found")
        finally:
            import os

            try:
                os.unlink(workflow_path)
            except OSError:
                pass

    def register_node(self, name: str, handler: Any) -> None:
        """Register a custom node handler (placeholder — requires server mode)."""
        self._custom_nodes[name] = handler

    def node(self, name: str, **kwargs: Any) -> Any:
        """Decorator for registering custom nodes."""

        def decorator(func: Any) -> Any:
            self.register_node(name, func)
            return func

        return decorator

    def __enter__(self) -> "WorkflowEngine":
        return self

    def __exit__(self, *args: Any) -> None:
        pass
