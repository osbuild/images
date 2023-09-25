package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"

	"github.com/osbuild/images/internal/cloud/awscloud"
)

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

func tearDown(aws *awscloud.AWS, res *resources) {
	fmt.Println("Tearing down")
	if res.InstanceID != nil {
		fmt.Printf("terminating instance %s\n", *res.InstanceID)
		if _, err := aws.TerminateInstanceEC2(res.InstanceID); err != nil {
			fmt.Fprintf(os.Stderr, "failed to terminate instance: %v\n", err)
		}
	}

	if res.SecurityGroup != nil {
		fmt.Printf("deleting security group %s\n", *res.SecurityGroup)
		if _, err := aws.DeleteSecurityGroupEC2(res.SecurityGroup); err != nil {
			fmt.Fprintf(os.Stderr, "cannot delete the security group: %v\n", err)
		}
	}

	if res.AMI != nil {
		fmt.Printf("deleting EC2 image %s and snapshot %s\n", *res.AMI, *res.Snapshot)
		if err := aws.DeleteEC2Image(res.AMI, res.Snapshot); err != nil {
			fmt.Fprintf(os.Stderr, "failed to deregister image: %v\n", err)
		}
	}

}

func uploadAndBoot() int {
	var accessKeyID string
	var secretAccessKey string
	var sessionToken string
	var region string
	var bucketName string
	var keyName string
	var filename string
	var imageName string
	var shareWith string
	var arch string
	var bootMode string
	var username string
	var sshKey string

	flag.StringVar(&accessKeyID, "access-key-id", "", "access key ID")
	flag.StringVar(&secretAccessKey, "secret-access-key", "", "secret access key")
	flag.StringVar(&sessionToken, "session-token", "", "session token")
	flag.StringVar(&region, "region", "", "target region")
	flag.StringVar(&bucketName, "bucket", "", "target S3 bucket name")
	flag.StringVar(&keyName, "key", "", "target S3 key name")
	flag.StringVar(&filename, "image", "", "image file to upload")
	flag.StringVar(&imageName, "name", "", "AMI name")
	flag.StringVar(&shareWith, "account-id", "", "account id to share image with")
	flag.StringVar(&arch, "arch", "", "arch (x86_64 or aarch64)")
	flag.StringVar(&bootMode, "boot-mode", "", "boot mode (legacy-bios, uefi, uefi-preferred)")
	flag.StringVar(&username, "username", "", "name of the user to create on the system")
	flag.StringVar(&sshKey, "ssh-key", "", "path to user's public ssh key")

	var resourcesFile string
	flag.StringVar(&resourcesFile, "tear-down", "", "path to resources file to use for tear-down")
	flag.Parse()

	a, err := awscloud.New(region, accessKeyID, secretAccessKey, sessionToken)
	if err != nil {
		fmt.Fprintf(os.Stderr, "awscloud.New() failed: %s\n", err.Error())
		return 1
	}

	res := &resources{}

	if len(resourcesFile) > 0 {
		resfile, err := os.Open(resourcesFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to open resources file: %s\n", err.Error())
			return 1
		}
		resdata, err := io.ReadAll(resfile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to read resources file: %s\n", err.Error())
			return 1
		}
		if err := json.Unmarshal(resdata, res); err != nil {
			fmt.Fprintf(os.Stderr, "failed to unmarshal resources data: %s\n", err.Error())
			return 1
		}
		defer tearDown(a, res)
		return 0
	}

	defer func() {
		resdata, err := json.MarshalIndent(res, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to marshal resources data: %s\n", err.Error())
			return
		}
		fmt.Println(string(resdata))
	}()

	userData, err := createUserData(username, sshKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "createUserData(): %s\n", err.Error())
		return 1
	}

	uploadOutput, err := a.Upload(filename, bucketName, keyName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Upload() failed: %s\n", err.Error())
		return 1
	}

	fmt.Printf("file uploaded to %s\n", aws.StringValue(&uploadOutput.Location))

	var share []string
	if shareWith != "" {
		share = append(share, shareWith)
	}

	var bootModePtr *string
	if bootMode != "" {
		bootModePtr = &bootMode
	}

	ami, snapshot, err := a.Register(imageName, bucketName, keyName, share, arch, bootModePtr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Register(): %s\n", err.Error())
		return 1
	}

	res.AMI = ami
	res.Snapshot = snapshot

	fmt.Printf("AMI registered: %s\n", aws.StringValue(ami))

	securityGroup, err := a.CreateSecurityGroupEC2("image-tests", "image-tests-security-group")
	if err != nil {
		fmt.Fprintf(os.Stderr, "CreateSecurityGroup(): %s\n", err.Error())
		return 1
	}

	res.SecurityGroup = securityGroup.GroupId

	_, err = a.AuthorizeSecurityGroupIngressEC2(securityGroup.GroupId, "0.0.0.0/0", 22, 22, "tcp")
	if err != nil {
		fmt.Fprintf(os.Stderr, "AuthorizeSecurityGroupIngressEC2(): %s\n", err.Error())
		return 1
	}

	runResult, err := a.RunInstanceEC2(ami, securityGroup.GroupId, userData, "t3.micro")
	if err != nil {
		fmt.Fprintf(os.Stderr, "RunInstanceEC2(): %s\n", err.Error())
		return 1
	}
	instanceID := runResult.Instances[0].InstanceId
	res.InstanceID = instanceID

	ip, err := a.GetInstanceAddress(instanceID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetInstanceAddress(): %s\n", err.Error())
		return 1
	}
	fmt.Printf("Instance %s is running and has IP address %s\n", *instanceID, ip)

	hostsfile := "/tmp/test_hosts"
	if err := keyscan(ip, hostsfile); err != nil {
		fmt.Fprintf(os.Stderr, "keyscan(): %s\n", err.Error())
		fmt.Printf("Press RETURN to clean up")
		bufio.NewReader(os.Stdin).ReadBytes('\n')
		return 1
	}

	key := strings.TrimSuffix(sshKey, ".pub")
	if err := sshCheck(ip, username, key, hostsfile); err != nil {
		fmt.Fprintf(os.Stderr, "sshCheck(): %s\n", err.Error())
		return 1
	}

	return 0
}

func run(c string, args ...string) ([]byte, []byte, error) {
	fmt.Printf("> %s %s\n", c, strings.Join(args, " "))
	cmd := exec.Command(c, args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, nil, err
	}

	cmdout, err := io.ReadAll(stdout)
	if err != nil {
		return nil, nil, err
	}

	cmderr, err := io.ReadAll(stderr)
	if err != nil {
		return nil, nil, err
	}

	err = cmd.Wait()
	if len(cmdout) > 0 {
		fmt.Println(string(cmdout))
	}
	if len(cmderr) > 0 {
		fmt.Fprintf(os.Stderr, string(cmderr)+"\n")
	}
	return cmdout, cmderr, err
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

func main() {
	os.Exit(uploadAndBoot())
}
