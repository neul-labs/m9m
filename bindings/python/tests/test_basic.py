"""
Basic tests for m9m Python bindings.
"""

import json
import os
import sys
from pathlib import Path

import pytest

# Add the package to the path for testing
sys.path.insert(0, str(Path(__file__).parent.parent))

from m9m import (
    DataItem,
    ExecutionResult,
    Workflow,
    WorkflowEngine,
    create_workflow,
    version,
)


class TestVersion:
    """Test version function."""

    def test_version_returns_string(self):
        v = version()
        assert isinstance(v, str)
        assert len(v) > 0


class TestWorkflowEngine:
    """Test WorkflowEngine class."""

    def test_create_engine(self):
        engine = WorkflowEngine()
        assert engine is not None

    def test_execute_empty_workflow(self):
        engine = WorkflowEngine()
        workflow = create_workflow(name="test", nodes=[], connections={})
        result = engine.execute(workflow)
        assert result is not None
        assert result.success

    def test_execute_with_input(self):
        engine = WorkflowEngine()
        workflow = create_workflow(name="test", nodes=[], connections={})
        input_data = [{"json": {"key": "value"}}]
        result = engine.execute(workflow, input_data)
        assert result is not None
        assert result.success

    def test_context_manager(self):
        with WorkflowEngine() as engine:
            workflow = create_workflow(name="test")
            result = engine.execute(workflow)
            assert result.success


class TestWorkflow:
    """Test Workflow class."""

    def test_create_workflow(self):
        workflow = create_workflow(name="Test Workflow")
        assert workflow is not None
        assert workflow.name == "Test Workflow"

    def test_workflow_from_json(self):
        json_str = json.dumps(
            {
                "id": "test-123",
                "name": "JSON Workflow",
                "active": True,
                "nodes": [],
                "connections": {},
            }
        )
        workflow = Workflow.from_json(json_str)
        assert workflow.id == "test-123"
        assert workflow.name == "JSON Workflow"
        assert workflow.active

    def test_workflow_to_json(self):
        workflow = create_workflow(name="Test", active=False)
        json_str = workflow.to_json()
        data = json.loads(json_str)
        assert data["name"] == "Test"
        assert data["active"] is False

    def test_workflow_context_manager(self):
        with Workflow.from_json('{"name": "test", "nodes": [], "connections": {}}') as wf:
            assert wf.name == "test"


class TestDataItem:
    """Test DataItem class."""

    def test_create_data_item(self):
        item = DataItem(json={"key": "value"})
        assert item.json["key"] == "value"

    def test_data_item_to_dict(self):
        item = DataItem(json={"a": 1}, binary={"file": {"data": "abc"}})
        d = item.to_dict()
        assert d["json"]["a"] == 1
        assert d["binary"]["file"]["data"] == "abc"

    def test_data_item_from_dict(self):
        item = DataItem.from_dict({"json": {"x": 2}})
        assert item.json["x"] == 2


class TestExecutionResult:
    """Test ExecutionResult class."""

    def test_success_result(self):
        result = ExecutionResult(data=[DataItem(json={"result": "ok"})])
        assert result.success
        assert result.error is None
        assert len(result.data) == 1

    def test_error_result(self):
        result = ExecutionResult(data=[], error="Something went wrong")
        assert not result.success
        assert result.error == "Something went wrong"

    def test_from_json(self):
        json_str = json.dumps({"data": [{"json": {"test": 123}}], "error": None})
        result = ExecutionResult.from_json(json_str)
        assert result.success
        assert len(result.data) == 1
        assert result.data[0].json["test"] == 123


class TestCustomNodes:
    """Test custom node registration."""

    def test_register_node(self):
        engine = WorkflowEngine()

        def my_node(input_data, params):
            return [{"json": {"processed": True}}]

        engine.register_node("custom.test", my_node)
        # Registration should not raise

    def test_node_decorator(self):
        engine = WorkflowEngine()

        @engine.node("custom.decorated")
        def decorated_node(input_data, params):
            return input_data

        # Decorator should not raise


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
