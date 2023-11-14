### Useful cmds

The following utilities, defined in the `cmd/` directory, are useful for
development and testing. They **should not** be relied on for production
purposes. In particular, command line options and default behaviour can change
at any time.

The following are high level descriptions of what some of the utilities can do
and how they can be used during development. For specific flags and options,
refer to each command's help output and doc strings.

Each utility can be compiled using `go build -o <outputfile> ./cmd/<utility>`
or run directly using `go run ./cmd/<utility>`. Use `go run ./cmd/<utility>
-help` for option descriptions (e.g. `go run ./cmd/gen-manifests -help`).

#### Manifest generation

The `gen-manifests` tool can be used to generate all or a subset of the
manifests for the images defined in the repository. This is useful for quickly
seeing the effects of changes in image definitions on the manifest and the
image build itself. While manifests are meant to be machine readable, it is
often much faster to inspect the difference between two manifests (before and
after a change in code) to evaluate if a change is having the desired effect.

Manifests can be generated with or without content resolution (e.g. package
depsolving, containers, ostree commits). If you are working on changes in image
definitions that do not rely on content (e.g. an image type's partition table),
manifests without resolved content can be generated almost instantly. Note that
even though content is not resolved and packages are not depsolved, the
selected packages without their dependencies are still added to generated
manifests, so disabling package depsolving can also be used to inspect package
selection without dependencies.

Manifests should be generated with all content enabled if they are going to be
built. A common workflow when working on changing image definitions, or adding
a new image type, might be:
1. Generate the manifests for the image types that you will be working on.
2. Make changes in an existing image definition or add a new image type.
3. Add appropriate configuration changes:
    - If a new image type is added, add it to the [config
      map](test/config-map.json) under an appropriate configuration file or
      write a new one.
    - If an existing image type is being modified, and the change depends on an
      image customization, make sure the modification is covered by an existing
      [test config](test/configs).
4. Generate the relevant manifests without content (`-packages=false
   -containers=false -commits=false`).
    - If the change depends on a customization, it might be more useful to
      generate multiple manifests with different configuration options set and
      inspect the differences between them.
5. Inspect the differences between manifests generated in steps 1 and 4.
6. Generate manifest with all content enabled for the relevant image types.
7. Build at least one of the manifests using `osuild` and inspect the output
   (boot the image or mount it to look for the desired changes).

_NOTE: By default, manifests created with the `gen-manifest` tool contain extra
metadata. The manifest itself is stored under the key "manifest". You can
extract the actual manifest using `jq .manifest
<manifestfile>.json`. Alternatively, you can generate manifests without
metadata using the `-metadata=false` option._

#### Building images

You can build an image by generating its manifest and then running
osbuild. Alternatively, the `cmd/build` tool can perform both steps in one
call. It will generate a manifest, build the image, and store both the image
and its manifest in the output directory.

The build tool must be run as root because image building with osbuild requires
superuser privileges. It is **not recommended** to run `sudo go run
./cmd/build` however. The `go run` command can make changes to the go build
cache and if these changes are made as root, it can cause issues when running
other go commands in the future as a regular user. Instead, it is recommended
to first build the binary and then run it as root:
```
go build -o bin/build ./cmd/build
sudo ./bin/build ...
```

#### Booting images

You can boot an image in its target environment by using the appropriate
command from `cmd/`. _Currently, only AWS is supported._

For example, to boot an AMI or EC2 image, you can use the `./cmd/boot-aws`
command with the `setup` subcommand:
```bash
go run ./cmd/boot-aws setup \
     --access-key-id "${AWS_ACCESS_KEY_ID}" \
     --secret-access-key "${AWS_SECRET_ACCESS_KEY}" \
     --region "${AWS_REGION}" \
     --bucket "${AWS_BUCKET}" \
     --ami-name "${IMAGE_NAME}" \
     --s3-key "${IMAGE_KEY}" \
     --username "${USERNAME}" \
     --arch "${IMAGE_ARCHITECTURE}" \
     --ssh-pubkey "${PATH_TO_SSH_PUBLIC_KEY}" \
     --ssh-privkey "${PATH_TO_SSH_PRIVATE_KEY}" \
     --resourcefile ./aws-test-resources.json \
     ${PATH_TO_IMAGE_FILE}
```
where:
- `${AWS_ACCESS_KEY_ID}` and `${AWS_SECRET_ACCESS_KEY}` are the AWS credentials,
- `${AWS_REGION}` is the AWS region to use,
- `${AWS_BUCKET}` is an S3 bucket (that must already exist),
- `${IMAGE_NAME}` is the name to use for registering the AMI,
- `${IMAGE_KEY}` is the key (filename) to use for the file in S3,
- `${USERNAME}` is the username to set up on the instance,
- `${IMAGE_ARCHITECTURE}` is the hardware architecture of the image being
  uploaded and booted,
- `${PATH_TO_SSH_PUBLIC_KEY}` and `${PATH_TO_SSH_PRIVATE_KEY}` point to an
  public/private SSH key pair.

This command will upload the image to S3, register the image as an AMI, create
a security group configured to allow SSH access, and launch an instance from
the AMI. It will then wait until the instance is ready and print its public IP
address. It will also use the public ssh key and provided username to configure
cloud-init to create a user and set the ssh key on first boot.

The IDs of all created resources are stored in the file specified by the
`--resourcefile` flag. This can be used to tear down all the resources created
by the `setup` subcommand:
```bash
go run ./cmd/boot-aws teardown \
     --access-key-id "${AWS_ACCESS_KEY_ID}" \
     --secret-access-key "${AWS_SECRET_ACCESS_KEY}" \
     --region "${AWS_REGION}" \
     --bucket "${AWS_BUCKET}" \
     --name "${IMAGE_NAME}" \
     --key "${IMAGE_KEY}" \
     --username "${USERNAME}" \
     --arch "${IMAGE_ARCHITECTURE}" \
     --ssh-pubkey "${PATH_TO_SSH_PUBLIC_KEY}" \
     --ssh-privkey "${PATH_TO_SSH_PRIVATE_KEY}" \
     --resourcefile ./aws-test-resources.json
```

Alternatively, a setup-test-teardown procedure can be run in a single command using the `run` subcommand:
```bash
go run ./cmd/boot-aws run \
     --access-key-id "${AWS_ACCESS_KEY_ID}" \
     --secret-access-key "${AWS_SECRET_ACCESS_KEY}" \
     --region "${AWS_REGION}" \
     --bucket "${AWS_BUCKET}" \
     --ami-name "${IMAGE_NAME}" \
     --s3-key "${IMAGE_KEY}" \
     --username "${USERNAME}" \
     --arch "${IMAGE_ARCHITECTURE}" \
     --ssh-pubkey "${PATH_TO_SSH_PUBLIC_KEY}" \
     --ssh-privkey "${PATH_TO_SSH_PRIVATE_KEY}" \
     ${PATH_TO_IMAGE_FILE} ${PATH_TO_SCRIPT}
```

This will perform the same steps as the `setup` subcommand, then upload the
script specified by `${PATH_TO_SCRIPT}` to the instance, run it, and then
perform the same actions as the `teardown` subcommand.

#### Listing available image type configurations

The `cmd/list-images` utility simply lists all available combinations of
distribution, architecture, and image type. It also supports filtering one or
more of those three variables.
