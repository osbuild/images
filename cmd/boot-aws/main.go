package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/osbuild/images/internal/cloud/awscloud"
)

// exitCheck can be deferred from the top of command functions to exit with an
// error code after any other defers are run in the same scope.
func exitCheck(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
		os.Exit(1)
	}
}

// createUserData creates cloud-init's user-data that contains user redhat with
// the specified public key
func createUserData(username, publicKeyFile string) (string, error) {
	publicKey, err := os.ReadFile(publicKeyFile)
	if err != nil {
		return "", err
	}

	userData := fmt.Sprintf(`#cloud-config
user: %s
ssh_authorized_keys:
  - %s
`, username, string(publicKey))

	return userData, nil
}

// resources created or allocated for an instance that can be cleaned up when
// tearing down.
type resources struct {
	AMI           *string `json:"ami,omitempty"`
	Snapshot      *string `json:"snapshot,omitempty"`
	SecurityGroup *string `json:"security-group,omitempty"`
	InstanceID    *string `json:"instance,omitempty"`
}

func run(c string, args ...string) ([]byte, []byte, error) {
	fmt.Printf("> %s %s\n", c, strings.Join(args, " "))
	cmd := exec.Command(c, args...)

	var cmdout, cmderr bytes.Buffer
	cmd.Stdout = &cmdout
	cmd.Stderr = &cmderr
	err := cmd.Run()
	if err != nil {
		return nil, nil, err
	}

	stdout := cmdout.Bytes()
	if len(stdout) > 0 {
		fmt.Println(string(stdout))
	}

	stderr := cmderr.Bytes()
	if len(stderr) > 0 {
		fmt.Fprintf(os.Stderr, string(stderr)+"\n")
	}
	return stdout, stderr, err
}

func sshCheck(ip, user, key, hostsfile string) error {
	_, _, err := run("ssh", "-i", key, "-o", fmt.Sprintf("UserKnownHostsFile=%s", hostsfile), "-l", user, ip, "rpm", "-qa")
	if err != nil {
		return err
	}

	_, _, err = run("ssh", "-i", key, "-o", fmt.Sprintf("UserKnownHostsFile=%s", hostsfile), "-l", user, ip, "cat", "/etc/os-release")
	if err != nil {
		return err
	}
	return nil
}

func keyscan(ip, filepath string) error {
	var keys []byte
	maxTries := 10
	var keyscanErr error
	for try := 0; try < maxTries; try++ {
		keys, _, keyscanErr = run("ssh-keyscan", ip)
		if keyscanErr == nil {
			break
		}
		time.Sleep(10 * time.Second)
	}
	if keyscanErr != nil {
		return keyscanErr
	}

	fmt.Printf("Creating known hosts file: %s\n", filepath)
	hostsFile, err := os.Create(filepath)
	if err != nil {
		return err
	}

	fmt.Printf("Writing to known hosts file: %s\n", filepath)
	if _, err := hostsFile.Write(keys); err != nil {
		return err
	}
	return nil
}

func newClientFromArgs(flags *pflag.FlagSet) (*awscloud.AWS, error) {
	region, err := flags.GetString("region")
	if err != nil {
		return nil, err
	}
	keyID, err := flags.GetString("access-key-id")
	if err != nil {
		return nil, err
	}
	secretKey, err := flags.GetString("secret-access-key")
	if err != nil {
		return nil, err
	}
	sessionToken, err := flags.GetString("session-token")
	if err != nil {
		return nil, err
	}

	return awscloud.New(region, keyID, secretKey, sessionToken)
}

func doSetup(a *awscloud.AWS, filename string, flags *pflag.FlagSet, res *resources) error {
	username, err := flags.GetString("username")
	if err != nil {
		return err
	}
	sshKey, err := flags.GetString("ssh-key")
	if err != nil {
		return err
	}

	userData, err := createUserData(username, sshKey)
	if err != nil {
		return fmt.Errorf("createUserData(): %s\n", err.Error())
	}

	bucketName, err := flags.GetString("bucket")
	if err != nil {
		return err
	}
	keyName, err := flags.GetString("key")
	if err != nil {
		return err
	}

	uploadOutput, err := a.Upload(filename, bucketName, keyName)
	if err != nil {
		return fmt.Errorf("Upload() failed: %s\n", err.Error())
	}

	fmt.Printf("file uploaded to %s\n", aws.StringValue(&uploadOutput.Location))

	var share []string
	if shareWith, err := flags.GetString("account-id"); shareWith != "" {
		share = append(share, shareWith)
	} else if err != nil {
		return err
	}

	var bootModePtr *string
	if bootMode, err := flags.GetString("boot-mode"); bootMode != "" {
		bootModePtr = &bootMode
	} else if err != nil {
		return err
	}

	imageName, err := flags.GetString("name")
	if err != nil {
		return err
	}

	arch, err := flags.GetString("arch")
	if err != nil {
		return err
	}

	ami, snapshot, err := a.Register(imageName, bucketName, keyName, share, arch, bootModePtr)
	if err != nil {
		return fmt.Errorf("Register(): %s\n", err.Error())
	}

	res.AMI = ami
	res.Snapshot = snapshot

	fmt.Printf("AMI registered: %s\n", aws.StringValue(ami))

	securityGroup, err := a.CreateSecurityGroupEC2("image-tests", "image-tests-security-group")
	if err != nil {
		return fmt.Errorf("CreateSecurityGroup(): %s\n", err.Error())
	}

	res.SecurityGroup = securityGroup.GroupId

	_, err = a.AuthorizeSecurityGroupIngressEC2(securityGroup.GroupId, "0.0.0.0/0", 22, 22, "tcp")
	if err != nil {
		return fmt.Errorf("AuthorizeSecurityGroupIngressEC2(): %s\n", err.Error())
	}

	runResult, err := a.RunInstanceEC2(ami, securityGroup.GroupId, userData, "t3.micro")
	if err != nil {
		return fmt.Errorf("RunInstanceEC2(): %s\n", err.Error())
	}
	instanceID := runResult.Instances[0].InstanceId
	res.InstanceID = instanceID

	ip, err := a.GetInstanceAddress(instanceID)
	if err != nil {
		return fmt.Errorf("GetInstanceAddress(): %s\n", err.Error())
	}
	fmt.Printf("Instance %s is running and has IP address %s\n", *instanceID, ip)

	hostsfile := "/tmp/test_hosts"
	if err := keyscan(ip, hostsfile); err != nil {
		return fmt.Errorf("keyscan(): %s\n", err.Error())
	}

	key := strings.TrimSuffix(sshKey, ".pub")
	if err := sshCheck(ip, username, key, hostsfile); err != nil {
		return fmt.Errorf("sshCheck(): %s\n", err.Error())
	}

	return nil
}

func setup(cmd *cobra.Command, args []string) {
	var err error
	defer func() { exitCheck(err) }()

	filename := args[0]
	flags := cmd.Flags()

	a, err := newClientFromArgs(flags)
	if err != nil {
		return
	}

	// collect resources into res and write them out when the function returns
	resourcesFile, err := flags.GetString("resourcefile")
	if err != nil {
		return
	}
	res := &resources{}
	defer func() {
		resdata, err := json.MarshalIndent(res, "", "  ")
		if err != nil {
			err = fmt.Errorf("failed to marshal resources data: %s\n", err.Error())
			return
		}
		resfile, err := os.Create(resourcesFile)
		if err != nil {
			err = fmt.Errorf("failed to create resources file: %s\n", err.Error())
			return
		}
		_, err = resfile.Write(resdata)
		if err != nil {
			err = fmt.Errorf("failed to write resources file: %s\n", err.Error())
			return
		}
		fmt.Printf("IDs for any newly created resources are stored in %s. Use the teardown command to clean them up.\n", resourcesFile)
		err = resfile.Close()
	}()

	err = doSetup(a, filename, flags, res)
}

func doTeardown(aws *awscloud.AWS, res *resources) error {
	if res.InstanceID != nil {
		fmt.Printf("terminating instance %s\n", *res.InstanceID)
		if _, err := aws.TerminateInstanceEC2(res.InstanceID); err != nil {
			return fmt.Errorf("failed to terminate instance: %v\n", err)
		}
	}

	if res.SecurityGroup != nil {
		fmt.Printf("deleting security group %s\n", *res.SecurityGroup)
		if _, err := aws.DeleteSecurityGroupEC2(res.SecurityGroup); err != nil {
			return fmt.Errorf("cannot delete the security group: %v\n", err)
		}
	}

	if res.AMI != nil {
		fmt.Printf("deleting EC2 image %s and snapshot %s\n", *res.AMI, *res.Snapshot)
		if err := aws.DeleteEC2Image(res.AMI, res.Snapshot); err != nil {
			return fmt.Errorf("failed to deregister image: %v\n", err)
		}
	}
	return nil
}

func teardown(cmd *cobra.Command, args []string) {
	var err error
	defer func() { exitCheck(err) }()

	flags := cmd.Flags()

	a, err := newClientFromArgs(flags)
	if err != nil {
		return
	}

	resourcesFile, err := flags.GetString("resourcefile")
	if err != nil {
		return
	}

	res := &resources{}

	resfile, err := os.Open(resourcesFile)
	if err != nil {
		err = fmt.Errorf("failed to open resources file: %s\n", err.Error())
		return
	}
	resdata, err := io.ReadAll(resfile)
	if err != nil {
		err = fmt.Errorf("failed to read resources file: %s\n", err.Error())
		return
	}
	if err := json.Unmarshal(resdata, res); err != nil {
		err = fmt.Errorf("failed to unmarshal resources data: %s\n", err.Error())
		return
	}

	err = doTeardown(a, res)
}

func setupCLI() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:                   "boot",
		Long:                  "upload and boot an image to the appropriate cloud provider",
		DisableFlagsInUseLine: true,
	}

	rootFlags := rootCmd.PersistentFlags()
	rootFlags.String("access-key-id", "", "access key ID")
	rootFlags.String("secret-access-key", "", "secret access key")
	rootFlags.String("session-token", "", "session token")
	rootFlags.String("region", "", "target region")
	rootFlags.String("bucket", "", "target S3 bucket name")
	rootFlags.String("key", "", "target S3 key name")
	rootFlags.String("name", "", "AMI name")
	rootFlags.String("account-id", "", "account id to share image with")
	rootFlags.String("arch", "", "arch (x86_64 or aarch64)")
	rootFlags.String("boot-mode", "", "boot mode (legacy-bios, uefi, uefi-preferred)")
	rootFlags.String("username", "", "name of the user to create on the system")
	rootFlags.String("ssh-key", "", "path to user's public ssh key")

	exitCheck(rootCmd.MarkPersistentFlagRequired("access-key-id"))
	exitCheck(rootCmd.MarkPersistentFlagRequired("secret-access-key"))
	exitCheck(rootCmd.MarkPersistentFlagRequired("region"))
	exitCheck(rootCmd.MarkPersistentFlagRequired("bucket"))

	// TODO: make it optional and use UUID if not specified
	exitCheck(rootCmd.MarkPersistentFlagRequired("key"))

	// TODO: make it optional and use UUID if not specified
	exitCheck(rootCmd.MarkPersistentFlagRequired("name"))

	exitCheck(rootCmd.MarkPersistentFlagRequired("arch"))

	// TODO: make it optional and use a default
	exitCheck(rootCmd.MarkPersistentFlagRequired("username"))

	// TODO: make ssh key optional for 'run' and if not specified generate a
	// temporary key pair
	exitCheck(rootCmd.MarkPersistentFlagRequired("ssh-key"))

	setupCmd := &cobra.Command{
		Use:                   "setup [--resourcefile <filename>] <filename>",
		Short:                 "upload and boot an image and save the created resource IDs to a file for later teardown",
		Args:                  cobra.ExactArgs(1),
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

	return rootCmd
}

func main() {
	cmd := setupCLI()
	cmd.Execute()
}
