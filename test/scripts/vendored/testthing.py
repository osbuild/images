#!/usr/bin/python3
# SPDX-License-Identifier: GPL-3.0-or-later

"""test.thing - A simple modern VM runner.

https://codeberg.org/lis/test.thing

A simple VM runner script exposing an API useful for use as a pytest fixture.
Can also be used to run a VM and login via the console.

Each VM is allocated an identifier: 'tt.0', 'tt.1', etc.

For each VM, an ephemeral ssh key is created and used to connect to the VM via
vsock with systemd-ssh-proxy, which works even if the guest doesn't have
networking configured.  The ephemeral key means that access is limited to the
current user (since vsock connections are otherwise available to all users on
the host system).  The guest needs to have systemd 256 for this to work.

An ssh control socket is created for sending commands and can also be used
externally, avoiding the need to authenticate.  A suggested ssh config:

```
Host tt.*
        ControlPath ${XDG_RUNTIME_DIR}/test.thing/%h/ssh
```

And then you can say `ssh tt.0` or `scp file tt.0:/tmp`.
"""

# When copying test.thing into your own project, try to use a tagged version.
# If you need to use a version between tags or have made your own
# modifications, please make note if it by modifying the version number.
__version__ = "0.4.0"

import argparse
import asyncio
import contextlib
import ctypes
import dataclasses
import functools
import itertools
import json
import logging
import os
import pathlib
import re
import shlex
import shutil
import signal
import sys
import tempfile
import traceback
import weakref
from collections.abc import (
    AsyncGenerator,
    Callable,
    Iterable,
    Iterator,
    Mapping,
    Sequence,
)
from pathlib import Path
from types import TracebackType
from typing import Any, Literal, Never, Self

logger = logging.getLogger(__name__)

COCKPIT_TEST_IDENTITY = """
-----BEGIN RSA PRIVATE KEY-----
MIIEpQIBAAKCAQEA1DrTSXQRF8isQQfPfK3U+eFC4zBrjur+Iy15kbHUYUeSHf5S
jXPYbHYqD1lHj4GJajC9okle9rykKFYZMmJKXLI6987wZ8vfucXo9/kwS6BDAJto
ZpZSj5sWCQ1PI0Ce8CbkazlTp5NIkjRfhXGP8mkNKMEhdNjaYceO49ilnNCIxhpb
eH5dH5hybmQQNmnzf+CGCCLBFmc4g3sFbWhI1ldyJzES5ZX3ahjJZYRUfnndoUM/
TzdkHGqZhL1EeFAsv5iV65HuYbchch4vBAn8jDMmHh8G1ixUCL3uAlosfarZLLyo
3HrZ8U/llq7rXa93PXHyI/3NL/2YP3OMxE8baQIDAQABAoIBAQCxuOUwkKqzsQ9W
kdTWArfj3RhnKigYEX9qM+2m7TT9lbKtvUiiPc2R3k4QdmIvsXlCXLigyzJkCsqp
IJiPEbJV98bbuAan1Rlv92TFK36fBgC15G5D4kQXD/ce828/BSFT2C3WALamEPdn
v8Xx+Ixjokcrxrdeoy4VTcjB0q21J4C2wKP1wEPeMJnuTcySiWQBdAECCbeZ4Vsj
cmRdcvL6z8fedRPtDW7oec+IPkYoyXPktVt8WsQPYkwEVN4hZVBneJPCcuhikYkp
T3WGmPV0MxhUvCZ6hSG8D2mscZXRq3itXVlKJsUWfIHaAIgGomWrPuqC23rOYCdT
5oSZmTvFAoGBAPs1FbbxDDd1fx1hisfXHFasV/sycT6ggP/eUXpBYCqVdxPQvqcA
ktplm5j04dnaQJdHZ8TPlwtL+xlWhmhFhlCFPtVpU1HzIBkp6DkSmmu0gvA/i07Z
pzo5Z+HRZFzruTQx6NjDtvWwiXVLwmZn2oiLeM9xSqPu55OpITifEWNjAoGBANhH
XwV6IvnbUWojs7uiSGsXuJOdB1YCJ+UF6xu8CqdbimaVakemVO02+cgbE6jzpUpo
krbDKOle4fIbUYHPeyB0NMidpDxTAPCGmiJz7BCS1fCxkzRgC+TICjmk5zpaD2md
HCrtzIeHNVpTE26BAjOIbo4QqOHBXk/WPen1iC3DAoGBALsD3DSj46puCMJA2ebI
2EoWaDGUbgZny2GxiwrvHL7XIx1XbHg7zxhUSLBorrNW7nsxJ6m3ugUo/bjxV4LN
L59Gc27ByMvbqmvRbRcAKIJCkrB1Pirnkr2f+xx8nLEotGqNNYIawlzKnqr6SbGf
Y2wAGWKmPyEoPLMLWLYkhfdtAoGANsFa/Tf+wuMTqZuAVXCwhOxsfnKy+MNy9jiZ
XVwuFlDGqVIKpjkmJyhT9KVmRM/qePwgqMSgBvVOnszrxcGRmpXRBzlh6yPYiQyK
2U4f5dJG97j9W7U1TaaXcCCfqdZDMKnmB7hMn8NLbqK5uLBQrltMIgt1tjIOfofv
BNx0raECgYEApAvjwDJ75otKz/mvL3rUf/SNpieODBOLHFQqJmF+4hrSOniHC5jf
f5GS5IuYtBQ1gudBYlSs9fX6T39d2avPsZjfvvSbULXi3OlzWD8sbTtvQPuCaZGI
Df9PUWMYZ3HRwwdsYovSOkT53fG6guy+vElUEDkrpZYczROZ6GUcx70=
-----END RSA PRIVATE KEY-----
"""
"""A copy of `bots/machine/identity` from the cockpit project.  Many existing
VM images have the public half of this key inside of them, so it's useful for
gaining access to those if they lack support for ephemeral ssh keys."""


# This is basically tempfile.TemporaryDirectory but sequentially-allocated.
# We do that so we can easily interact with the VMs from outside (with ssh).
class IpcDirectory:
    """A context manager for the VM IPC directory.

    This is very similar to tempfile.TemporaryDirectory() except that the
    allocation is predictable (sequential): the created directory will be
    `/run/user/$uid/test.thing/tt.n` for the smallest `n` that we find.

    It works the same way:

        with IpcDirectory() as path:
            ...use path...

    The directory gets tagged with a `pid` file containing the pid and pidfd
    inode of the current process.  This could be helpful for pruning dead
    directories, but is currently unused.
    """

    finalizer: Callable[[], None] | None = None

    @staticmethod
    def _find_dir() -> Path:
        try:
            xdg_rundir = Path(os.environ["XDG_RUNTIME_DIR"])
        except KeyError:
            # No XDG_RUNTIME_DIR?  Somewhere in /tmp will have to do
            return Path(tempfile.mkdtemp())

        for n in range(10000):
            tmpdir = xdg_rundir / "test.thing" / f"tt.{n}"

            try:
                tmpdir.mkdir(exist_ok=False, parents=True, mode=0o700)
            except FileExistsError:
                continue

            return tmpdir

        raise FileExistsError

    def __enter__(self) -> Path:
        """Create a unique directory.

        This will sequentially allocate the first available 'tt.0', 'tt.1',
        etc. directory and return it as a `Path`.
        """
        pid = os.getpid()
        pidfd = os.pidfd_open(pid)
        try:
            buf = os.fstat(pidfd)
            unique_id = f"{pid} {buf.st_ino}\n"
        finally:
            os.close(pidfd)

        tmpdir = self._find_dir()
        self.finalizer = weakref.finalize(self, shutil.rmtree, tmpdir)
        (tmpdir / "pid").write_text(unique_id)
        return tmpdir

    def __exit__(self, *args: object) -> None:
        """Delete the IPC directory and its contents."""
        del args
        if self.finalizer:
            self.finalizer()


def _normalize_args(
    *args: str | pathlib.PurePath | tuple[str | pathlib.PurePath, ...],
) -> Iterable[Iterable[str]]:
    for chunk in args:
        if not isinstance(chunk, tuple):
            yield (str(chunk),)
        elif len(chunk) != 0:
            yield map(str, chunk)


def _pretty_print_args(
    *args: str | pathlib.PurePath | tuple[str | pathlib.PurePath, ...],
) -> str:
    """Pretty-print a nested argument list.

    This takes the argument list format used by test.thing and turns it into a
    format that looks like a nicer version of `set -x` from POSIX shell.
    """
    if not any(isinstance(arg, tuple) for arg in args):
        # No tuples: use the boring format
        return shlex.join(map(str, args))

    # There are tuples: use the fancy format
    return " \\\n      ".join(map(shlex.join, _normalize_args(*args)))


def _find_qemu() -> Path:
    for candidate in ("qemu-kvm", "kvm"):
        if cmd := shutil.which(candidate):
            return Path(cmd)

    raise FileNotFoundError("Unable to find qemu-kvm")


def _find_ovmf() -> Path:
    candidates = [
        # path for Fedora/RHEL (our tasks container)
        "/usr/share/OVMF/OVMF_CODE.fd",
        # path for Ubuntu (GitHub Actions runners)
        "/usr/share/ovmf/OVMF.fd",
        # path for Arch
        "/usr/share/edk2/x64/OVMF.4m.fd",
    ]

    for path in map(Path, candidates):
        if path.exists():
            return path

    raise FileNotFoundError("Unable to find OVMF UEFI BIOS")


async def _qmp_command(ipc: Path, command: str) -> object:
    reader, writer = await asyncio.open_unix_connection(ipc / "qmp")

    async def execute(command: str) -> object:
        writer.write((json.dumps({"execute": command}) + "\n").encode())
        await writer.drain()
        while True:
            response = json.loads(await reader.readline())
            if "event" in response:
                continue
            if "return" in response:
                return response["return"]
            raise RuntimeError(f"Got error response from qmp: {response!r}")

    # Trivial handshake (ignore them, send nothing)
    _ = json.loads(await reader.readline())
    await execute("qmp_capabilities")

    response = await execute(command)

    writer.close()
    await writer.wait_closed()

    return response


def _ssh_direct_args(
    identities: Sequence[Path], port: Path
) -> tuple[tuple[str, str], ...]:
    options = {
        # Fake that we know the host key
        "KnownHostsCommand": "/bin/echo %H %t %K",
        # Use systemd-ssh-proxy to connect via vsock
        "ProxyCommand": f"/usr/lib/systemd/systemd-ssh-proxy vsock-mux/{port} 22",
        "ProxyUseFdpass": "yes",
        # Try to prevent interactive prompting and/or updating known_hosts
        # files or otherwise interacting with the environment
        "BatchMode": "yes",
        "IdentitiesOnly": "yes",
        "PKCS11Provider": "none",
        "PasswordAuthentication": "no",
        "StrictHostKeyChecking": "yes",
        "User": "root",
        "UserKnownHostsFile": "/dev/null",
    }

    return (
        ("-F", "none"),  # don't use the user's config
        *(("-o", f"{k} {v}") for k, v in options.items()),
        *(("-o", f"IdentityFile {path}") for path in identities),
    )


@functools.cache
def _stderr_is_tty() -> bool:
    return os.isatty(2)


class UI:
    """A helper for printing messages and launching subprocesses."""

    def __init__(self, *, status_messages: bool, verbose: bool) -> None:
        """Create a UI helper.

        This controls the stderr output of test.thing.  The test.thing library
        never writes to stdout.

         - status_messages: if intermediary messages about the state of the
           machine should be printed or not
         - verbose: extra output is printed (like all executed commands)

        If both are false then nothing will be printed.
        """
        self._status_messages = status_messages
        self._verbose = verbose

    def clear_status_message(self) -> None:
        """Clear any displayed status message."""
        if _stderr_is_tty():
            sys.stderr.write("\r\033[2K")

    def print_status(self, line: str) -> None:
        """Print a status line message.

        This is only printed if status_messages=True.  A status message is a
        transient message that will be erased at the next output (unless stderr
        is not a TTY).
        """
        if self._status_messages:
            if os.isatty(2):
                sys.stderr.write("\r\033[2K  " + line + "\r")
            else:
                sys.stderr.write(line + "\n")

    def print_verbose(self, line: str) -> None:
        """Print a verbose message, if verbose=True."""
        if self._verbose:
            self.clear_status_message()
            sys.stderr.write(line + "\n")

    def print(self, line: str) -> None:
        """Print a message, unconditionally."""
        self.clear_status_message()
        sys.stderr.write(line + "\n")

    async def _wait_stdin(self, msg: str) -> None:
        r"""Wait until stdin sees \n or EOF.

        This prints the given message to stdout without adding an extra newline.

        The input is consumed (and discarded) up to the \n and maybe more...
        """
        done = asyncio.Event()

        def stdin_ready() -> None:
            data = os.read(0, 4096)
            if not data:
                sys.stdout.write("\n")
            if not data or b"\n" in data:
                done.set()

        loop = asyncio.get_running_loop()
        loop.add_reader(0, stdin_ready)
        sys.stdout.write(msg)
        sys.stdout.flush()
        try:
            await done.wait()
        finally:
            loop.remove_reader(0)

    async def sit(
        self, msg: str | None = None, exc: BaseException | None = None
    ) -> None:
        """Wait for the user to press Enter."""
        # <pitti> lis: the ages old design question: does the button show
        # the current state or the future one when you press it :)
        # <lis> pitti: that's exactly my question.  by taking the one with
        # both then i don't have to choose! :D
        # <lis> although to be honest, i find your argument convincing.
        # i'll put the ⏸️ back
        # <pitti> lis: it was more of a joke -- I actually agree that a
        # play/pause button is nicer
        # <lis> too late lol

        self.clear_status_message()

        if exc is not None:
            sys.stderr.write(f"\n🤦 {''.join(traceback.format_exception(exc))}\n")
        if msg is not None:
            sys.stderr.write(f"\n{msg}\n")

        await self._wait_stdin("\nEnter or EOF to exit ⏸️ ")

    async def spawn(
        self,
        *args: str | Path | tuple[str | Path, ...],
        stdin: int | None = asyncio.subprocess.DEVNULL,
        stdout: int | None = None,
        stderr: int | None = None,
    ) -> asyncio.subprocess.Process:
        """Spawn a process.

        This has a couple of extra niceties: the args list is flattened, Path is
        converted to str, the spawned process is logged to stderr for debugging,
        and we call PR_SET_PDEATHSIG with SIGTERM after forking to make sure the
        process exits with us.

        The flattening allows grouping logically-connected arguments together,
        producing nicer verbose output, allowing for adding groups of arguments
        from helper functions or comprehensions, and works nicely with code
        formatters:

        For example:

        private = Path(...)
        options = { ... }

        ssh = await spawn(
            "ssh",
            ("-i", private),
            *(("-o", f"{k} {v}") for k, v in options.items()),
            ("-l", "root", "x"),
            ...
        )

        The type of the groups is `tuple`.  It could be `Sequence` but this would
        also allow using bare strings, which would be split into their individual
        characters.  Using `tuple` prevents this from happening.
        """
        # This might be complicated: do it before the fork
        prctl = ctypes.CDLL(None, use_errno=True).prctl

        def pr_set_pdeathsig() -> None:
            PR_SET_PDEATHSIG = 1  # noqa: N806
            if prctl(PR_SET_PDEATHSIG, signal.SIGTERM):
                os._exit(1)  # should never happen

        # If we are expecting output, clear the status message
        if stdout is None or stderr is None:
            self.clear_status_message()

        self.print_verbose(f"+ {_pretty_print_args(*args)}\n")

        return await asyncio.subprocess.create_subprocess_exec(
            *itertools.chain(*_normalize_args(*args)),
            stdin=stdin,
            stdout=stdout,
            stderr=stderr,
            preexec_fn=pr_set_pdeathsig,
        )

    async def run(
        self,
        *args: str | Path | tuple[str | Path, ...],
        stdin: int | None = asyncio.subprocess.DEVNULL,
        check: bool = True,
    ) -> int:
        """Run a process, waiting for it to exit.

        This takes the same arguments as spawn, plus a "check" argument (True by
        default) which works in the usual way.
        """
        process = await self.spawn(*args, stdin=stdin)
        returncode = await process.wait()
        if check and returncode != 0:
            raise SubprocessError(args, returncode=returncode)
        return returncode


class GuestPath(pathlib.PurePosixPath):
    """A path on the virtual machine guest.

    This aims to support similar operations to pathlib.Path (with similar
    APIs), but most operations are async and many have slightly different
    feature sets.
    """

    __slots__ = ("_vm",)

    def __init__(self, *args: str | os.PathLike[str], vm: "VirtualMachine") -> None:
        """Create a GuestPath for a path on a guest."""
        super().__init__(*args)
        self._vm = vm

    def with_segments(self, *pathsegments: str | os.PathLike[str]) -> Self:
        """Create a new path by combining the given pathsegments."""
        return type(self)(*pathsegments, vm=self._vm)

    async def mkdir(self, *, mode: int | None = None, parents: bool = False) -> None:
        """Create a directory."""
        await self._vm.execute(
            "mkdir",
            "-p" if parents else (),
            ("-m", f"{mode:0o}") if mode is not None else (),
            self,
        )

    async def chmod(self, mode: int | str, *, follow_symlinks: bool = True) -> None:
        """Change a file mode."""
        await self._vm.execute(
            "chmod",
            "-h" if not follow_symlinks else (),
            f"{mode:0o}" if isinstance(mode, int) else mode,
            self,
        )

    async def chown(
        self,
        owner: str | tuple[str | int | None, str | int | None],
        *,
        follow_symlinks: bool = True,
    ) -> None:
        """Change the owner of a file.

        The owner can be a string like 'user:group' or a pair of (user, group)
        where each can be a string, int, or None (to make no change).
        """
        if isinstance(owner, tuple):
            user, group = owner
            owner = f"{user or ''}:{group or ''}"

        await self._vm.execute(
            "chown", "-h" if not follow_symlinks else (), owner, self
        )

    async def write_bytes(self, data: bytes, *, append: bool = False) -> None:
        """Write or append to to a binary file."""
        await self._vm.execute(
            "dd",
            "status=none",
            ("conv=notrunc", "oflag=append") if append else (),
            f"of={self}",
            input=data,
        )

    async def read_text(self) -> str:
        """Read a text file."""
        return await self._vm.execute("cat", self)

    async def write_text(self, data: str, *, append: bool = False) -> None:
        """Write or append to a text file."""
        await self.write_bytes(data.encode(), append=append)

    async def unlink(
        self, *, missing_ok: bool = False, recursive: bool = False
    ) -> None:
        """Unlink the given file."""
        await self._vm.execute(
            "rm", "-f" if missing_ok else (), "-r" if recursive else (), self
        )

    async def rmdir(self) -> None:
        """Remove a directory."""
        await self._vm.execute("rmdir", self)


@dataclasses.dataclass
class Network:
    """A virtual machine network."""

    id: int | Literal["user"]
    """The network identifier.  If this is an integer then it specifies a
    multicast network on which other virtual machines can communicate.  If it's
    the literal value "user" then this sets up usermode networking."""

    @classmethod
    def user(cls) -> Self:
        """Create a user-mode network."""
        return cls(id="user")

    @classmethod
    def multicast(cls, netnr: int) -> Self:
        """Create a multicast network for talking to other machines."""
        return cls(id=netnr)

    def to_qemu(self) -> str:
        """Describe the network in a way that qemu understands."""
        if self.id == "user":
            return "user"

        # Same as Cockpit
        return f"socket,mcast=230.0.0.1:{self.id},localaddr=127.0.0.1"


class VirtualMachine:
    """A handle to a running virtual machine.

    This is meant to be used as an async context manager like so:

    with IpcDirectory() as ipc:
        image = Path("...")
        async with VirtualMachine(image, ipc=ipc) as vm:
            await vm.execute("cat", "/usr/lib/os-release")

    The user of the context manager runs in context of an asyncio.TaskGroup and
    will be cancelled if anything unexpected happens (ssh connection lost, VM
    exiting, etc).

    When the context manager is exited the machine is taken down.

    When the machine is running it is also possible to access it from outside.
    See the documentation for the module.
    """

    """ssh command-line arguments for executing commands"""
    ssh_args: tuple[str | Path | tuple[str | Path, ...], ...]

    """ssh command-line arguments to be used to connect directly to the vsock"""
    ssh_direct_args: tuple[str | Path | tuple[str | Path, ...], ...] | None

    # our three background tasks
    _notify_server_task: asyncio.Task[None] | None = None
    _ssh_control_task: asyncio.Task[None] | None = None
    _qemu_task: asyncio.Task[None] | None = None

    def __init__(
        self,
        image: Path | str,
        *,
        ipc: Path,
        attach_console: bool = False,
        boot: Literal["efi", "mbr"] = "efi",
        cloud_init_user_data: Mapping[str, object] | None = None,
        credentials: Mapping[str, str] = {},
        cpus: int = 4,
        identity: tuple[Path, str | None] | None = None,
        identities: Sequence[str | Path] = (),
        memory: int | str = "4G",
        networks: Sequence[Network] = (),
        provision_ssh_key: bool = False,
        sit: bool = False,
        snapshot: bool = True,
        status_messages: bool = False,
        target: str = "sockets.target",
        timeout: float = 30.0,
        ui: UI | None = None,
        verbose: bool = False,
    ) -> None:
        """Construct a VM.

        The kwargs allow customizing the behaviour:
          - attach_console: if qemu should connect the console to stdio
          - boot: if we should boot with EFI or via the MBR
          - cloud_init_user_data: JSON user-data for cloud-init
          - credentials: extra system credentials
          - cpus: the number of CPUs
          - identity: a path to an ssh private key and the public key as a string.
            If the public key is specified as None then it won't be configured on
            the guest.  The default (None) is to generate an ephemeral keypair.
          - identities: extra private keys (either as paths on the disk or
            directly as strings) to pass to ssh.  Useful if you're not sure
            which key the image will accept and want to try multiple.
          - memory: how much memory the guest gets in MiB, or a string like "4G"
          - networks: a list of Network objects (or empty to disable networking)
          - provision_ssh_key: if we should attempt to install the public side
            of the ssh key as ~root/.ssh/authorized_keys in the guest
          - sit: if we should "sit" when an exception occurs: print the exception
            and wait for input (to allow inspecting the running VM)
          - snapshot: if the 'snapshot' option is used on the disk (changes are
            transient)
          - status_messages: if we should do output of status messages (stderr)
          - target: the name of the systemd target to wait for
          - timeout: how long to wait for the VM to start, or 'inf'
          - ui: a custom instance of UI (otherwise a new one is constructed per
            status_messages= and verbose=)
          - verbose: if we should do output of verbose messages (stderr)
        """
        self.image = image
        self._ipc = ipc
        self._attach_console = attach_console
        self._boot = boot
        self._cloud_init_user_data = cloud_init_user_data
        self._cpus = cpus
        self._credentials = credentials
        self._identity = identity
        self._identities = identities
        self._memory = memory
        self._networks = networks
        self._provision_ssh_key = provision_ssh_key
        self._sit = sit
        self._snapshot = snapshot
        self._target = target
        self._timeout = timeout
        self._ui = ui or UI(status_messages=status_messages, verbose=verbose)

        self._tasks = asyncio.TaskGroup()
        self._ssh_control_ready = asyncio.Event()
        self._qemu_exited = asyncio.Event()
        self._shutdown_ok = False

        self.root = GuestPath("/", vm=self)
        self.home = GuestPath(".", vm=self)

    async def _run_helper(
        self,
        *args: str | Path | tuple[str | Path, ...],
        ready_when: Path,
    ) -> asyncio.Task[None]:
        """Run a helper process in the background.

        The function returns when the given named file (usually a socket)
        appears.  The process is awaited from our task group.  It should never
        exit.  On cancellation, the process is terminated.
        """
        process = await self._ui.spawn(*args)

        # block until we see the expected socket or an unexpected exit
        while not ready_when.exists():
            with contextlib.suppress(TimeoutError):
                returncode = await asyncio.wait_for(process.wait(), 0.01)
                raise SubprocessError(args, returncode=returncode)

        async def wait_exit() -> None:
            try:
                returncode = await process.wait()  # this should never return
                raise SubprocessError(args, returncode=returncode)
            except asyncio.CancelledError:
                process.kill()
                await process.wait()

        return self._tasks.create_task(wait_exit())

    async def _ssh_keygen(self) -> tuple[Path, str]:
        """Create a ssh key in the given directory.

        Returns the path to the private key and the public key as a string.
        """
        private_key = self._ipc / "id"

        await self._ui.run(
            "ssh-keygen",
            "-q",  # quiet
            ("-t", "ed25519"),
            ("-N", ""),  # no passphrase
            ("-C", ""),  # no comment
            ("-f", f"{private_key}"),
        )

        return private_key, (self._ipc / "id.pub").read_text().strip()

    def _sd_notify(self, line: str) -> None:
        logger.debug("sd_notify:%s", line)

        # Only print target updates when ssh is offline
        if self._ssh_control_task is not None:
            return

        key, _, value = line.partition("=")
        if key == "X_SYSTEMD_UNIT_ACTIVE":
            self._ui.print_status(f"Reached target: {value}")
            if value == self._target:
                self._ssh_control_task = self._tasks.create_task(self._ssh_control())

        elif key == "X_SYSTEMD_UNIT_INACTIVE":
            self._ui.print_status(f"Unit inactive: {value}")
        elif key == "X_SYSTEMD_SHUTDOWN":
            self._ui.print_status(f"Shutdown: {value}")

    async def _sd_notify_connection(
        self, reader: asyncio.StreamReader, writer: asyncio.StreamWriter
    ) -> None:
        logger.debug("sd_notify connection")
        try:
            # Actually we should read until EOF but see
            # https://github.com/rust-vmm/vhost-device/issues/874
            message = await reader.read(65536)
            self._sd_notify(message.decode())
        finally:
            writer.close()
            await writer.wait_closed()

    async def _qemu(
        self,
        *,
        port: int,
        public: str | None,
    ) -> None:
        creds = {
            **self._credentials,
            "vmm.notify_socket": f"vsock-stream:2:{port}",
        }

        if public is not None:
            creds["ssh.ephemeral-authorized_keys-all"] = public
            if self._provision_ssh_key:
                creds["ssh.authorized_keys.root"] = public

        snap = "on" if self._snapshot else "off"
        drives = [f"file={self.image},format=qcow2,discard=unmap,snapshot={snap}"]

        if self._cloud_init_user_data:
            cloud_init = self._ipc / "cloud-init"
            cloud_init.mkdir()
            (cloud_init / "meta-data").touch()
            (cloud_init / "user-data").write_text(
                "#cloud-config\n" + json.dumps(self._cloud_init_user_data) + "\n"
            )
            drives.append(f"driver=vvfat,dir={cloud_init},readonly=on,label=CIDATA")

        args = (
            _find_qemu(),
            "-nodefaults",
            ("-object", f"memory-backend-memfd,share=on,id=mem0,size={self._memory}"),
            ("-bios", _find_ovmf()) if self._boot == "efi" else (),
            ("-machine", "q35,accel=kvm,memory-backend=mem0"),
            ("-cpu", "host"),
            ("-smp", f"{self._cpus}"),
            ("-m", f"{self._memory}"),
            ("-display", "none"),
            ("-qmp", f"unix:{self._ipc}/qmp,server,wait=off"),
            ("-chardev", f"socket,id=vsock,reconnect=0,path={self._ipc}/vsock-device"),
            ("-device", "vhost-user-vsock-pci,chardev=vsock"),
            # Console stuff...
            ("-device", "virtio-serial-pci"),
            (
                "-chardev",
                f"socket,path={self._ipc}/vsock_1111,id=tt-notify,reconnect-ms=1",
            ),
            ("-device", "virtserialport,chardev=tt-notify,name=tt-notify"),
            ("-serial", "chardev:console"),
            *(
                (
                    ("-chardev", "stdio,mux=on,signal=off,id=console"),
                    ("-mon", "chardev=console,mode=readline"),
                )
                if self._attach_console
                else (
                    # In the cases that the console isn't directed to stdio
                    # then we write it to a log file instead.  Unfortunately,
                    # we also get a getty in our log file:
                    # https://github.com/systemd/systemd/issues/37928
                    ("-chardev", f"file,path={self._ipc}/console,id=console"),
                )
            ),
            *(("-drive", f"{drive},if=virtio,media=disk") for drive in drives),
            *(
                ("-nic", net.to_qemu() + ",model=virtio-net-pci")
                for net in self._networks
            ),
            # Credentials
            *(
                ("-smbios", f"type=11,value=io.systemd.credential:{k}={v}")
                for k, v in creds.items()
            ),
        )

        qemu = None
        try:
            self._ui.print_status("Waiting for guest")
            qemu = await self._ui.spawn(*args, stdin=None)
            returncode = await qemu.wait()
            if not self._shutdown_ok:
                raise SubprocessError(args, returncode)
        except asyncio.CancelledError:
            logger.debug("qemu task cancelled")
            if qemu is not None:
                logger.debug("Terminating qemu")
                qemu.terminate()
                try:
                    logger.debug("Waiting for qemu to quit")
                    await asyncio.shield(asyncio.wait_for(qemu.wait(), 5))
                except TimeoutError:
                    logger.debug("Timed out -- killing qemu")
                    qemu.kill()
                    await asyncio.shield(qemu.wait())
        finally:
            logger.debug("qemu exited")
            self._qemu_exited.set()

    async def _ssh_control(self) -> None:
        ssh = None
        try:
            assert self.ssh_direct_args is not None

            self._ui.print_status("ssh control socket: Connecting")

            control_socket = self._ipc / "ssh"

            args = (
                "ssh",
                *self.ssh_direct_args,
                ("-N", "-n"),  # no command, stdin disconnected
                ("-M", "-S", control_socket),  # listen on the control socket
                self.get_id(),  # unused, but shows up in messages
            )
            ssh = await self._ui.spawn(
                *args,
                stdin=asyncio.subprocess.DEVNULL,
                stdout=asyncio.subprocess.PIPE,
            )

            # ssh sends EOF after the connection succeeds
            assert ssh.stdout
            await ssh.stdout.read()

            # ..but that might have been because of an error, so check if the
            # control socket actually exists
            if not control_socket.exists():
                raise SubprocessError(args, await ssh.wait())

            # we're online!
            self._ssh_control_ready.set()
            self._ui.print_status("ssh control socket: Connected")

            returncode = await ssh.wait()
            if not self._shutdown_ok:
                raise SubprocessError(args, returncode)
        except asyncio.CancelledError:
            if ssh is not None:
                ssh.terminate()
                await asyncio.shield(ssh.wait())
        finally:
            # We try to reset our state best as possible here to deal with
            # reboots in the shutdown_ok case: we want the control socket
            # reestablished when the machine comes back.
            self._ssh_control_ready.clear()
            self._ssh_control_task = None

    async def __aenter__(self) -> Self:
        """Start the virtual machine."""
        await self._tasks.__aenter__()

        self.ssh_args = (
            ("-F", "none"),  # don't use the user's config
            ("-o", f"ControlPath={self._ipc}/ssh"),  # connect via the control socket
        )

        try:
            if self._identity is None:
                self._identity = await self._ssh_keygen()

            private, public = self._identity
            identities = [private]

            for nr, identity in enumerate(self._identities):
                if isinstance(identity, str):
                    path = self._ipc / f"id.{nr}"
                    with path.open("x") as extra:
                        extra.write(identity)
                    path.chmod(0o400)
                    identities.append(path)
                else:
                    identities.append(identity)

            self.ssh_direct_args = _ssh_direct_args(identities, self._ipc / "vsock")

            # It goes like this:
            #  - we start listening on the sd-notify socket
            #  - we start qemu
            #  - at some point the guest will notify that ssh is ready
            #    - this causes us to spawn the ssh control task
            #  - once connected, _ssh_control_ready gets set
            #  - we wait for that, so when it's done, we're done

            self._vsock_helper = await self._run_helper(
                "vhost-device-vsock",
                ("--socket", self._ipc / "vsock-device"),
                ("--uds-path", self._ipc / "vsock"),
                ready_when=self._ipc / "vsock-device",
            )

            server = await asyncio.start_unix_server(
                self._sd_notify_connection, self._ipc / "vsock_1111"
            )
            self._notify_server_task = self._tasks.create_task(server.serve_forever())

            self._qemu_task = self._tasks.create_task(
                self._qemu(port=1111, public=public)
            )

            # the notify socket server will create the ssh control socket task
            # which, in turn, sets this ready once it's online.
            try:
                await asyncio.wait_for(self._ssh_control_ready.wait(), self._timeout)
            except TimeoutError as exc:
                try:
                    log = (self._ipc / "console").read_text(errors="replace")
                except FileNotFoundError:
                    log = ""

                # Remove ANSI escapes and control characters
                log_lines = re.sub(
                    r"\x1b\[[ -?]*[@-~]|"  # CSI: ESC [ + params/interms + final
                    r"\x1b\][^\a\x1b]*|"  # OSC: ESC ] + everything to \a or ESC
                    r"\x1b[ ()O].|\x1b.|"  # two- and one-character escapes
                    r"[\x00-\b\v-\x1f\x7f]|",  # all control chars but [\t\n]
                    "",
                    log,
                ).splitlines()

                lines = [
                    f"Timed out waiting for the VM to start (after {self._timeout}s).",
                    "Console log:" if log_lines else "Console log unavailable.",
                    *filter(bool, log_lines),  # remove empty lines
                ]

                raise TimeoutError("\n".join(lines) + "\n") from exc

        except:
            exc_info = sys.exc_info()
            logger.debug("Startup failed: %r", exc_info)
            await self._tasks.__aexit__(*exc_info)
            raise
        else:
            logger.debug("Startup was successful")
            return self
        finally:
            logger.debug("Startup is done")
            self._ui.clear_status_message()

    async def __aexit__(
        self,
        et: type[BaseException] | None,
        exc: BaseException | None,
        tb: TracebackType | None,
    ) -> None:
        """Stop the virtual machine."""
        self._shutdown_ok = True

        try:
            if self._sit and exc is not None:
                current = asyncio.current_task()
                cancelled = current and current.cancelled()

                # This will report exceptions thrown from inside of the context managed
                # block but won't work properly in case we were cancelled due to an
                # error elsewhere (ie: qemu unexpectedly exiting) because at that point
                # the main task will have been cancelled and we won't be able to block
                # while waiting to read from stdin.  That's ok: the goal if this is to
                # catch test failures, not infra failures.
                if not cancelled:
                    await self._ui.sit(exc=exc)

            if self._ssh_control_task is not None:
                task = self._ssh_control_task
                task.cancel()
                await task  # Cancellation is async, so wait

            # If we're in snapshot mode then we don't have to do a clean shutdown
            with contextlib.suppress(FileNotFoundError):
                await self.qmp("quit" if self._snapshot else "system_powerdown")
            await self._qemu_exited.wait()

            assert self._vsock_helper is not None
            self._vsock_helper.cancel()

            assert self._notify_server_task is not None
            self._notify_server_task.cancel()

        finally:
            # If the TaskGroup fails (possibly by way of 'exc') and shutdown
            # also failed then the error from the shutdown will be suppressed.
            await self._tasks.__aexit__(et, exc, tb)
            if exc is not None:
                raise exc

    def get_id(self) -> str:
        """Get the machine identifier like `tt.0`, `tt.1`, etc."""
        return self._ipc.name

    async def wait_exit(self) -> None:
        """Wait for the VM to exit."""
        self._shutdown_ok = True
        await self._qemu_exited.wait()

    async def _ssh_cmd(self, *args: tuple[str | Path, ...]) -> None:
        await self._ui.run("ssh", *self.ssh_args, *args, self.get_id())

    async def forward_port(self, *args: tuple[str, ...]) -> None:
        """Set up a port forward.

        The `spec` is the format used by `ssh -L`, and looks something like
        `2222:127.0.0.1:22`.
        """
        return await self._ssh_cmd(("-O", "forward"), *args)

    async def cancel_port(self, *args: tuple[str, ...]) -> None:
        """Cancel a previous forward."""
        return await self._ssh_cmd(("-O", "cancel"), *args)

    async def wait_boot(self) -> None:
        """Wait for the machine to be fully-booted."""
        await self.execute("systemctl", "is-system-running", "--wait")

    async def execute(
        self,
        cmd: str,
        *args: str | GuestPath | tuple[str | GuestPath, ...],
        check: bool = True,
        direct: bool = False,
        input: bytes | str | None = b"",  # noqa:A002  # shadows `input()` but so does subprocess module
        environment: Mapping[str, str] = {},
        stdin: int | None = asyncio.subprocess.PIPE,
        stdout: int | None = asyncio.subprocess.PIPE,
    ) -> str:
        """Execute a command on the guest.

        If a single argument is given, it is expected to be a valid shell
        script.  If multiple arguments are given, they will interpreted as an
        argument vector and will be properly quoted before being sent to the guest.
        """
        if args:
            cmd = shlex.join(itertools.chain(*_normalize_args(cmd, *args)))

        assert self.ssh_direct_args is not None
        full_command = (
            "ssh",
            *(self.ssh_direct_args if direct else self.ssh_args),
            self.get_id(),  # unused, but shows up in messages
            ("--", "set -eu;"),
            *(f"export {k}={v and shlex.quote(v)};" for k, v in environment.items()),
            cmd,
        )

        ssh = await self._ui.spawn(*full_command, stdin=stdin, stdout=stdout)
        input_bytes = input.encode() if isinstance(input, str) else input
        output, _ = await ssh.communicate(input_bytes)
        returncode = await ssh.wait()
        if check and returncode != 0:
            raise SubprocessError(full_command, returncode, output)
        return output.decode() if output is not None else ""

    async def write(
        self,
        dest: str | GuestPath,
        content: str | bytes,
        *,
        mkdir: bool = True,
        owner: str | tuple[str | int | None, str | int | None] | None = None,
        perm: str | int | None = None,
    ) -> None:
        """Write a file into the test machine.

        Arguments:
            dest: The file name in the machine to write to
            content: Raw data to write to file
            append: If True, append to existing file instead of replacing it
            mkdir: if the parent directory should be created
            owner: If set, call chown on the file with the given owner string
            perm: Optional file permission as chmod shell string or integer

        """
        dest = GuestPath(dest, vm=self)

        if mkdir:
            await dest.parent.mkdir(parents=True)

        if isinstance(content, str):
            await dest.write_text(content)
        else:
            await dest.write_bytes(content)

        if owner is not None:
            await dest.chown(owner)

        if perm:
            await dest.chmod(perm)

    async def scp(self, *args: str | pathlib.Path, direct: bool = False) -> None:
        """Do a file transfer with scp.

        The hostname is ignored, so for paths on the guest use something like
        `vm:/tmp/path`.

        All arguments that are given in the form of `pathlib.Path` are passed to
        `scp` in absolute form, avoiding worries about `:` characters, but also
        making it impossible to use certain scp features (such as the special
        treatment of `.` — which pathlib collapses anyway).  Use string form if
        you need this, and worry about the escaping yourself.
        """
        assert self.ssh_direct_args is not None
        await self._ui.run(
            "scp",
            *(self.ssh_direct_args if direct else self.ssh_args),
            tuple(p.absolute() if isinstance(p, pathlib.Path) else p for p in args),
        )

    async def upload(
        self, *args: str | pathlib.Path, target_directory: str | GuestPath | None = None
    ) -> None:
        """Upload files to the guest.

        This works similarly to `cp --target-directory` (`-t`).

        All arguments are interpreted as local paths.  The target directory is
        a directory on the remote system (defaulting to root's home directory)
        which will be created (`mkdir -p`) if it doesn't exist.  Each argument
        is recursively copied into that directory by its basename.
        """
        if target_directory is not None:
            await self.execute("mkdir", "-p", target_directory)
        await self.scp(
            "-r", *map(pathlib.Path, args), f"{self.get_id()}:{target_directory or ''}"
        )

    @contextlib.asynccontextmanager
    async def disconnected(self) -> AsyncGenerator[None]:
        """Temporarily disconnect the control socket.

        On enter, disconnect the ssh control socket from the guest system.
        Inside of the block it's possible to perform commands that would
        otherwise result in the control socket being destroyed (which would be
        a hard error).

        On exit from the block, the connection is reestablished.

        The most obvious use-case for this is rebooting.
        """
        assert self._ssh_control_task is not None
        assert self._ssh_control_ready.is_set()
        self._ssh_control_ready.clear()
        self._ssh_control_task.cancel()
        self._ssh_control_task = None

        try:
            yield
        finally:
            await self._ssh_control_ready.wait()

    async def reboot(self) -> None:
        """Reboot the guest, waiting until it's back online."""
        async with self.disconnected():
            await self.qmp("system_reset")

    async def qmp(self, command: str) -> object:
        """Send a QMP command to the hypervisor.

        This can be used for things like modifying the hardware configuration.
        Don't power it off this way: the correct way to stop the VM is to exit
        the context manager.
        """
        return await _qmp_command(self._ipc, command)


class SubprocessError(Exception):
    """An exception thrown when a subprocess failed unexpectedly."""

    def __init__(
        self,
        args: tuple[str | Path | tuple[str | Path, ...], ...],
        returncode: int,
        output: bytes | None = None,
    ) -> None:
        """Create a SubprocessError instance.

        - args: the arguments to the command that failed
        - returncode: the non-zero return code
        """
        self.args = args
        self.returncode = returncode
        self.output = output

        if returncode < 0:
            msg = f"Subprocess terminated by {signal.Signals(-returncode).name}"
        else:
            msg = f"Subprocess exited unexpectedly with return code {returncode}:\n"

        if self.output:
            out = "\n🗯️  Output:\n\n" + self.output.decode(errors="replace")
        else:
            out = ""

        super().__init__(f"{msg}\n{_pretty_print_args(*args)}\n{out}\n")


def cleanup_on_signal() -> None:
    """Register SIGHUP and SIGTERM signal handlers to cleanly exit.

    This raises an exception, cleaning up running subprocesses and the IPC
    directory, in contrast to the default interpreter behaviour of a direct
    exit.
    """

    def _term(*args: object) -> Never:
        del args
        # This raises SystemExit which will bubble out of the handler
        sys.exit("I don't blame you.")

    signal.signal(signal.SIGHUP, _term)
    signal.signal(signal.SIGTERM, _term)


async def _ssh_properly_configured() -> bool:
    proc = None
    try:
        proc = await asyncio.subprocess.create_subprocess_exec(
            *("ssh", "-G", "tt.n"),
            stdout=asyncio.subprocess.PIPE,
        )
        stdout, _stderr = await proc.communicate()
    except OSError:
        return False
    else:
        return b"/test.thing/tt.n/ssh\n" in stdout
    finally:
        if proc is not None:
            await proc.wait()


async def _show_ssh_hints(ui: UI, vm_id: str) -> None:
    if not _stderr_is_tty():
        return

    if await _ssh_properly_configured():
        ui.print(f"\n🍓 VM running.  Connect with: \033[1mssh {vm_id}\033[0m\n")
    else:
        ui.print(f"""
Please consider adding this stanza to your SSH config:

Host tt.*
        ControlPath ${{XDG_RUNTIME_DIR}}/test.thing/%h/ssh

At which point you can connect to the VM using \033[1mssh {vm_id}\033[0m\n""")


@contextlib.contextmanager
def cli_helper() -> Iterator[None]:
    """Help use test.thing from CLI tools.

    This installs a signal handler for clean exit on SIGHUP and SIGTERM and
    catches SubprocessError and TimeoutError, printing a message to stderr and
    calling sys.exit().
    """
    cleanup_on_signal()
    try:
        yield
    except* (SubprocessError, TimeoutError) as eg:
        for exc in eg.exceptions:
            sys.stderr.write(f"\n🤦 {exc}\n")
        sys.exit("I'm sorry it didn't work out.")


def _main() -> None:
    class AppendTuple(argparse.Action):
        def __call__(
            self,
            parser: argparse.ArgumentParser,
            namespace: argparse.Namespace,
            values: str | Sequence[Any] | None,
            option_string: str | None = None,
        ) -> None:
            del parser
            fwds = getattr(namespace, self.dest) or ()
            fwds = (*fwds, (option_string, values))
            setattr(namespace, self.dest, fwds)

    parser = argparse.ArgumentParser(
        description="test.thing - a simple modern VM runner"
    )
    parser.add_argument(
        "--maintain", "-m", action="store_true", help="Changes are permanent"
    )
    parser.add_argument(
        "--attach", "-a", action="store_true", help="Attach to the VM console"
    )
    parser.add_argument(
        "--sit", action="store_true", help="Wait for enter key on exceptions"
    )
    parser.add_argument(
        "--debug", "-d", action="store_true", help="Enable debug output"
    )
    parser.add_argument(
        "--boot",
        choices=("efi", "mbr"),
        default="efi",
        help="How to boot the image (default: efi)",
    )
    parser.add_argument(
        "--verbose", "-v", action="store_true", help="Print verbose output"
    )
    parser.add_argument(
        "--ssh-key",
        "-i",
        type=Path,
        action="append",
        help="Path to SSH private key (default: generate)",
    )
    parser.add_argument(
        "--timeout", type=float, help="For startup, in seconds, or 'inf' (default: 30)"
    )
    parser.add_argument(
        "--no-network", action="store_true", help="Isolate the VM from the Internet"
    )
    parser.add_argument(
        "-L",
        "-R",
        "-D",
        default=[],
        dest="fwd_spec",
        action=AppendTuple,
        help="Setup an SSH-style port forward",
    )
    parser.add_argument(
        "--script",
        "-c",
        metavar="COMMAND",
        action="append",
        help="Execute this (shell-interpreted) command",
    )
    parser.add_argument(
        "--start-unit",
        "-s",
        metavar="UNIT",
        action="append",
        dest="script",
        type=(lambda s: f"systemctl enable --now {shlex.quote(s)}"),
        help="Start this systemd unit",
    )

    parser.add_argument("image", type=Path, help="The path to a qcow2 VM image to run")
    parser.add_argument("cmd", nargs="*")
    args = parser.parse_intermixed_args()

    async def _async_main() -> None:
        with cli_helper(), IpcDirectory() as ipc:
            ui = UI(status_messages=not args.attach, verbose=args.verbose)

            async with VirtualMachine(
                args.image,
                ipc=ipc,
                attach_console=args.attach,
                boot=args.boot,
                identities=args.ssh_key or (COCKPIT_TEST_IDENTITY,),
                networks=(() if args.no_network else (Network.user(),)),
                provision_ssh_key=not args.maintain,
                sit=args.sit,
                snapshot=not args.maintain,
                timeout=args.timeout,
                ui=ui,
            ) as vm:
                for spec in args.fwd_spec:
                    await vm.forward_port(spec)

                for cmd in args.script or ():
                    await vm.execute(cmd, stdout=None)

                if args.attach:
                    await vm.wait_exit()
                elif args.cmd:
                    await vm.execute(*args.cmd, stdin=None, stdout=None)
                else:
                    await _show_ssh_hints(ui, vm.get_id())
                    await ui.run("ssh", *vm.ssh_args, vm.get_id(), stdin=None)

    if args.debug:
        logging.basicConfig(level=logging.DEBUG)

    asyncio.run(_async_main(), debug=args.debug)


if __name__ == "__main__":
    _main()
