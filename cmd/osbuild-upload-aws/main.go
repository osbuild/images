package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"

	"github.com/osbuild/images/internal/cloud/awscloud"
)

func check(err error) {
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
		return "", fmt.Errorf("cannot read the public key: %v", err)
	}

	userData := fmt.Sprintf(`#cloud-config
user: %s
ssh_authorized_keys:
  - %s
`, username, string(publicKey))

	return userData, nil
}

func main() {
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
	flag.Parse()

	a, err := awscloud.New(region, accessKeyID, secretAccessKey, sessionToken)
	check(err)

	uploadOutput, err := a.Upload(filename, bucketName, keyName)
	check(err)

	fmt.Printf("file uploaded to %s\n", aws.StringValue(&uploadOutput.Location))

	var share []string
	if shareWith != "" {
		share = append(share, shareWith)
	}

	var bootModePtr *string
	if bootMode != "" {
		bootModePtr = &bootMode
	}

	ami, err := a.Register(imageName, bucketName, keyName, share, arch, bootModePtr)
	check(err)

	fmt.Printf("AMI registered: %s\n", aws.StringValue(ami))

	// TODO: defer deregister AMI

	userData, err := createUserData(username, sshKey)
	check(err)

	securityGroup, err := a.CreateSecurityGroupEC2("image-tests", "image-tests-security-group")
	check(err)

	defer func() {
		if _, err := a.DeleteSecurityGroupEC2(securityGroup.GroupId); err != nil {
			fmt.Fprintf(os.Stderr, "cannot delete the security group: %v\n", err)
		}
	}()

	_, err = a.AuthorizeSecurityGroupIngressEC2(securityGroup.GroupId, "0.0.0.0/0", 22, 22, "tcp")
	check(err)

	runResult, err := a.RunInstanceEC2(ami, securityGroup.GroupId, userData, "t3.micro")
	instanceID := runResult.Instances[0].InstanceId

	ip, err := a.GetInstanceAddress(instanceID)
	check(err)

	fmt.Printf("Instance %s is running and has IP address %s\n", *instanceID, ip)

	defer func() {
		if _, err := a.TerminateInstanceEC2(instanceID); err != nil {
			fmt.Fprintf(os.Stderr, "failed to terminate instance: %v\n", err)
		}
	}()

	fmt.Printf("Press RETURN to terminate instance")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}
