"""m9m Python SDK — High-performance workflow automation engine."""

from .binary import get_binary_path, download_binary
from .data import DataItem, ExecutionResult
from .engine import WorkflowEngine
from .workflow import Workflow, create_workflow


def version() -> str:
    """Return the m9m binary version."""
    import subprocess

    binary = get_binary_path()
    try:
        result = subprocess.run(
            [binary, "version"],
            capture_output=True,
            text=True,
            check=True,
        )
        return result.stdout.strip()
    except (subprocess.CalledProcessError, FileNotFoundError):
        return "unknown"


__all__ = [
    "DataItem",
    "ExecutionResult",
    "Workflow",
    "WorkflowEngine",
    "create_workflow",
    "version",
    "get_binary_path",
    "download_binary",
]
