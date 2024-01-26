import os
import subprocess as sp
import tempfile

import pytest

import imgtestlib as testlib

TEST_ARCHES = ["amd64", "arm64"]


def can_sudo_nopw() -> bool:
    """
    Check if we can run sudo without a password.
    """
    job = sp.run(["sudo", "-n", "true"], capture_output=True, check=False)
    return job.returncode == 0


def test_runcmd():
    stdout, stderr = testlib.runcmd(["/bin/echo", "hello"])
    assert stdout == b"hello\n"
    assert stderr == b""


def test_runcmd_env():
    os.environ["RUNCMD_GLOBAL_TEST_VAR"] = "global test value"
    stdout, stderr = testlib.runcmd(["env"], extra_env={"RUNCMD_TEST_VAR": "the test value"})
    assert b"RUNCMD_TEST_VAR=the test value\n" in stdout, "extra env var not set"
    assert b"RUNCMD_GLOBAL_TEST_VAR=global test value\n" in stdout, "global env vars not preserved"
    assert stderr == b""


def test_read_seed():
    # check that it's read without error - no need to test the value itself
    seed_env = testlib.rng_seed_env()
    assert "OSBUILD_TESTING_RNG_SEED" in seed_env


def test_path_generators():
    testlib.get_osbuild_nevra = lambda: "osbuild-104-1.fc39.noarch"

    assert testlib.gen_build_info_dir_path("inforoot", testlib.get_osbuild_nevra(), "abc123") == \
        "inforoot/osbuild-104-1.fc39.noarch/abc123/"
    assert testlib.gen_build_info_path("inforoot", testlib.get_osbuild_nevra(), "abc123") == \
        "inforoot/osbuild-104-1.fc39.noarch/abc123/info.json"
    assert testlib.gen_build_info_s3("fedora-39", "aarch64", "abc123") == \
        testlib.S3_BUCKET + "/images/builds/fedora-39/aarch64/osbuild-104-1.fc39.noarch/abc123/"


test_container = "registry.gitlab.com/redhat/services/products/image-builder/ci/osbuild-composer/manifest-list-test"

# manifest IDs for
#  registry.gitlab.com/redhat/services/products/image-builder/ci/osbuild-composer/manifest-list-test:latest
manifest_ids = {
    "amd64": "sha256:601c98c8148720ec5c29b8e854a1d5d88faddbc443eca12920d76cf993d7290e",
    "arm64": "sha256:1a19a94647b1379fed8c23eb7553327cb604ba546eb93f9f6c1e6d11911c8beb",
}

# image IDs for
#  registry.gitlab.com/redhat/services/products/image-builder/ci/osbuild-composer/manifest-list-test:latest
image_ids = {
    "amd64": "sha256:dbb63178dc9157068107961f11397df3fb62c02fa64f697d571bf84aad71cb99",
    "arm64": "sha256:62d2a7b3bf9e0b4f3aba22553d6971227b5a39f7f408d46347b1ee74eb97cb20",
}


@pytest.mark.parametrize("arch", TEST_ARCHES)
def test_skopeo_inspect_id_manifest_list(arch):
    transport = "docker://"
    image_id = image_ids[arch]
    assert testlib.skopeo_inspect_id(f"{transport}{test_container}:latest", arch) == image_id


@pytest.mark.parametrize("arch", TEST_ARCHES)
def test_skopeo_inspect_image_manifest(arch):
    transport = "docker://"
    manifest_id = manifest_ids[arch]
    image_id = image_ids[arch]
    # arch arg to skopeo_inspect_id doesn't matter here
    assert testlib.skopeo_inspect_id(f"{transport}{test_container}@{manifest_id}", arch) == image_id


@pytest.mark.skipif(not can_sudo_nopw(), reason="requires passwordless sudo")
@pytest.mark.parametrize("arch", TEST_ARCHES)
def test_skopeo_inspect_localstore(arch):
    transport = "containers-storage:"
    image = "registry.gitlab.com/redhat/services/products/image-builder/ci/osbuild-composer/manifest-list-test:latest"
    with tempfile.TemporaryDirectory() as tmpdir:
        testlib.runcmd(["sudo", "podman", "pull", f"--arch={arch}", "--storage-driver=vfs", f"--root={tmpdir}", image])

        # arch arg to skopeo_inspect_id doesn't matter here
        assert testlib.skopeo_inspect_id(f"{transport}[vfs@{tmpdir}]{image}", arch) == image_ids[arch]
