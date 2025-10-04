package ibmcloud

import (
	"fmt"
	"io"
	"os"

	"github.com/IBM/ibm-cos-sdk-go/service/s3/s3manager"

	"github.com/osbuild/images/pkg/cloud"

	"github.com/IBM/ibm-cos-sdk-go/aws"
	"github.com/IBM/ibm-cos-sdk-go/aws/credentials/ibmiam"
	"github.com/IBM/ibm-cos-sdk-go/aws/session"
)

var _ = cloud.Uploader(&ibmcloudUploader{})

type ibmcloudUploader struct {
	region     string
	bucketName string
	imageName  string
}

func NewUploader(region string, bucketName string, imageName string) (cloud.Uploader, error) {
	return &ibmcloudUploader{
		region:     region,
		bucketName: bucketName,
		imageName:  imageName,
	}, nil
}

func (iu *ibmcloudUploader) Check(status io.Writer) error {
	return nil
}

func (iu *ibmcloudUploader) UploadAndRegister(r io.Reader, uploadSize uint64, status io.Writer) (err error) {
	fmt.Fprintf(status, "Uploading to IBM Cloud...\n")

	apiKey := os.Getenv("IBMCLOUD_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("Please set your IBM Cloud API key as $IBMCLOUD_API_KEY")
	}

	crn := os.Getenv("IBMCLOUD_CRN")
	if crn == "" {
		return fmt.Errorf("Please set your IBM Cloud Resource Name as $IBMCLOUD_CRN")
	}

	endpoint := fmt.Sprintf("s3.%s.cloud-object-storage.appdomain.cloud", iu.region)
	credentials := ibmiam.NewStaticCredentials(
		aws.NewConfig(),
		"https://iam.cloud.ibm.com/identity/token",
		apiKey,
		crn,
	)
	conf := aws.NewConfig().
		WithRegion(iu.region).
		WithEndpoint(endpoint).
		WithCredentials(credentials).
		WithS3ForcePathStyle(true)
	session := session.Must(session.NewSession(conf))
	uploader := s3manager.NewUploader(session)

	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(iu.bucketName),
		Key:    aws.String(iu.imageName),
		Body:   r,
	})
	if err != nil {
		return fmt.Errorf("Failed to upload: %w", err)
	}

	return nil
}
