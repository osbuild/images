import json
import os
import pathlib
import subprocess
import textwrap

import pytest

def top_srcdir() -> pathlib.Path:
    d = pathlib.Path(__file__).parent
    while not (d / "cmd").exists():
        d = d.parent
        if d == "/":
            raise RuntimeError("cannot find cmd dir")
    return d

def test_manifest_unchanged(tmp_path):
    manifests_ref = top_srcdir() / "test/data/manifestdb-light"
    manifests_new = tmp_path / "new"

    # TODO: omit once we have a "riscv64" mirror and sources entry
    arches = ["x86_64", "aarch64", "ppc64el", "s390x"]
    # Only run a subset of the distros for now. All manifests are
    # about 62Mb, (gzip) compressed 9Mb and even generating them
    # all ~1000 just takes less than 1min so it seems feasiable
    # to eventually do it
    distros = ["centos-10", "rhel-10.0"]
    
    env = os.environ.copy()
    env["OSBUILD_TESTING_RNG_SEED"] = "0"
    subprocess.run([
        "go", "run",
        os.fspath(top_srcdir() / "cmd/gen-manifests"),
        "-packages=false",
        "-metadata=false",
        "-containers=false",
        "-arches", ",".join(arches),
        "-distros", ",".join(distros),
        "-output", manifests_new,
    ], env=env, check=True)

    ret = subprocess.run([
        "diff", "-uNr", manifests_ref, manifests_new,
    ], capture_output=True, text=True, check=False)
    if ret.returncode != 0:
        msg = textwrap.dedent(f"""\
        unexpected difference between {manifests_new} and reference manifests:
        ---
        {ret.stdout}
        ---
        if this is expected, please run:
        cp -a {manifests_new}/* {manifests_ref}
        """)
        pytest.fail(msg)
        
