package awscloud

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"slices"

	"github.com/aws/aws-sdk-go-v2/config"
	credentialsv2 "github.com/aws/aws-sdk-go-v2/credentials"
	s3manager "github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/osbuild/images/pkg/olog"
)

type AWS struct {
	ec2        *ec2.EC2
	s3         s3Client
	s3uploader s3Uploader
	s3presign  s3Presign
}

// S3PermissionsMatrix Maps a requested permission to all permissions that are sufficient for the requested one
var S3PermissionsMatrix = map[s3types.Permission][]s3types.Permission{
	s3types.PermissionRead:        {s3types.PermissionRead, s3types.PermissionWrite, s3types.PermissionFullControl},
	s3types.PermissionWrite:       {s3types.PermissionWrite, s3types.PermissionFullControl},
	s3types.PermissionFullControl: {s3types.PermissionFullControl},
	s3types.PermissionReadAcp:     {s3types.PermissionReadAcp, s3types.PermissionWriteAcp},
	s3types.PermissionWriteAcp:    {s3types.PermissionWriteAcp},
}

// Create a new session from the credentials and the region and returns an *AWS object initialized with it.
func newAwsFromCreds(creds *credentials.Credentials, region string) (*AWS, error) {
	// Create a Session with a custom region
	sess, err := session.NewSession(&aws.Config{
		Credentials: creds,
		Region:      aws.String(region),
	})
	if err != nil {
		return nil, err
	}

	credsValue, err := creds.Get()
	if err != nil {
		return nil, err
	}
	cfg, err := config.LoadDefaultConfig(
		context.TODO(),
		config.WithRegion(region),
		config.WithCredentialsProvider(credentialsv2.NewStaticCredentialsProvider(
			credsValue.AccessKeyID,
			credsValue.SecretAccessKey,
			credsValue.SessionToken,
		)),
	)
	if err != nil {
		return nil, err
	}

	s3cli := s3.NewFromConfig(cfg)

	return &AWS{
		ec2:        ec2.New(sess),
		s3:         s3cli,
		s3uploader: s3manager.NewUploader(s3cli),
		s3presign:  s3.NewPresignClient(s3cli),
	}, nil
}

// Initialize a new AWS object from individual bits. SessionToken is optional
func New(region string, accessKeyID string, accessKey string, sessionToken string) (*AWS, error) {
	return newAwsFromCreds(credentials.NewStaticCredentials(accessKeyID, accessKey, sessionToken), region)
}

// Initializes a new AWS object with the credentials info found at filename's location.
// The credential files should match the AWS format, such as:
// [default]
// aws_access_key_id = secretString1
// aws_secret_access_key = secretString2
//
// If filename is empty the underlying function will look for the
// "AWS_SHARED_CREDENTIALS_FILE" env variable or will default to
// $HOME/.aws/credentials.
func NewFromFile(filename string, region string) (*AWS, error) {
	return newAwsFromCreds(credentials.NewSharedCredentials(filename, "default"), region)
}

// Initialize a new AWS object from defaults.
// Looks for env variables, shared credential file, and EC2 Instance Roles.
func NewDefault(region string) (*AWS, error) {
	return newAwsFromCreds(nil, region)
}

// Create a new session from the credentials and the region and returns an *AWS object initialized with it.
func newAwsFromCredsWithEndpoint(creds *credentials.Credentials, region, endpoint, caBundle string, skipSSLVerification bool) (*AWS, error) {
	// Create a Session with a custom region
	s3ForcePathStyle := true
	sessionOptions := session.Options{
		Config: aws.Config{
			Credentials:      creds,
			Region:           aws.String(region),
			Endpoint:         &endpoint,
			S3ForcePathStyle: &s3ForcePathStyle,
		},
	}

	credsValue, err := creds.Get()
	if err != nil {
		return nil, err
	}
	v2OptionFuncs := []func(*config.LoadOptions) error{
		config.WithRegion(region),
		config.WithCredentialsProvider(credentialsv2.NewStaticCredentialsProvider(
			credsValue.AccessKeyID,
			credsValue.SecretAccessKey,
			credsValue.SessionToken,
		)),
	}

	if caBundle != "" {
		caBundleReader, err := os.Open(caBundle)
		if err != nil {
			return nil, err
		}
		defer caBundleReader.Close()
		sessionOptions.CustomCABundle = caBundleReader
		v2OptionFuncs = append(v2OptionFuncs, config.WithCustomCABundle(caBundleReader))
	}

	if skipSSLVerification {
		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} // #nosec G402
		sessionOptions.Config.HTTPClient = &http.Client{
			Transport: transport,
		}
		v2OptionFuncs = append(v2OptionFuncs, config.WithHTTPClient(&http.Client{
			Transport: transport,
		}))
	}

	sess, err := session.NewSessionWithOptions(sessionOptions)
	if err != nil {
		return nil, err
	}

	cfg, err := config.LoadDefaultConfig(
		context.TODO(),
		v2OptionFuncs...,
	)
	if err != nil {
		return nil, err
	}

	s3cli := s3.NewFromConfig(cfg, func(options *s3.Options) {
		options.BaseEndpoint = aws.String(endpoint)
		options.UsePathStyle = true
	})

	return &AWS{
		ec2:        ec2.New(sess),
		s3:         s3cli,
		s3uploader: s3manager.NewUploader(s3cli),
		s3presign:  s3.NewPresignClient(s3cli),
	}, nil
}

// Initialize a new AWS object targeting a specific endpoint from individual bits. SessionToken is optional
func NewForEndpoint(endpoint, region, accessKeyID, accessKey, sessionToken, caBundle string, skipSSLVerification bool) (*AWS, error) {
	return newAwsFromCredsWithEndpoint(credentials.NewStaticCredentials(accessKeyID, accessKey, sessionToken), region, endpoint, caBundle, skipSSLVerification)
}

// Initializes a new AWS object targeting a specific endpoint with the credentials info found at filename's location.
// The credential files should match the AWS format, such as:
// [default]
// aws_access_key_id = secretString1
// aws_secret_access_key = secretString2
//
// If filename is empty the underlying function will look for the
// "AWS_SHARED_CREDENTIALS_FILE" env variable or will default to
// $HOME/.aws/credentials.
func NewForEndpointFromFile(filename, endpoint, region, caBundle string, skipSSLVerification bool) (*AWS, error) {
	return newAwsFromCredsWithEndpoint(credentials.NewSharedCredentials(filename, "default"), region, endpoint, caBundle, skipSSLVerification)
}

func (a *AWS) Upload(filename, bucket, key string) (*s3manager.UploadOutput, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	defer func() {
		err := file.Close()
		if err != nil {
			olog.Printf("[AWS] ‼ Failed to close the file uploaded to S3️: %v", err)
		}
	}()
	return a.UploadFromReader(file, bucket, key)
}

func (a *AWS) UploadFromReader(r io.Reader, bucket, key string) (*s3manager.UploadOutput, error) {
	olog.Printf("[AWS] 🚀 Uploading image to S3: %s/%s", bucket, key)
	return a.s3uploader.Upload(
		context.TODO(),
		&s3.PutObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
			Body:   r,
		},
	)
}

// WaitUntilImportSnapshotCompleted uses the Amazon EC2 API operation
// DescribeImportSnapshots to wait for a condition to be met before returning.
// If the condition is not met within the max attempt window, an error will
// be returned.
func WaitUntilImportSnapshotTaskCompleted(c *ec2.EC2, input *ec2.DescribeImportSnapshotTasksInput) error {
	return WaitUntilImportSnapshotTaskCompletedWithContext(c, aws.BackgroundContext(), input)
}

// WaitUntilImportSnapshotCompletedWithContext is an extended version of
// WaitUntilImportSnapshotCompleted. With the support for passing in a
// context and options to configure the Waiter and the underlying request
// options.
//
// The context must be non-nil and will be used for request cancellation. If
// the context is nil a panic will occur. In the future the SDK may create
// sub-contexts for http.Requests. See https://golang.org/pkg/context/
// for more information on using Contexts.
//
// NOTE(mhayden): The MaxAttempts is set to zero here so that we will keep
// checking the status of the image import until it succeeds or fails. This
// process can take anywhere from 5 to 60+ minutes depending on how quickly
// AWS can import the snapshot.
func WaitUntilImportSnapshotTaskCompletedWithContext(c *ec2.EC2, ctx aws.Context, input *ec2.DescribeImportSnapshotTasksInput, opts ...request.WaiterOption) error {
	w := request.Waiter{
		Name:        "WaitUntilImportSnapshotTaskCompleted",
		MaxAttempts: 0,
		Delay:       request.ConstantWaiterDelay(15 * time.Second),
		Acceptors: []request.WaiterAcceptor{
			{
				State:   request.SuccessWaiterState,
				Matcher: request.PathAllWaiterMatch, Argument: "ImportSnapshotTasks[].SnapshotTaskDetail.Status",
				Expected: "completed",
			},
			{
				State:   request.FailureWaiterState,
				Matcher: request.PathAllWaiterMatch, Argument: "ImportSnapshotTasks[].SnapshotTaskDetail.Status",
				Expected: "deleted",
			},
		},
		Logger: c.Config.Logger,
		NewRequest: func(opts []request.Option) (*request.Request, error) {
			var inCpy *ec2.DescribeImportSnapshotTasksInput
			if input != nil {
				tmp := *input
				inCpy = &tmp
			}
			req, _ := c.DescribeImportSnapshotTasksRequest(inCpy)
			req.SetContext(ctx)
			req.ApplyOptions(opts...)
			return req, nil
		},
	}
	w.ApplyOptions(opts...)

	return w.WaitWithContext(ctx)
}

// Register is a function that imports a snapshot, waits for the snapshot to
// fully import, tags the snapshot, cleans up the image in S3, and registers
// an AMI in AWS.
// The caller can optionally specify the boot mode of the AMI. If the boot
// mode is not specified, then the instances launched from this AMI use the
// default boot mode value of the instance type.
// The caller can also specify the name of the role used to do the import.
// If nil is given, the default one from the SDK is used (vmimport).
// Returns the image ID and the snapshot ID.
//
// XXX: make this return (string, string, error) instead of pointers
func (a *AWS) Register(name, bucket, key string, shareWith []string, rpmArch string, bootMode, importRole *string) (*string, *string, error) {
	rpmArchToEC2Arch := map[string]string{
		"x86_64":  "x86_64",
		"aarch64": "arm64",
	}

	ec2Arch, validArch := rpmArchToEC2Arch[rpmArch]
	if !validArch {
		return nil, nil, fmt.Errorf("ec2 doesn't support the following arch: %s", rpmArch)
	}

	if bootMode != nil {
		if !slices.Contains(ec2.BootModeValues_Values(), *bootMode) {
			return nil, nil, fmt.Errorf("ec2 doesn't support the following boot mode: %s", *bootMode)
		}
	}

	olog.Printf("[AWS] 📥 Importing snapshot from image: %s/%s", bucket, key)
	snapshotDescription := fmt.Sprintf("Image Builder AWS Import of %s", name)
	importTaskOutput, err := a.ec2.ImportSnapshot(
		&ec2.ImportSnapshotInput{
			Description: aws.String(snapshotDescription),
			DiskContainer: &ec2.SnapshotDiskContainer{
				UserBucket: &ec2.UserBucket{
					S3Bucket: aws.String(bucket),
					S3Key:    aws.String(key),
				},
			},
			RoleName: importRole,
		},
	)
	if err != nil {
		olog.Printf("[AWS] error importing snapshot: %s", err)
		return nil, nil, err
	}

	olog.Printf("[AWS] 🚚 Waiting for snapshot to finish importing: %s", *importTaskOutput.ImportTaskId)
	err = WaitUntilImportSnapshotTaskCompleted(
		a.ec2,
		&ec2.DescribeImportSnapshotTasksInput{
			ImportTaskIds: []*string{
				importTaskOutput.ImportTaskId,
			},
		},
	)
	if err != nil {
		return nil, nil, err
	}

	// we no longer need the object in s3, let's just delete it
	olog.Printf("[AWS] 🧹 Deleting image from S3: %s/%s", bucket, key)
	if err = a.DeleteObject(bucket, key); err != nil {
		return nil, nil, err
	}

	importOutput, err := a.ec2.DescribeImportSnapshotTasks(
		&ec2.DescribeImportSnapshotTasksInput{
			ImportTaskIds: []*string{
				importTaskOutput.ImportTaskId,
			},
		},
	)
	if err != nil {
		return nil, nil, err
	}

	snapshotID := importOutput.ImportSnapshotTasks[0].SnapshotTaskDetail.SnapshotId

	// Tag the snapshot with the image name.
	req, _ := a.ec2.CreateTagsRequest(
		&ec2.CreateTagsInput{
			Resources: []*string{snapshotID},
			Tags: []*ec2.Tag{
				{
					Key:   aws.String("Name"),
					Value: aws.String(name),
				},
			},
		},
	)
	err = req.Send()
	if err != nil {
		return nil, nil, err
	}

	olog.Printf("[AWS] 📋 Registering AMI from imported snapshot: %s", *snapshotID)
	registerOutput, err := a.ec2.RegisterImage(
		&ec2.RegisterImageInput{
			Architecture:       aws.String(ec2Arch),
			BootMode:           bootMode,
			VirtualizationType: aws.String("hvm"),
			Name:               aws.String(name),
			RootDeviceName:     aws.String("/dev/sda1"),
			EnaSupport:         aws.Bool(true),
			BlockDeviceMappings: []*ec2.BlockDeviceMapping{
				{
					DeviceName: aws.String("/dev/sda1"),
					Ebs: &ec2.EbsBlockDevice{
						SnapshotId: snapshotID,
					},
				},
			},
		},
	)
	if err != nil {
		return nil, nil, err
	}

	olog.Printf("[AWS] 🎉 AMI registered: %s", *registerOutput.ImageId)

	// Tag the image with the image name.
	req, _ = a.ec2.CreateTagsRequest(
		&ec2.CreateTagsInput{
			Resources: []*string{registerOutput.ImageId},
			Tags: []*ec2.Tag{
				{
					Key:   aws.String("Name"),
					Value: aws.String(name),
				},
			},
		},
	)
	err = req.Send()
	if err != nil {
		return nil, nil, err
	}

	if len(shareWith) > 0 {
		err = a.shareSnapshot(snapshotID, shareWith)
		if err != nil {
			return nil, nil, err
		}
		err = a.shareImage(registerOutput.ImageId, shareWith)
		if err != nil {
			return nil, nil, err
		}
	}

	return registerOutput.ImageId, snapshotID, nil
}

func (a *AWS) DeleteObject(bucket, key string) error {
	_, err := a.s3.DeleteObject(
		context.TODO(),
		&s3.DeleteObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		},
	)
	return err
}

// target region is determined by the region configured in the aws session
func (a *AWS) CopyImage(name, ami, sourceRegion string) (string, error) {
	result, err := a.ec2.CopyImage(
		&ec2.CopyImageInput{
			Name:          aws.String(name),
			SourceImageId: aws.String(ami),
			SourceRegion:  aws.String(sourceRegion),
		},
	)
	if err != nil {
		return "", err
	}

	dIInput := &ec2.DescribeImagesInput{
		ImageIds: []*string{result.ImageId},
	}

	// Custom waiter which waits indefinitely until a final state
	w := request.Waiter{
		Name:        "WaitUntilImageAvailable",
		MaxAttempts: 0,
		Delay:       request.ConstantWaiterDelay(15 * time.Second),
		Acceptors: []request.WaiterAcceptor{
			{
				State:   request.SuccessWaiterState,
				Matcher: request.PathAllWaiterMatch, Argument: "Images[].State",
				Expected: "available",
			},
			{
				State:   request.FailureWaiterState,
				Matcher: request.PathAnyWaiterMatch, Argument: "Images[].State",
				Expected: "failed",
			},
		},
		Logger: a.ec2.Config.Logger,
		NewRequest: func(opts []request.Option) (*request.Request, error) {
			var inCpy *ec2.DescribeImagesInput
			if dIInput != nil {
				tmp := *dIInput
				inCpy = &tmp
			}
			req, _ := a.ec2.DescribeImagesRequest(inCpy)
			req.SetContext(aws.BackgroundContext())
			req.ApplyOptions(opts...)
			return req, nil
		},
	}
	err = w.WaitWithContext(aws.BackgroundContext())
	if err != nil {
		return *result.ImageId, err
	}

	// Tag image with name
	_, err = a.ec2.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{result.ImageId},
		Tags: []*ec2.Tag{
			{
				Key:   aws.String("Name"),
				Value: aws.String(name),
			},
		},
	})

	if err != nil {
		return *result.ImageId, err
	}

	imgs, err := a.ec2.DescribeImages(dIInput)
	if err != nil {
		return *result.ImageId, err
	}
	if len(imgs.Images) == 0 {
		return *result.ImageId, fmt.Errorf("Unable to find image with id: %v", ami)
	}

	// Tag snapshot with name
	for _, bdm := range imgs.Images[0].BlockDeviceMappings {
		_, err = a.ec2.CreateTags(&ec2.CreateTagsInput{
			Resources: []*string{bdm.Ebs.SnapshotId},
			Tags: []*ec2.Tag{
				{
					Key:   aws.String("Name"),
					Value: aws.String(name),
				},
			},
		})
		if err != nil {
			return *result.ImageId, err
		}
	}

	return *result.ImageId, nil
}

func (a *AWS) ShareImage(ami string, userIds []string) error {
	imgs, err := a.ec2.DescribeImages(
		&ec2.DescribeImagesInput{
			ImageIds: []*string{aws.String(ami)},
		},
	)
	if err != nil {
		return err
	}
	if len(imgs.Images) == 0 {
		return fmt.Errorf("Unable to find image with id: %v", ami)
	}

	for _, bdm := range imgs.Images[0].BlockDeviceMappings {
		err = a.shareSnapshot(bdm.Ebs.SnapshotId, userIds)
		if err != nil {
			return err
		}
	}

	err = a.shareImage(aws.String(ami), userIds)
	if err != nil {
		return err
	}
	return nil
}

func (a *AWS) shareImage(ami *string, userIds []string) error {
	olog.Println("[AWS] 🎥 Sharing ec2 snapshot")
	var uIds []*string
	for i := range userIds {
		uIds = append(uIds, &userIds[i])
	}

	olog.Println("[AWS] 💿 Sharing ec2 AMI")
	var launchPerms []*ec2.LaunchPermission
	for _, id := range uIds {
		launchPerms = append(launchPerms, &ec2.LaunchPermission{
			UserId: id,
		})
	}
	_, err := a.ec2.ModifyImageAttribute(
		&ec2.ModifyImageAttributeInput{
			ImageId: ami,
			LaunchPermission: &ec2.LaunchPermissionModifications{
				Add: launchPerms,
			},
		},
	)
	if err != nil {
		olog.Printf("[AWS] 📨 Error sharing AMI: %v", err)
		return err
	}
	olog.Println("[AWS] 💿 Shared AMI")
	return nil
}

func (a *AWS) shareSnapshot(snapshotId *string, userIds []string) error {
	olog.Println("[AWS] 🎥 Sharing ec2 snapshot")
	var uIds []*string
	for i := range userIds {
		uIds = append(uIds, &userIds[i])
	}
	_, err := a.ec2.ModifySnapshotAttribute(
		&ec2.ModifySnapshotAttributeInput{
			Attribute:     aws.String(ec2.SnapshotAttributeNameCreateVolumePermission),
			OperationType: aws.String("add"),
			SnapshotId:    snapshotId,
			UserIds:       uIds,
		},
	)
	if err != nil {
		olog.Printf("[AWS] 📨 Error sharing ec2 snapshot: %v", err)
		return err
	}
	olog.Println("[AWS] 📨 Shared ec2 snapshot")
	return nil
}

func (a *AWS) RemoveSnapshotAndDeregisterImage(image *ec2.Image) error {
	if image == nil {
		return fmt.Errorf("image is nil")
	}

	var snapshots []*string
	for _, bdm := range image.BlockDeviceMappings {
		snapshots = append(snapshots, bdm.Ebs.SnapshotId)
	}

	_, err := a.ec2.DeregisterImage(
		&ec2.DeregisterImageInput{
			ImageId: image.ImageId,
		},
	)
	if err != nil {
		return err
	}

	for _, s := range snapshots {
		_, err = a.ec2.DeleteSnapshot(
			&ec2.DeleteSnapshotInput{
				SnapshotId: s,
			},
		)
		if err != nil {
			// TODO return err?
			olog.Println("Unable to remove snapshot", s)
		}
	}
	return err
}

// For service maintenance images are discovered by the "Name:composer-api-*" tag filter. Currently
// all image names in the service are generated, so they're guaranteed to be unique as well. If
// users are ever allowed to name their images, an extra tag should be added.
func (a *AWS) DescribeImagesByTag(tagKey, tagValue string) ([]*ec2.Image, error) {
	imgs, err := a.ec2.DescribeImages(
		&ec2.DescribeImagesInput{
			Filters: []*ec2.Filter{
				{
					Name:   aws.String(fmt.Sprintf("tag:%s", tagKey)),
					Values: []*string{aws.String(tagValue)},
				},
			},
		},
	)
	return imgs.Images, err
}

func (a *AWS) S3ObjectPresignedURL(bucket, objectKey string) (string, error) {
	olog.Printf("[AWS] 📋 Generating Presigned URL for S3 object %s/%s", bucket, objectKey)
	req, err := a.s3presign.PresignGetObject(
		context.TODO(),
		&s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(objectKey),
		},
		func(opts *s3.PresignOptions) {
			opts.Expires = time.Duration(7 * 24 * time.Hour)
		},
	)
	if err != nil {
		return "", err
	}

	olog.Println("[AWS] 🎉 S3 Presigned URL ready")
	return req.URL, nil
}

func (a *AWS) MarkS3ObjectAsPublic(bucket, objectKey string) error {
	olog.Printf("[AWS] 👐 Making S3 object public %s/%s", bucket, objectKey)
	_, err := a.s3.PutObjectAcl(
		context.TODO(),
		&s3.PutObjectAclInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(objectKey),
			ACL:    s3types.ObjectCannedACL(s3types.ObjectCannedACLPublicRead),
		},
	)
	if err != nil {
		return err
	}

	olog.Println("[AWS] ✔️ Making S3 object public successful")
	return nil
}

func (a *AWS) Regions() ([]string, error) {
	out, err := a.ec2.DescribeRegions(&ec2.DescribeRegionsInput{})
	if err != nil {
		return nil, err
	}

	result := []string{}
	for _, r := range out.Regions {
		result = append(result, aws.StringValue(r.RegionName))
	}

	return result, nil
}

func (a *AWS) Buckets() ([]string, error) {
	out, err := a.s3.ListBuckets(
		context.TODO(),
		nil,
	)
	if err != nil {
		return nil, err
	}

	result := []string{}
	for _, b := range out.Buckets {
		result = append(result, aws.StringValue(b.Name))
	}

	return result, nil
}

// checkAWSPermissionMatrix internal helper function, checks if the requiredPermission is
// covered by the currentPermission (consulting the PermissionsMatrix)
func checkAWSPermissionMatrix(requiredPermission s3types.Permission, currentPermission s3types.Permission) bool {
	requiredPermissions, exists := S3PermissionsMatrix[requiredPermission]
	if !exists {
		return false
	}

	for _, permission := range requiredPermissions {
		if permission == currentPermission {
			return true
		}
	}
	return false
}

// CheckBucketPermission check if the current account (of a.s3) has the `permission` on the given bucket
func (a *AWS) CheckBucketPermission(bucketName string, permission s3types.Permission) (bool, error) {
	resp, err := a.s3.GetBucketAcl(
		context.TODO(),
		&s3.GetBucketAclInput{
			Bucket: aws.String(bucketName),
		},
	)
	if err != nil {
		return false, err
	}

	for _, grant := range resp.Grants {
		if checkAWSPermissionMatrix(permission, grant.Permission) {
			return true, nil
		}
	}
	return false, nil
}

func (a *AWS) CreateSecurityGroupEC2(name, description string) (*ec2.CreateSecurityGroupOutput, error) {
	return a.ec2.CreateSecurityGroup(&ec2.CreateSecurityGroupInput{
		GroupName:   aws.String(name),
		Description: aws.String(description),
	})
}

func (a *AWS) DeleteSecurityGroupEC2(groupID *string) (*ec2.DeleteSecurityGroupOutput, error) {
	return a.ec2.DeleteSecurityGroup(&ec2.DeleteSecurityGroupInput{
		GroupId: groupID,
	})
}

func (a *AWS) AuthorizeSecurityGroupIngressEC2(groupID *string, address string, from, to int64, proto string) (*ec2.AuthorizeSecurityGroupIngressOutput, error) {
	return a.ec2.AuthorizeSecurityGroupIngress(&ec2.AuthorizeSecurityGroupIngressInput{
		CidrIp:     aws.String(address),
		GroupId:    groupID,
		FromPort:   aws.Int64(from),
		ToPort:     aws.Int64(to),
		IpProtocol: aws.String(proto),
	})
}

func (a *AWS) RunInstanceEC2(imageID, secGroupID *string, userData, instanceType string) (*ec2.Reservation, error) {
	reservation, err := a.ec2.RunInstances(&ec2.RunInstancesInput{
		MaxCount:         aws.Int64(1),
		MinCount:         aws.Int64(1),
		ImageId:          imageID,
		InstanceType:     aws.String(instanceType),
		SecurityGroupIds: []*string{secGroupID},
		UserData:         aws.String(encodeBase64(userData)),
	})
	if err != nil {
		return nil, err
	}

	if err := a.ec2.WaitUntilInstanceRunning(describeInstanceInput(reservation.Instances[0].InstanceId)); err != nil {
		return nil, err
	}
	return reservation, nil
}

func (a *AWS) TerminateInstanceEC2(instanceID *string) (*ec2.TerminateInstancesOutput, error) {
	// We need to terminate the instance now and wait until the termination is done.
	// Otherwise, it wouldn't be possible to delete the image.
	res, err := a.ec2.TerminateInstances(&ec2.TerminateInstancesInput{
		InstanceIds: []*string{
			instanceID,
		},
	})
	if err != nil {
		return nil, err
	}

	if err := a.ec2.WaitUntilInstanceTerminated(describeInstanceInput(instanceID)); err != nil {
		return nil, err
	}
	return res, nil
}

func (a *AWS) GetInstanceAddress(instanceID *string) (string, error) {
	desc, err := a.ec2.DescribeInstances(describeInstanceInput(instanceID))
	if err != nil {
		return "", err
	}

	return *desc.Reservations[0].Instances[0].PublicIpAddress, nil
}

// DeleteEC2Image deletes the specified image and its associated snapshot
func (a *AWS) DeleteEC2Image(imageID, snapshotID *string) error {
	var retErr error

	// firstly, deregister the image
	_, err := a.ec2.DeregisterImage(&ec2.DeregisterImageInput{
		ImageId: imageID,
	})

	if err != nil {
		return err
	}

	// now it's possible to delete the snapshot
	_, err = a.ec2.DeleteSnapshot(&ec2.DeleteSnapshotInput{
		SnapshotId: snapshotID,
	})

	if err != nil {
		return err
	}

	return retErr
}

// encodeBase64 encodes string to base64-encoded string
func encodeBase64(input string) string {
	return base64.StdEncoding.EncodeToString([]byte(input))
}

func describeInstanceInput(id *string) *ec2.DescribeInstancesInput {
	return &ec2.DescribeInstancesInput{
		InstanceIds: []*string{id},
	}
}
