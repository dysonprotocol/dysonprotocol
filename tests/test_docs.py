"""
Test module for validating that all Jupyter notebooks in the docs directory run without errors.

This module uses nbconvert to execute notebooks programmatically and pytest to assert
that they complete successfully without raising exceptions.
"""

import os
import glob
import json
import pytest
import subprocess
import tempfile
from pathlib import Path


class NotebookExecutionError(Exception):
    """Custom exception for notebook execution failures."""
    pass


def get_notebook_files():
    """
    Get all Jupyter notebook files from the docs directory.
    
    Returns:
        list: List of paths to .ipynb files
    """
    docs_dir = Path(__file__).parent.parent / "docs"
    notebook_files = list(docs_dir.glob("*.ipynb"))
    
    # Filter out checkpoint files
    notebook_files = [
        nb for nb in notebook_files 
        if ".ipynb_checkpoints" not in str(nb)
    ]
    
    return notebook_files


def execute_notebook(notebook_path, dyson_home):
    """
    Execute a Jupyter notebook using nbconvert.
    
    Args:
        notebook_path (Path): Path to the notebook file
        
    Returns:
        tuple: (success: bool, output: str, error: str)
    """
    try:
        # Execute the notebook using nbconvert in place
        cmd = [
            "jupyter", "nbconvert",
            "--to", "notebook",
            "--execute",
            "--inplace",
            "--ExecutePreprocessor.timeout=5",  # DO NOT CHANGE THIS, INSTEAD FIX YOUR TESTS!!!!
            "--ExecutePreprocessor.kernel_name=python3",
            str(notebook_path)
        ]

        print(f"Executing notebook: {' '.join(cmd)}")
        
        result = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            cwd=notebook_path.parent,
            env=os.environ.copy()
        )
        
        if result.returncode == 0:
            return True, result.stdout, ""
        else:
            return False, result.stdout, result.stderr
            
    except Exception as e:
        return False, "", str(e)


def validate_notebook_structure(notebook_path):
    """
    Validate that the notebook file has valid JSON structure.
    
    Args:
        notebook_path (Path): Path to the notebook file
        
    Returns:
        bool: True if valid, False otherwise
    """
    try:
        with open(notebook_path, 'r', encoding='utf-8') as f:
            notebook_data = json.load(f)
            
        # Basic validation - check for required fields
        required_fields = ['cells', 'metadata', 'nbformat']
        for field in required_fields:
            if field not in notebook_data:
                return False
                
        return True
        
    except (json.JSONDecodeError, FileNotFoundError, UnicodeDecodeError):
        return False


def pytest_generate_tests(metafunc):
    """
    Dynamically generate tests for each notebook file.
    
    This creates individual named tests for each notebook, allowing for:
    - pytest --ff (fail fast) to work properly
    - Running specific notebook tests by name
    - Better test reporting with individual notebook names
    """
    if "notebook_path" in metafunc.fixturenames:
        notebook_files = get_notebook_files()
        
        # Create test IDs using notebook names (without .ipynb extension)
        test_ids = [nb.stem for nb in notebook_files]
        
        metafunc.parametrize(
            "notebook_path", 
            notebook_files, 
            ids=test_ids
        )


@pytest.mark.docs
def test_notebook_structure(notebook_path):
    """
    Test that each notebook has valid JSON structure.
    
    Args:
        notebook_path (Path): Path to the notebook file
    """
    assert validate_notebook_structure(notebook_path), f"Invalid notebook structure: {notebook_path}"


@pytest.mark.docs
def test_notebook_execution(notebook_path, chainnet):
    """
    Test that each notebook executes without errors.
    
    Args:
        notebook_path (Path): Path to the notebook file
        chainnet: Chainnet fixture providing access to test chains
        test_config_path: Path to the chainnet configuration file
    """
    # Skip execution if jupyter/nbconvert is not available
    result = subprocess.run(["jupyter", "--version"], capture_output=True)
    assert result.returncode == 0, "Jupyter not available - cannot execute notebooks"
    
    # Get the first chain's run command
    dysond_bin = chainnet[0]
    
    # Get the home directory of the first node of the first chain
    dyson_home = dysond_bin("config", "home").strip()
    print(f"Dyson home: {dyson_home}")
    
    # Set up environment variables for notebook execution
    #"DYSON_HOME": dyson_home,
    
    os.environ["DYSON_HOME"] = dyson_home
    
    # Execute the notebook with the environment variables
    success, stdout, stderr = execute_notebook(notebook_path, dyson_home)
    
    assert success, f"Notebook execution failed: {notebook_path}\nSTDOUT:\n{stdout}\nSTDERR:\n{stderr}"


@pytest.mark.docs
def test_notebooks_exist():
    """Test that there are actually notebook files to test."""
    notebook_files = get_notebook_files()
    assert len(notebook_files) > 0, "No notebook files found in docs directory"

