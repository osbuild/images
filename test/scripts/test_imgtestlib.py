import os

import imgtestlib as testlib


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
