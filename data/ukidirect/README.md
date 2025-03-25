# Kernel install drop-in file for UKI

The 99-uki-uefi-setup.install script, prior to v25.3 of the uki-direct package, would run `bootctl -p` to discover the ESP [1]. This doesn't work in osbuild because the system isn't booted or live. Since v25.3, the install script respects the `$BOOT_ROOT` env var that we set in osbuild during the org.osbuild.rpm stage.

The install script in this directory bundles the version of the script found in v25.3 so it can be embedded in images that install an older version of the package.

The updated package is expected to be released in RHEL 9.7 and 10.1.

[1] https://gitlab.com/kraxel/virt-firmware/-/commit/ca385db4f74a4d542455b9d40c91c8448c7be90c
