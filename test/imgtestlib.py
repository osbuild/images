import json
import os
import pathlib
import subprocess as sp
import sys

TEST_CACHE_ROOT = os.path.expanduser("~/.cache/osbuild-images")
CONFIGS_PATH = "./test/configs"
CONFIG_MAP = "./test/config-map.json"

S3_BUCKET = "s3://image-builder-ci-artifacts"
S3_PREFIX = "images/builds"

REGISTRY = "registry.gitlab.com/redhat/services/products/image-builder/ci/images"


# ostree containers are pushed to the CI registry to be reused by dependants
OSTREE_CONTAINERS = [
    "iot-container",
    "edge-container"
]


# base and terraform bits copied from main .gitlab-ci.yml
# needed for status reporting and defining the runners
BASE_CONFIG = """
.base:
  before_script:
    - mkdir -p /tmp/artifacts
    - schutzbot/ci_details.sh > /tmp/artifacts/ci-details-before-run.txt
    - cat schutzbot/team_ssh_keys.txt | tee -a ~/.ssh/authorized_keys > /dev/null
  after_script:
    - schutzbot/ci_details.sh > /tmp/artifacts/ci-details-after-run.txt || true
    - schutzbot/update_github_status.sh update || true
    - schutzbot/save_journal.sh || true
    - schutzbot/upload_artifacts.sh
  interruptible: true
  retry: 1
  tags:
    - terraform

.terraform:
  extends: .base
  tags:
    - terraform
"""

NULL_CONFIG = """
NullBuild:
  stage: test
  script: "true"
"""


def runcmd(cmd, stdin=None):
    job = sp.run(cmd, input=stdin, capture_output=True)
    if job.returncode > 0:
        print(f"Command failed: {cmd}")
        if job.stdout:
            print(job.stdout.decode())
        if job.stderr:
            print(job.stderr.decode())
        sys.exit(job.returncode)

    return job.stdout, job.stderr


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
    out, err = runcmd(["go", "run", "./cmd/list-images", "-json",
                       "-distros", distros_arg, "-arches", arches_arg, "-images", images_arg])
    return json.loads(out)


def s3_auth_args():
    s3_key = os.environ.get("V2_AWS_SECRET_ACCESS_KEY")
    s3_key_id = os.environ.get("V2_AWS_ACCESS_KEY_ID")
    if s3_key and s3_key_id:
        return [f"--access_key={s3_key_id}", f"--secret_key={s3_key}"]

    return []


def dl_s3_configs(destination):
    """
    Downloads all the configs from the s3 bucket.
    """
    s3url = f"{S3_BUCKET}/{S3_PREFIX}/"
    print(f"Downloading configs from {s3url}")
    job = sp.run(["s3cmd", *s3_auth_args(), "sync", "--delete-removed", s3url, destination], capture_output=True)
    ok = job.returncode == 0
    if not ok:
        print(f"Failed to sync contents of {s3url}:")
        print(job.stderr.decode())
    return ok


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
        print("ERROR: The following test configs have names that don't match their filenames.")
        print("\n".join(bad_configs))
        print("This will produce incorrect test generation and results.")
        print("Aborting.")
        sys.exit(1)


def read_manifests(path):
    """
    Read all manifests in the given path, calculate their IDs, and return a dictionary mapping each filename to the data
    and its ID.
    """
    print(f"Reading manifests in {path}")
    manifests = {}
    for manifest_fname in os.listdir(path):
        manifest_path = os.path.join(path, manifest_fname)
        with open(manifest_path) as manifest_file:
            manifest_data = json.load(manifest_file)
        manifests[manifest_fname] = {
            "data": manifest_data,
            "id": get_manifest_id(manifest_data["manifest"]),
        }
    print("Done")
    return manifests


def filter_builds(manifests, skip_ostree_pull=True):
    """
    Returns a list of build requests for the manifests that have no matching config in the test build cache.
    """
    print(f"Filtering {len(manifests)} build configurations")
    dl_path = os.path.join(TEST_CACHE_ROOT, "s3configs", "builds/")
    os.makedirs(dl_path, exist_ok=True)
    build_requests = []

    dl_s3_configs(dl_path)

    errors = []

    for manifest_fname, data in manifests.items():
        manifest_id = data["id"]
        id_fname = manifest_id + ".json"

        data = data.get("data")
        build_request = data["build-request"]
        distro = build_request["distro"]
        arch = build_request["arch"]
        image_type = build_request["image-type"]
        config = build_request["config"]
        config_name = config["name"]

        # check if the config specifies an ostree URL and skip it if requested
        if skip_ostree_pull and config.get("ostree", {}).get("url"):
            print(f"Skipping {distro}/{arch}/{image_type}/{config_name} (ostree dependency)")
            continue

        # add manifest id to build request
        build_request["manifest-checksum"] = manifest_id

        # check if the hash_fname exists in the synced directory
        dl_config_dir = os.path.join(dl_path, distro, arch)
        id_config_path = os.path.join(dl_config_dir, id_fname)

        # check if the id_fname exists in the synced directory
        if os.path.exists(id_config_path):
            try:
                with open(id_config_path) as dl_config_fp:
                    dl_config = json.load(dl_config_fp)
                commit = dl_config["commit"]
                print(f"Manifest {manifest_fname} was successfully built in commit {commit}")
                continue
            except json.JSONDecodeError as jd:
                errors.append((
                        f"failed to parse {id_config_path}\n"
                        f"{jd.msg}\n"
                        "Scheduling config for rebuild\n"
                        f"Config: {distro}/{arch}/{image_type}/{config_name}\n"
                ))

        build_requests.append(build_request)

    print("Config filtering done!\n")
    if errors:
        # print errors at the end so they're visible
        print("Errors:")
        print("\n".join(errors))

    return build_requests
