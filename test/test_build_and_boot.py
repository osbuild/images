import platform
import subprocess

import pytest
import scripts.imgtestlib as testlib

def load_test_cases():
    return subprocess.check_output([
        "go", "run", "./cmd/gen-manifests",
        # we may consider cross arch tests here at some point but for now
        # assume we run native
        "-arches", platform.uname().machine,
        "-print-only",
    ], text=True).strip().split("\n")


@pytest.mark.parametrize("distro,arch,img_type,config_name", [tcase.split(",") for tcase in load_test_cases()])
def test_build_boot_image(distro,arch,img_type,config_name):
    subprocess.check_call(
        ["./test/scripts/build-image", distro, img_type, f"test/configs/{config_name}.json"])
    build_dir = os.path.join("build", testlib.gen_build_name(distro, arch, image_type, config_name))
    subprocess.check_call(
        ["./test/scripts/boot-image", build_dir])
