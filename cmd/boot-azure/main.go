package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/osbuild/images/internal/test"
	"github.com/osbuild/images/pkg/arch"
	"github.com/osbuild/images/pkg/cloud/azure"
)

const (
	StorageContainer = "images"
)

// exitCheck can be deferred from the top of command functions to exit with an
// error code after any other defers are run in the same scope.
func exitCheck(err error) {
	if err != nil {
		fmt.Fprint(os.Stderr, err.Error()+"\n")
		os.Exit(1)
	}
}

// resources created or allocated for an instance that can be cleaned up when
// tearing down.
type resources struct {
	VM        *azure.VM           `json:"vm,omitempty"`
	GI        *azure.GalleryImage `json:"galleryimage,omitempty"`
	BlobName  *string             `json:"blob,omitempty"`
	ImageName *string             `json:"image,omitempty"`
}

func newClientFromArgs(flags *pflag.FlagSet) (*azure.Client, error) {
	client, err := flags.GetString("client-id")
	if err != nil {
		return nil, err
	}
	secret, err := flags.GetString("client-secret")
	if err != nil {
		return nil, err
	}
	tenant, err := flags.GetString("tenant")
	if err != nil {
		return nil, err
	}
	subscr, err := flags.GetString("subscription")
	if err != nil {
		return nil, err
	}

	return azure.NewClient(
		azure.Credentials{
			ClientID:     client,
			ClientSecret: secret,
		},
		tenant,
		subscr,
	)
}

func getDefaultSize(architecture string) (string, error) {
	switch architecture {
	case "x86_64":
		return "Standard_DS1_v2", nil
	case "aarch64":
		return "Standard_D2pls_v5", nil
	default:
		return "", fmt.Errorf("getDefaultSize(): unknown architecture %q", architecture)
	}
}

func upload(ac *azure.Client, subscription, resourceGroup, localImage, remoteName, architecture string, res *resources) (string, error) {
	ctx := context.Background()
	location, err := ac.GetResourceGroupLocation(ctx, resourceGroup)
	if err != nil {
		return "", err
	}

	staccTag := azure.Tag{
		Name:  "imagesStorageAccount",
		Value: fmt.Sprintf("location=%s", location),
	}
	stacc, err := ac.GetResourceNameByTag(
		ctx,
		resourceGroup,
		staccTag,
	)
	if err != nil {
		return "", err
	}

	if stacc == "" {
		stacc = azure.RandomStorageAccountName("images")
		err = ac.CreateStorageAccount(ctx, resourceGroup, stacc, "", staccTag)
		if err != nil {
			return "", err
		}
	}

	storekey, err := ac.GetStorageAccountKey(ctx, resourceGroup, stacc)
	if err != nil {
		return "", err
	}

	storeClient, err := azure.NewStorageClient(stacc, storekey)
	if err != nil {
		return "", err
	}

	err = storeClient.CreateStorageContainerIfNotExist(ctx, stacc, StorageContainer)
	if err != nil {
		return "", err
	}
	blobName := azure.EnsureVHDExtension(remoteName)
	err = storeClient.UploadPageBlob(
		azure.BlobMetadata{
			StorageAccount: stacc,
			ContainerName:  StorageContainer,
			BlobName:       blobName,
		},
		localImage,
		azure.DefaultUploadThreads,
	)
	if err != nil {
		return "", err
	}
	res.BlobName = &blobName

	switch architecture {
	case "x86_64":
		err = ac.RegisterImage(
			ctx,
			resourceGroup,
			stacc,
			StorageContainer,
			blobName,
			remoteName,
			"",
			azure.HyperVGenV2,
		)
		if err != nil {
			return "", err
		}
		res.ImageName = &remoteName
		image := fmt.Sprintf(
			"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/images/%s",
			subscription, resourceGroup, remoteName)
		return image, nil
	case "aarch64":
		gi, err := ac.RegisterGalleryImage(
			ctx,
			resourceGroup,
			stacc,
			StorageContainer,
			blobName,
			remoteName,
			"",
			azure.HyperVGenV2,
			arch.ARCH_AARCH64,
		)
		if err != nil {
			return "", err
		}
		res.GI = gi
		return res.GI.ImageRef, nil
	default:
		return "", fmt.Errorf("upload(): unknown architecture %q", architecture)
	}
}

func doSetup(ac *azure.Client, flags *pflag.FlagSet, localImage string, res *resources) error {
	rg, err := flags.GetString("resource-group")
	if err != nil {
		return err
	}

	image, err := flags.GetString("image")
	if err != nil {
		return err
	}

	architecture, err := flags.GetString("arch")
	if err != nil {
		return err
	}

	if localImage != "" {
		subscription, err := flags.GetString("subscription")
		if err != nil {
			return err
		}
		remoteImageName, err := flags.GetString("image-name")
		if err != nil {
			return err
		}
		image, err = upload(ac, subscription, rg, localImage, remoteImageName, architecture, res)
		if err != nil {
			return err
		}
	}

	vmName, err := flags.GetString("vm-name")
	if err != nil {
		return err
	}

	size, err := flags.GetString("size")
	if err != nil {
		return err
	}
	if size == "" {
		size, err = getDefaultSize(architecture)
		if err != nil {
			return err
		}
	}

	username, err := flags.GetString("username")
	if err != nil {
		return err
	}

	keyPath, err := flags.GetString("ssh-pubkey")
	if err != nil {
		return err
	}

	keyfile, err := os.Open(keyPath)
	if err != nil {
		return err
	}
	defer func() {
		if err := keyfile.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "unable to close public key file: %s\n", err.Error())
		}
	}()

	keyData, err := io.ReadAll(keyfile)
	if err != nil {
		return err
	}

	ctx := context.Background()
	vm, err := ac.CreateVM(
		ctx,
		rg,
		image,
		vmName,
		size,
		username,
		string(keyData),
	)
	if err != nil {
		return err
	}
	res.VM = vm
	return nil
}

func setup(cmd *cobra.Command, args []string) {
	var fnerr error
	defer func() { exitCheck(fnerr) }()

	flags := cmd.Flags()

	ac, err := newClientFromArgs(flags)
	if err != nil {
		fnerr = err
		return
	}

	localImage := ""
	if len(args) == 1 {
		localImage = args[0]
	}

	res := &resources{}
	fnerr = doSetup(ac, flags, localImage, res)
	if fnerr != nil {
		fmt.Fprintf(os.Stderr, "setup() failed: %s\n", fnerr.Error())
		fmt.Fprint(os.Stderr, "tearing down resources\n")

		rg, err := flags.GetString("resource-group")
		if err != nil {
			fnerr = fmt.Errorf("failed to get resource group to tear down resources: %s", err.Error())
			return
		}

		if err := doTeardown(ac, rg, res); err != nil {
			fnerr = fmt.Errorf("failed to tear down resources: %s", err.Error())
			return
		}
	}

	resourcesFile, err := flags.GetString("resourcefile")
	if err != nil {
		fnerr = err
		return
	}
	resfile, err := os.Create(resourcesFile)
	if err != nil {
		fnerr = err
		return
	}
	defer func() {
		if err := resfile.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "unable to close resources file: %s\n", err.Error())
		}
	}()

	resdata, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		fnerr = err
		return
	}
	_, err = resfile.Write(resdata)
	if err != nil {
		fnerr = err
		return
	}
}

func doTeardown(ac *azure.Client, resourceGroup string, res *resources) error {
	ctx := context.Background()

	if res.VM != nil {
		if err := ac.DestroyVM(ctx, res.VM); err != nil {
			return err
		}
	}

	if res.GI != nil {
		if err := ac.DeleteGalleryImage(ctx, res.GI); err != nil {
			return err
		}
	}

	if res.ImageName != nil {
		if err := ac.DeleteImage(ctx, resourceGroup, *res.ImageName); err != nil {
			return err
		}
	}

	if res.BlobName == nil {
		return nil
	}

	location, err := ac.GetResourceGroupLocation(ctx, resourceGroup)
	if err != nil {
		return err
	}

	staccTag := azure.Tag{
		Name:  "imagesStorageAccount",
		Value: fmt.Sprintf("location=%s", location),
	}
	stacc, err := ac.GetResourceNameByTag(
		ctx,
		resourceGroup,
		staccTag,
	)
	if err != nil {
		return err
	}

	// storage account no longer exists, so assume everything is gone
	if stacc == "" {
		return nil
	}

	storekey, err := ac.GetStorageAccountKey(ctx, resourceGroup, stacc)
	if err != nil {
		return err
	}

	storeClient, err := azure.NewStorageClient(stacc, storekey)
	if err != nil {
		return err
	}

	if err := storeClient.DeleteBlob(ctx, azure.BlobMetadata{
		StorageAccount: stacc,
		ContainerName:  StorageContainer,
		BlobName:       *res.BlobName,
	}); err != nil {
		return err
	}
	return nil
}

func teardown(cmd *cobra.Command, args []string) {
	var fnerr error
	defer func() { exitCheck(fnerr) }()

	flags := cmd.Flags()
	ac, err := newClientFromArgs(flags)
	if err != nil {
		fnerr = err
		return
	}

	resourcesFile, err := flags.GetString("resourcefile")
	if err != nil {
		return
	}

	res := &resources{}
	resfile, err := os.Open(resourcesFile)
	if err != nil {
		fnerr = fmt.Errorf("failed to open resources file: %s", err.Error())
		return
	}
	defer func() {
		if err := resfile.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "unable to close resources file: %s\n", err.Error())
		}
	}()

	resdata, err := io.ReadAll(resfile)
	if err != nil {
		fnerr = fmt.Errorf("failed to read resources file: %s", err.Error())
		return
	}
	if err := json.Unmarshal(resdata, res); err != nil {
		fnerr = fmt.Errorf("failed to unmarshal resources data: %s", err.Error())
		return
	}

	rg, err := flags.GetString("resource-group")
	if err != nil {
		fnerr = fmt.Errorf("failed to get resource group: %s", err.Error())
		return
	}

	fnerr = doTeardown(ac, rg, res)
}

func doRunExec(ac *azure.Client, command []string, flags *pflag.FlagSet, res *resources) error {
	privKey, err := flags.GetString("ssh-privkey")
	if err != nil {
		return err
	}

	username, err := flags.GetString("username")
	if err != nil {
		return err
	}

	tmpdir, err := os.MkdirTemp("", "boot-test-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpdir)

	hostsfile := filepath.Join(tmpdir, "known_hosts")
	ip := res.VM.IPAddress
	if err := test.Keyscan(ip, hostsfile); err != nil {
		return err
	}

	// ssh into the remote machine and exit immediately to check connection
	if err := test.SshRun(ip, username, privKey, hostsfile, "exit"); err != nil {
		return err
	}

	isFile := func(path string) bool {
		fileInfo, err := os.Stat(path)
		if err != nil {
			// ignore error and assume it's not a path
			return false
		}

		// Check if it's a regular file
		return fileInfo.Mode().IsRegular()
	}

	// copy every argument that is a file to the remote host (basename only)
	// and construct remote command
	// NOTE: this wont work with directories or with multiple args in different
	// paths that share the same basename - it's very limited
	remoteCommand := make([]string, len(command))
	for idx := range command {
		arg := command[idx]
		if isFile(arg) {
			// scp the file and add it to the remote command by its base name
			remotePath := filepath.Base(arg)
			remoteCommand[idx] = remotePath
			if err := test.ScpFile(ip, username, privKey, hostsfile, arg, remotePath); err != nil {
				return err
			}
		} else {
			// not a file: add the arg as is
			remoteCommand[idx] = arg
		}
	}

	// add ./ to first element for the executable
	remoteCommand[0] = fmt.Sprintf("./%s", remoteCommand[0])

	// run the executable
	return test.SshRun(ip, username, privKey, hostsfile, remoteCommand...)
}

func runExec(cmd *cobra.Command, args []string) {
	var fnerr error
	defer func() { exitCheck(fnerr) }()
	image := args[0]

	command := args[1:]
	flags := cmd.Flags()

	ac, fnerr := newClientFromArgs(flags)
	if fnerr != nil {
		return
	}

	res := &resources{}
	defer func() {
		rg, err := flags.GetString("resource-group")
		if err != nil {
			fnerr = fmt.Errorf("failed to get resource group to tear down resources: %s", err.Error())
			return
		}
		if err := doTeardown(ac, rg, res); err != nil {
			fnerr = fmt.Errorf("failed to destroy vm: %s", err.Error())
			return
		}
	}()

	fnerr = doSetup(ac, flags, image, res)
	if fnerr != nil {
		return
	}

	fnerr = doRunExec(ac, command, flags, res)
}

func setupCLI() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:                   "boot",
		Long:                  "upload and boot an image to the appropriate cloud provider",
		DisableFlagsInUseLine: true,
	}

	rootFlags := rootCmd.PersistentFlags()
	rootFlags.String("client-id", "", "client ID")
	rootFlags.String("client-secret", "", "client secret")
	rootFlags.String("tenant", "", "tenant id")
	rootFlags.String("subscription", "", "subscription")
	rootFlags.String("resource-group", "", "resource group of image and vm")
	rootFlags.String("username", "azure", "name of the user to create on the system")
	rootFlags.String("ssh-pubkey", "", "path to user's public ssh key, must be an rsa key")
	rootFlags.String("ssh-privkey", "", "path to user's private ssh key")
	rootFlags.String("image", "", "full resource ID of the remote image name, this should already exist in the resource group, if no local image is provided")
	rootFlags.String("vm-name", "vm-name", "name of the VM to create, all dependencies will be prefixed with this name")
	rootFlags.String("image-name", "image-name", "the image and blob will")
	rootFlags.String("size", "", "size or instance type of the VM to create")
	rootFlags.String("arch", "x86_64", "size or instance type of the VM to create")

	exitCheck(rootCmd.MarkPersistentFlagRequired("client-id"))
	exitCheck(rootCmd.MarkPersistentFlagRequired("client-secret"))
	exitCheck(rootCmd.MarkPersistentFlagRequired("tenant"))
	exitCheck(rootCmd.MarkPersistentFlagRequired("subscription"))

	setupCmd := &cobra.Command{
		Use:                   "setup [--resourcefile <filename>] <filename>",
		Short:                 "upload and boot an image and save the created resource IDs to a file for later teardown",
		Args:                  cobra.MaximumNArgs(1),
		Run:                   setup,
		DisableFlagsInUseLine: true,
	}
	setupCmd.Flags().StringP("resourcefile", "r", "resources.json", "path to store the resource IDs")
	rootCmd.AddCommand(setupCmd)

	teardownCmd := &cobra.Command{
		Use:   "teardown [--resourcefile <filename>]",
		Short: "teardown (clean up) all the resources specified in a resources file created by a previous 'setup' call",
		Args:  cobra.NoArgs,
		Run:   teardown,
	}
	teardownCmd.Flags().StringP("resourcefile", "r", "resources.json", "path to store the resource IDs")
	rootCmd.AddCommand(teardownCmd)

	runCmd := &cobra.Command{
		Use:   "run <image> <executable>...",
		Short: "upload and boot an image, then upload the specified executable and run it on the remote host",
		Long:  "upload and boot an image on Azure, then upload the executable file specified by the second positional argument and execute it via SSH with the args on the command line",
		Args:  cobra.MinimumNArgs(2),
		Run:   runExec,
	}
	rootCmd.AddCommand(runCmd)

	return rootCmd
}

func main() {
	cmd := setupCLI()
	exitCheck(cmd.Execute())
}
