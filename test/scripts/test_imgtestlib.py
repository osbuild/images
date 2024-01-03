import os

from imgtestlib import rng_seed_env, runcmd


def test_runcmd():
    stdout, stderr = runcmd(["/bin/echo", "hello"])
    assert stdout == b"hello\n"
    assert stderr == b""


def test_runcmd_env():
    os.environ["RUNCMD_GLOBAL_TEST_VAR"] = "global test value"
    stdout, stderr = runcmd(["env"], extra_env={"RUNCMD_TEST_VAR": "the test value"})
    assert b"RUNCMD_TEST_VAR=the test value\n" in stdout, "extra env var not set"
    assert b"RUNCMD_GLOBAL_TEST_VAR=global test value\n" in stdout, "global env vars not preserved"
    assert stderr == b""


def test_read_seed():
    # check that it's read without error - no need to test the value itself
    seed_env = rng_seed_env()
    assert "OSBUILD_TESTING_RNG_SEED" in seed_env
