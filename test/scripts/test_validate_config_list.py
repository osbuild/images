import importlib
from types import ModuleType

import pytest


def import_validator() -> ModuleType:
    name = "validate_config_list"
    path = "test/scripts/validate-config-list"
    loader = importlib.machinery.SourceFileLoader(name, path)
    spec = importlib.util.spec_from_loader(loader.name, loader)
    if spec is None:
        raise ImportError(f"cannot import {name} from {path}, got None as the spec")
    mod = importlib.util.module_from_spec(spec)
    loader.exec_module(mod)
    return mod


def write_configs(files, root):
    for config_file in files:
        config_path = root / config_file
        config_path.parent.mkdir(exist_ok=True)
        config_path.touch()


@pytest.mark.parametrize("config_list,files,missing_files,invalid_cfgs", (
    # valid
    (
        [{
            "path": "everything.json",
            "filters": {
                "distros": [
                    "fedora*",
                ],
            },
            "empty.json": {
                "image-types": ["qcow2"],
            },
        }],
        ["everything.json", "empty.json"],
        [],
        [],
    ),
    (
        [
            {"path": "configs/cfg-1.json", "filters": {}},
            {
                "path": "configs/cfg-2.json",
                "filters": {
                    "distros": ["centos*"],
                    "arches": ["s390x"],
                    "image-types": ["qcow2"],
                },
            },
        ],
        ["configs/cfg-1.json", "configs/cfg-2.json"],
        [],
        [],
    ),
    (
        [
            {
                "path": "configs/cfg-3.json",
                "filters": {
                    "distros": ["fedora*"],
                },
            },
            {
                "path": "configs/cfg-4.json",
                "filters": {
                    "image-types": ["qcow2"],
                },
            }
        ],
        ["configs/cfg-3.json", "configs/cfg-4.json"],
        [],
        [],
    ),

    # missing files
    (
        [
            {
                "path": "everything.json",
                "filters": {
                    "distros": [
                        "fedora*",
                    ],
                },
            },
            {
                "path": "empty.json",
                "filters": {
                    "image-types": ["qcow2"],
                },
            }
        ],
        ["everything.json"],
        ["empty.json"],
        [],
    ),
    (
        [
            {
                "path": "configs/cfg-1.json", "filters": {},
            },
            {
                "path": "configs/cfg-2.json",
                "filters": {
                    "distros": ["centos*"],
                    "arches": ["s390x"],
                    "image-types": ["qcow2"],
                },
            },
        ],
        [],
        ["configs/cfg-1.json", "configs/cfg-2.json"],
        [],
    ),
    (
        [
            {
                "path": "configs/cfg-3.json",
                "filters": {
                    "distros": ["fedora*"],
                },
            },
            {
                "path": "configs/cfg-4.json",
                "filters": {
                    "image-types": ["qcow2"],
                },
            },
        ],
        ["configs/cfg-4.json"],
        ["configs/cfg-3.json"],
        [],
    ),

    # bad config
    (
        [
            {
                "path": "everything.json",
                "filters": {
                    "distros": [
                        "fedora*",
                    ],
                },
            },
            {
                "path": "empty.json",
                "filters": {
                    "image-types": ["not-qcow2"],
                },
            },
        ],
        ["everything.json", "empty.json"],
        [],
        [
            (
                "empty.json",
                {
                    "image-types": ["not-qcow2"],
                },
            )
        ],
    ),
    (
        [
            {
                "path": "configs/cfg-1.json", "filters": {},
            },
            {
                "path": "configs/cfg-2.json",
                "filters": {
                    "distros": ["centos*"],
                    "arches": ["noarch"],
                    "image-types": ["qcow2"],
                },
            },
        ],
        ["configs/cfg-1.json", "configs/cfg-2.json"],
        [],
        [
            (
                "configs/cfg-2.json",
                {
                    "distros": ["centos*"],
                    "arches": ["noarch"],
                    "image-types": ["qcow2"],
                },
            )
        ],
    ),
    (
        [
            {
                "path": "configs/cfg-3.json",
                "filters": {
                    "distros": ["archlinux"],
                },
            },
            {
                "path": "configs/cfg-4.json",
                "filters": {
                    "distros": ["ubuntu*"],
                },
            },
        ],
        ["configs/cfg-3.json", "configs/cfg-4.json"],
        [],
        [
            (
                "configs/cfg-3.json",
                {
                    "distros": ["archlinux"],
                },
            ),
            (
                "configs/cfg-4.json",
                {
                    "distros": ["ubuntu*"],
                },
            ),
        ],
    ),
))
def test_valid_config_list(config_list, files, missing_files, invalid_cfgs, tmp_path):
    validator = import_validator()
    write_configs(files, tmp_path)

    assert validator.validate_config_file_paths(config_list, tmp_path) == [tmp_path / mf for mf in missing_files]
    assert validator.validate_build_config(config_list) == invalid_cfgs
