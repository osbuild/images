import argparse
import json
import os
import pathlib
import subprocess as sp
import sys
from glob import glob
from typing import Dict

TEST_CACHE_ROOT = ".cache/osbuild-images"
CONFIGS_PATH = "./test/configs"
CONFIG_MAP = "./test/config-map.json"

S3_BUCKET = "s3://" + os.environ.get("AWS_BUCKET", "images-ci-cache")
S3_PREFIX = "images/builds"

REGISTRY = "registry.gitlab.com/redhat/services/products/image-builder/ci/images"

SCHUTZFILE = "Schutzfile"
OS_RELEASE_FILE = "/etc/os-release"

# image types that can be boot tested
CAN_BOOT_TEST = [
    "ami",
    "ec2",
    "ec2-ha",
    "ec2-sap",
    "edge-ami",
    "iot-bootable-container",
]

BIB_TYPES = [
    "iot-bootable-container"
]


# base and terraform bits copied from main .gitlab-ci.yml
# needed for status reporting and defining the runners
BASE_CONFIG = """
.base:
  before_script:
    - cat schutzbot/team_ssh_keys.txt |
        tee -a ~/.ssh/authorized_keys > /dev/null
  interruptible: true
  retry: 1
  tags:
    - terraform
  variables:
    PYTHONUNBUFFERED: 1

.terraform:
  extends: .base
  tags:
    - terraform
"""

NULL_CONFIG = """
NullBuild:
  stage: test
  script: "true"
  tags:
    - "shell"
"""


def runcmd(cmd, stdin=None, extra_env=None, capture_output=True):
    """
    Run the cmd using sp.run() and exit with the returncode if it is non-zero.
    Output is captured and both stdout and stderr are returned if the run succeeds.
    If it fails, the output is printed before exiting.
    """
    env = None
    if extra_env:
        env = os.environ
        env.update(extra_env)
    job = sp.run(cmd, input=stdin, capture_output=capture_output, env=env, check=False)
    if job.returncode > 0:
        print(f"❌ Command failed: {cmd}")
        if job.stdout:
            print(job.stdout.decode())
        if job.stderr:
            print(job.stderr.decode())
        sys.exit(job.returncode)

    return job.stdout, job.stderr


def runcmd_nc(cmd, stdin=None, extra_env=None):
    """
    Run the cmd using sp.run() and exit with the returncode if it is non-zero.
    Output it not captured.
    """
    runcmd(cmd, stdin=stdin, extra_env=extra_env, capture_output=False)


def list_images(distros=None, arches=None, images=None):
    distros_arg = "*"
    if distros:
        distros_arg = ",".join(distros)
    arches_arg = "*"
    if arches:
        arches_arg = ",".join(arches)
    images_arg = "*"
    if images:
        images_arg = ",".join(images)
    out, _ = runcmd(["go", "run", "./cmd/list-images", "--json",
                     "--distros", distros_arg, "--arches", arches_arg, "--types", images_arg])
    return json.loads(out)


def dl_build_info(destination, distro=None, arch=None, osbuild_ref=None, runner_distro=None):
    """
    Downloads all the configs from the s3 bucket.
    """
    s3url = gen_build_info_s3_dir_path(distro, arch, osbuild_ref=osbuild_ref, runner_distro=runner_distro)
    print(f"⬇️ Downloading configs from {s3url}")
    # only download info.json (exclude everything, then include) files, otherwise we get manifests and whole images
    job = sp.run(["aws", "s3", "sync",
                  "--no-progress",  # wont show progress but will print file list
                  "--exclude=*",
                  "--include=*/info.json",
                  "--include=*/bib-*",
                  s3url, destination],
                 capture_output=True,
                 check=False)
    ok = job.returncode == 0
    if not ok:
        print(f"⚠️ Failed to sync contents of {s3url}:")
        print(job.stdout.decode())
        print(job.stderr.decode())
    return job.stdout.decode(), ok


def get_manifest_id(manifest_data):
    md = json.dumps(manifest_data).encode()
    out, _ = runcmd(["osbuild", "--inspect", "-"], stdin=md)
    data = json.loads(out)
    # last stage ID depends on all previous stage IDs, so we can use it as a manifest ID
    return data["pipelines"][-1]["stages"][-1]["id"]


def _u(s):
    return s.replace("-", "_")


def gen_build_name(distro, arch, image_type, config_name):
    return f"{_u(distro)}-{_u(arch)}-{_u(image_type)}-{_u(config_name)}"


def gen_build_info_dir_path_prefix(distro=None, arch=None, manifest_id=None, osbuild_ref=None, runner_distro=None):
    """
    Generates the relative path prefix for the location where build info and artifacts will be stored for a specific
    build. This is a simple concatenation of the components, but ensures that paths are consistent. The caller is
    responsible for prepending the location root to the generated path.

    If no 'osbuild_ref' is specified, the value returned by get_osbuild_commit() for the 'runner_distro' will be used.
    if no 'runner_distro' is specified, the value returned by get_host_distro() will be used.

    A fully specified path is returned if all of the 'distro', 'arch' and 'manifest_id' parameters are specified,
    otherwise a partial path is returned. Partial path may be useful for working with a superset of build infos.
    For a more specific path to be generated when specifying any of the optional parameters, the caller must specify
    all of the previous parameters. For example, if 'arch' is specified, 'distro' must also be specified for 'arch' to
    be included in the path.

    The returned path always has a trailing separator at the end to signal that it is a directory.
    """
    if runner_distro is None:
        runner_distro = get_host_distro()
    if osbuild_ref is None:
        osbuild_ref = get_osbuild_commit(runner_distro)

    path = os.path.join(f"osbuild-ref-{osbuild_ref}", f"runner-{runner_distro}")
    for p in (distro, arch, f"manifest-id-{manifest_id}" if manifest_id else None):
        if p is None:
            return path + "/"
        path = os.path.join(path, p)
    return path + "/"


def gen_build_info_s3_dir_path(distro=None, arch=None, manifest_id=None, osbuild_ref=None, runner_distro=None):
    """
    Generates the s3 URL for the location where build info and artifacts will be stored for a specific
    one or more builds, depending on the parameters specified.

    A fully specified path is returned if all parameters are specified, otherwise a partial path is returned.
    This function basically just prepends the S3_BUCKET and S3_PREFIX to the path generated by
    gen_build_info_dir_path_prefix().
    """
    return os.path.join(
        S3_BUCKET,
        S3_PREFIX,
        gen_build_info_dir_path_prefix(distro, arch, manifest_id, osbuild_ref, runner_distro),
    )


def check_config_names():
    """
    Check that all the configs we rely on have names that match the file name, otherwise the test skipping and pipeline
    generation will be incorrect.
    """
    bad_configs = []
    for file in pathlib.Path(CONFIGS_PATH).glob("*.json"):
        config = json.loads(file.read_text())
        if file.stem != config["name"]:
            bad_configs.append(str(file))

    if bad_configs:
        print("☠️ ERROR: The following test configs have names that don't match their filenames.")
        print("\n".join(bad_configs))
        print("This will produce incorrect test generation and results.")
        print("Aborting.")
        sys.exit(1)


def gen_manifests(outputdir, config_map=None, distros=None, arches=None, images=None,
                  commits=False, skip_no_config=False):
    # pylint: disable=too-many-arguments,too-many-positional-arguments
    cmd = ["go", "run", "./cmd/gen-manifests",
           "--cache", os.path.join(TEST_CACHE_ROOT, "rpmmd"),
           "--output", outputdir,
           "--workers", "100"]
    if config_map:
        cmd.extend(["--config-map", config_map])
    if distros:
        cmd.extend(["--distros", ",".join(distros)])
    if arches:
        cmd.extend(["--arches", ",".join(arches)])
    if images:
        cmd.extend(["--types", ",".join(images)])
    if commits:
        cmd.append("--commits")
    if skip_no_config:
        cmd.append("--skip-noconfig")
    print("⌨️" + " ".join(cmd))
    _, stderr = runcmd(cmd, extra_env=rng_seed_env())
    return stderr


def read_manifests(path):
    """
    Read all manifests in the given path, calculate their IDs, and return a dictionary mapping each filename to the data
    and its ID.
    """
    print(f"📖 Reading manifests in {path}")
    manifests = {}
    for manifest_fname in os.listdir(path):
        manifest_path = os.path.join(path, manifest_fname)
        with open(manifest_path, encoding="utf-8") as manifest_file:
            manifest_data = json.load(manifest_file)
        manifests[manifest_fname] = {
            "data": manifest_data,
            "id": get_manifest_id(manifest_data["manifest"]),
        }
    print("✅ Done")
    return manifests


def check_for_build(manifest_fname, build_info_dir, errors):
    build_info_path = os.path.join(build_info_dir, "info.json")
    # rebuild if matching build info is not found
    if not os.path.exists(build_info_path):
        print(f"Build info not found: {build_info_path}")
        print("  Adding config to build pipeline.")
        return True

    try:
        with open(build_info_path, encoding="utf-8") as build_info_fp:
            dl_config = json.load(build_info_fp)
    except json.JSONDecodeError as jd:
        errors.append((
            f"failed to parse {build_info_path}\n"
            f"{jd.msg}\n"
            "  Adding config to build pipeline.\n"
        ))

    commit = dl_config["commit"]
    pr = dl_config.get("pr")
    url = f"https://github.com/osbuild/images/commit/{commit}"
    print(f"🖼️ Manifest {manifest_fname} was successfully built in commit {commit}\n  {url}")
    if "gh-readonly-queue" in pr:
        print(f"  This commit was on a merge queue: {pr}")
    elif pr:
        print(f"  PR-{pr}: https://github.com/osbuild/images/pull/{pr}")
    else:
        print("  No PR/branch info available")

    image_type = dl_config["image-type"]
    if image_type not in CAN_BOOT_TEST:
        print(f"  Boot testing for {image_type} is not yet supported")
        return False

    # boot testing supported: check if it's been tested, otherwise queue it for rebuild and boot
    if dl_config.get("boot-success", False):
        print("  This image was successfully boot tested")

        # check if it's a BIB type and compare image IDs
        if image_type in BIB_TYPES:
            # Successful boot tests with BIB add a file to the directory as bib-<image ID>. Collect them and compare.
            bib_ids = glob("bib-*", root_dir=build_info_dir)
            # add the _old_ bib ID that we used to keep in the info.json
            config_bib_id = dl_config.get("bib-id")
            if config_bib_id:
                bib_ids.append(f"bib-{config_bib_id}")
            bib_ref = get_bib_ref()
            current_id = skopeo_inspect_id(f"docker://{bib_ref}", host_container_arch())
            if f"bib-{current_id}" not in bib_ids:
                if bib_ids:
                    print("  Container disk image was built with the following bootc-image-builder images:")
                    print("    - " + "\n    -".join(bib_ids))
                else:
                    print("  No bib IDs found.")
                print(f"  Testing {current_id}")
                print("  Adding config to build pipeline.")
                return True

        return False
    print("  Boot test success not found.")

    # default to build
    print("  Adding config to build pipeline.")
    return True


def filter_builds(manifests, distro=None, arch=None, skip_ostree_pull=True):
    """
    Returns a list of build requests for the manifests that have no matching config in the test build cache.
    """
    print(f"⚙️ Filtering {len(manifests)} build configurations")
    dl_root_path = os.path.join(TEST_CACHE_ROOT, "s3configs", "builds")
    dl_path = os.path.join(dl_root_path, gen_build_info_dir_path_prefix(distro, arch))
    os.makedirs(dl_path, exist_ok=True)
    build_requests = []

    out, dl_ok = dl_build_info(dl_path, distro, arch)
    # continue even if the dl failed; will build all configs
    if dl_ok:
        # print output which includes list of downloaded files for CI job log
        print(out)

    errors: list[str] = []
    for manifest_fname, data in manifests.items():
        manifest_id = data["id"]
        data = data.get("data")
        build_request = data["build-request"]
        distro = build_request["distro"]
        arch = build_request["arch"]
        image_type = build_request["image-type"]
        config = build_request["config"]
        config_name = config["name"]
        options = config.get("options", {})

        # check if the config specifies an ostree URL and skip it if requested
        if skip_ostree_pull and options.get("ostree", {}).get("url"):
            print(f"🦘 Skipping {distro}/{arch}/{image_type}/{config_name} (ostree dependency)")
            continue

        # add manifest id to build request
        build_request["manifest-checksum"] = manifest_id

        # check if the hash_fname exists in the synced directory
        build_info_dir = os.path.join(
            dl_root_path,
            gen_build_info_dir_path_prefix(distro, arch, manifest_id)
        )

        if check_for_build(manifest_fname, build_info_dir, errors):
            build_requests.append(build_request)

    print("✅ Config filtering done!\n")
    if errors:
        # print errors at the end so they're visible
        print("⚠️ Errors:")
        print("\n".join(errors))

    return build_requests


def clargs():
    default_arch = os.uname().machine
    parser = argparse.ArgumentParser()
    parser.add_argument("config", type=str, help="path to write config")
    parser.add_argument("--distro", type=str, required=True,
                        help="distro to generate configs for")
    parser.add_argument("--arch", type=str, default=default_arch,
                        help="architecture to generate configs for (defaults to host architecture)")

    return parser


def read_osrelease():
    """Read Operating System Information from `os-release`

    This creates a dictionary with information describing the running operating system. It reads the information from
    the path array provided as `paths`.  The first available file takes precedence. It must be formatted according to
    the rules in `os-release(5)`.
    """
    osrelease = {}

    with open(OS_RELEASE_FILE, encoding="utf8") as orf:
        for line in orf:
            line = line.strip()
            if not line:
                continue
            if line[0] == "#":
                continue
            key, value = line.split("=", 1)
            osrelease[key] = value.strip('"')

    return osrelease


def get_host_distro():
    """
    Get the host distro version based on data in the os-release file.
    The format is <distro>-<version> (e.g. fedora-41).
    """
    osrelease = read_osrelease()
    return f"{osrelease['ID']}-{osrelease['VERSION_ID']}"


def get_osbuild_commit(distro_version):
    """
    Get the osbuild commit defined in the Schutzfile for the host distro.
    If not set, returns None.
    """
    with open(SCHUTZFILE, encoding="utf-8") as schutzfile:
        data = json.load(schutzfile)

    return data.get(distro_version, {}).get("dependencies", {}).get("osbuild", {}).get("commit", None)


def get_bib_ref():
    """
    Get the bootc-image-builder ref defined in the Schutzfile for the host distro.
    If not set, returns None.
    """
    with open(SCHUTZFILE, encoding="utf-8") as schutzfile:
        data = json.load(schutzfile)

    return data.get("common", {}).get("bootc-image-builder", {}).get("ref", None)


def rng_seed_env():
    """
    Read the rng seed from the Schutzfile and return it as a map to use as an environment variable with the appropriate
    key. Assumes the file exists and that it contains the key 'rngseed', otherwise raises an exception.
    """

    with open(SCHUTZFILE, encoding="utf-8") as schutzfile:
        data = json.load(schutzfile)

    seed = data.get("common", {}).get("rngseed")
    if seed is None:
        raise RuntimeError("'common.rngseed' not found in Schutzfile")

    return {"OSBUILD_TESTING_RNG_SEED": str(seed)}


def host_container_arch():
    host_arch = os.uname().machine
    return {
        "x86_64": "amd64",
        "aarch64": "arm64"
    }.get(host_arch, host_arch)


def is_manifest_list(data):
    """Inspect a manifest determine if it's a multi-image manifest-list."""
    media_type = data.get("mediaType")
    #  Check if mediaType is set according to docker or oci specifications
    if media_type in ("application/vnd.docker.distribution.manifest.list.v2+json",
                      "application/vnd.oci.image.index.v1+json"):
        return True

    # According to the OCI spec, setting mediaType is not mandatory. So, if it is not set at all, check for the
    # existence of manifests
    if media_type is None and data.get("manifests") is not None:
        return True

    return False


def skopeo_inspect_id(image_name: str, arch: str) -> str:
    """
    Returns the image ID (config digest) of the container image. If the image resolves to a manifest list, the config
    digest of the given architecture is resolved.

    Runs with 'sudo' when inspecting a local container because in our tests we need to read the root container storage.
    """
    cmd = ["skopeo", "inspect", "--raw", image_name]
    if image_name.startswith("containers-storage"):
        cmd = ["sudo"] + cmd
    out, _ = runcmd(cmd)
    data = json.loads(out)
    if not is_manifest_list(data):
        return data["config"]["digest"]

    for manifest in data.get("manifests", []):
        platform = manifest.get("platform", {})
        img_arch = platform.get("architecture", "")
        img_ostype = platform.get("os", "")

        if arch != img_arch or img_ostype != "linux":
            continue

        if "@" in image_name:
            image_no_tag = image_name.split("@")[0]
        else:
            image_no_tag = ":".join(image_name.split(":")[:-1])
        manifest_digest = manifest["digest"]
        arch_image_name = f"{image_no_tag}@{manifest_digest}"
        # inspect the arch-specific manifest to get the image ID (config digest)
        return skopeo_inspect_id(arch_image_name, arch)

    # don't error out, just return an empty string and let the caller handle it
    return ""


def get_common_ci_runner():
    """
    CI runner for common tasks.

    Currently this is used for all gitlab CI jobs. In the future, we might switch to running build jobs on the same host
    distro as the target image, but this CI runner will still be used for generic tasks like check-build-coverage.
    """
    with open(SCHUTZFILE, encoding="utf-8") as schutzfile:
        data = json.load(schutzfile)

    if (runner := data.get("common", {}).get("gitlab-ci-runner")) is None:
        raise KeyError(f"gitlab-ci-runner not defined in {SCHUTZFILE}")

    return runner


def get_common_ci_runner_distro():
    """
    CI runner distro for common tasks.

    Returns the distro part from the value of the common.gitlab-ci-runner key in the Schutzfile.
    For example, if the value is "aws/fedora-999", this function will return "fedora-999".
    """
    return get_common_ci_runner().split("/")[1]


def find_image_file(build_path: str) -> str:
    """
    Find the path to the image by reading the manifest to get the name of the last pipeline and searching for the file
    under the directory named after the pipeline. Raises RuntimeError if no or multiple files are found in the expected
    path.
    """
    manifest_file = os.path.join(build_path, "manifest.json")
    with open(manifest_file, encoding="utf-8") as manifest:
        data = json.load(manifest)

    last_pipeline = data["pipelines"][-1]["name"]
    files = os.listdir(os.path.join(build_path, last_pipeline))
    if len(files) > 1:
        error = "Multiple files found in build path while searching for image file"
        error += "\n".join(files)
        raise RuntimeError(error)

    if len(files) == 0:
        raise RuntimeError("No found in build path while searching for image file")

    return os.path.join(build_path, last_pipeline, files[0])


def read_build_info(build_path: str) -> Dict:
    """
    Read the info.json file from the build directory and return the data as a dictionary.
    """
    info_file_path = os.path.join(build_path, "info.json")
    with open(info_file_path, encoding="utf-8") as info_fp:
        return json.load(info_fp)


def write_build_info(build_path: str, data: Dict):
    """
    Write the data to the info.json file in the build directory.
    """
    info_file_path = os.path.join(build_path, "info.json")
    with open(info_file_path, "w", encoding="utf-8") as info_fp:
        json.dump(data, info_fp, indent=2)
