package awscloud

import (
	"errors"
	"fmt"
	"io"
	"slices"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/google/uuid"

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/arch"
	"github.com/osbuild/images/pkg/cloud"
	"github.com/osbuild/images/pkg/platform"
)

type AwsUploader struct {
	client awsClient

	region     string
	bucketName string
	imageName  string
	targetArch string
	bootMode   *string

	resultAmi string
}

type UploaderOptions struct {
	TargetArch string
	// BootMode to set for the AMI. If nil, no explicit boot mode will be set.
	BootMode *platform.BootMode
}

func (ou *UploaderOptions) ec2BootMode() (*string, error) {
	if ou == nil || ou.BootMode == nil {
		return nil, nil
	}

	switch *ou.BootMode {
	case platform.BOOT_LEGACY:
		return common.ToPtr(ec2.BootModeValuesLegacyBios), nil
	case platform.BOOT_UEFI:
		return common.ToPtr(ec2.BootModeValuesUefi), nil
	case platform.BOOT_HYBRID:
		return common.ToPtr(ec2.BootModeValuesUefiPreferred), nil
	default:
		return nil, fmt.Errorf("invalid boot mode: %s", ou.BootMode)
	}
}

// testing support
type awsClient interface {
	Regions() ([]string, error)
	Buckets() ([]string, error)
	CheckBucketPermission(string, S3Permission) (bool, error)
	UploadFromReader(io.Reader, string, string) (*s3manager.UploadOutput, error)
	Register(name, bucket, key string, shareWith []string, rpmArch string, bootMode, importRole *string) (*string, *string, error)
	DeleteObject(string, string) error
}

var newAwsClient = func(region string) (awsClient, error) {
	return NewDefault(region)
}

func NewUploader(region, bucketName, imageName string, opts *UploaderOptions) (*AwsUploader, error) {
	if opts == nil {
		opts = &UploaderOptions{}
	}
	bootMode, err := opts.ec2BootMode()
	if err != nil {
		return nil, err
	}
	client, err := newAwsClient(region)
	if err != nil {
		return nil, err
	}

	return &AwsUploader{
		client:     client,
		region:     region,
		bucketName: bucketName,
		imageName:  imageName,
		targetArch: opts.TargetArch,
		bootMode:   bootMode,
	}, nil
}

var _ cloud.Uploader = &AwsUploader{}

func (au *AwsUploader) Check(status io.Writer) error {
	fmt.Fprintf(status, "Checking AWS region access...\n")
	regions, err := au.client.Regions()
	if err != nil {
		return fmt.Errorf("retrieving AWS regions for '%s' failed: %w", au.region, err)
	}

	if !slices.Contains(regions, au.region) {
		return fmt.Errorf("given AWS region '%s' not found", au.region)
	}

	fmt.Fprintf(status, "Checking AWS bucket...\n")
	buckets, err := au.client.Buckets()
	if err != nil {
		return fmt.Errorf("retrieving AWS list of buckets failed: %w", err)
	}
	if !slices.Contains(buckets, au.bucketName) {
		return fmt.Errorf("bucket '%s' not found in the given AWS account", au.bucketName)
	}

	fmt.Fprintf(status, "Checking AWS bucket permissions...\n")
	writePermission, err := au.client.CheckBucketPermission(au.bucketName, S3PermissionWrite)
	if err != nil {
		return err
	}
	if !writePermission {
		return fmt.Errorf("you don't have write permissions to bucket '%s' with the given AWS account", au.bucketName)
	}
	fmt.Fprintf(status, "Upload conditions met.\n")
	return nil
}

func (au *AwsUploader) UploadAndRegister(r io.Reader, status io.Writer) (err error) {
	keyName := fmt.Sprintf("%s-%s", uuid.New().String(), au.imageName)
	fmt.Fprintf(status, "Uploading %s to %s:%s\n", au.imageName, au.bucketName, keyName)

	res, err := au.client.UploadFromReader(r, au.bucketName, keyName)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			aErr := au.client.DeleteObject(au.bucketName, keyName)
			fmt.Fprintf(status, "Deleted S3 object %s:%s\n", au.bucketName, keyName)
			err = errors.Join(err, aErr)
		}
	}()
	fmt.Fprintf(status, "File uploaded to %s\n", aws.StringValue(&res.Location))
	if au.targetArch == "" {
		au.targetArch = arch.Current().String()
	}

	fmt.Fprintf(status, "Registering AMI %s\n", au.imageName)
	ami, snapshot, err := au.client.Register(au.imageName, au.bucketName, keyName, nil, au.targetArch, au.bootMode, nil)
	if err != nil {
		return err
	}

	fmt.Fprintf(status, "Deleted S3 object %s:%s\n", au.bucketName, keyName)
	if err := au.client.DeleteObject(au.bucketName, keyName); err != nil {
		return err
	}
	fmt.Fprintf(status, "AMI registered: %s\nSnapshot ID: %s\n", aws.StringValue(ami), aws.StringValue(snapshot))
	if err != nil {
		return err
	}

	if ami == nil {
		return fmt.Errorf("internal error: ami registered but nil ami string")
	}
	au.resultAmi = *ami

	return nil
}

func (au *AwsUploader) Result() (ami, region string, err error) {
	if au.resultAmi == "" {
		return "", "", fmt.Errorf("no successful upload found")
	}
	return au.resultAmi, au.region, nil
}
