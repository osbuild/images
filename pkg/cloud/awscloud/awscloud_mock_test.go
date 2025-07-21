package awscloud_test

import (
	"context"
	"fmt"

	awsSigner "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/osbuild/images/pkg/cloud/awscloud"
)

type fakeS3Client struct {
	deleteObjectCalls []s3.DeleteObjectInput
	deleteObjectErr   error

	listBucketsCalls int
	buckets          []string
	listBucketsErr   error

	putObjectAclCalls []s3.PutObjectAclInput
	putObjectAclErr   error

	getBucketAclCalls []s3.GetBucketAclInput
	bucketAcl         *s3.GetBucketAclOutput
	getBucketAclErr   error
}

var _ awscloud.S3Client = (*fakeS3Client)(nil)

func (f *fakeS3Client) DeleteObject(ctx context.Context, input *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	f.deleteObjectCalls = append(f.deleteObjectCalls, *input)
	if f.deleteObjectErr != nil {
		return nil, f.deleteObjectErr
	}
	return &s3.DeleteObjectOutput{}, nil
}

func (f *fakeS3Client) ListBuckets(ctx context.Context, input *s3.ListBucketsInput, optFns ...func(*s3.Options)) (*s3.ListBucketsOutput, error) {
	f.listBucketsCalls++
	if f.listBucketsErr != nil {
		return nil, f.listBucketsErr
	}
	bkts := make([]s3types.Bucket, len(f.buckets))
	for i, b := range f.buckets {
		bkts[i] = s3types.Bucket{
			Name: &b,
		}
	}
	return &s3.ListBucketsOutput{
		Buckets: bkts,
	}, nil
}

func (f *fakeS3Client) PutObjectAcl(ctx context.Context, input *s3.PutObjectAclInput, optFns ...func(*s3.Options)) (*s3.PutObjectAclOutput, error) {
	f.putObjectAclCalls = append(f.putObjectAclCalls, *input)
	if f.putObjectAclErr != nil {
		return nil, f.putObjectAclErr
	}
	return &s3.PutObjectAclOutput{}, nil
}

func (f *fakeS3Client) GetBucketAcl(ctx context.Context, input *s3.GetBucketAclInput, optFns ...func(*s3.Options)) (*s3.GetBucketAclOutput, error) {
	f.getBucketAclCalls = append(f.getBucketAclCalls, *input)
	if f.getBucketAclErr != nil {
		return nil, f.getBucketAclErr
	}
	return f.bucketAcl, nil
}

type fakeS3Uploader struct {
	uploadCalls []s3.PutObjectInput
	uploadErr   error
}

var _ awscloud.S3Uploader = (*fakeS3Uploader)(nil)

func (f *fakeS3Uploader) Upload(ctx context.Context, input *s3.PutObjectInput, optFns ...func(*manager.Uploader)) (*manager.UploadOutput, error) {
	f.uploadCalls = append(f.uploadCalls, *input)
	if f.uploadErr != nil {
		return nil, f.uploadErr
	}
	return &manager.UploadOutput{
		Key: input.Key,
	}, nil
}

type fakeS3Presign struct {
	presignGetObjectCalls []s3.GetObjectInput
	presignGetObjectErr   error
}

var _ awscloud.S3Presign = (*fakeS3Presign)(nil)

func (f *fakeS3Presign) PresignGetObject(ctx context.Context, input *s3.GetObjectInput, optFns ...func(*s3.PresignOptions)) (*awsSigner.PresignedHTTPRequest, error) {
	f.presignGetObjectCalls = append(f.presignGetObjectCalls, *input)
	if f.presignGetObjectErr != nil {
		return nil, f.presignGetObjectErr
	}
	return &awsSigner.PresignedHTTPRequest{
		URL: fmt.Sprintf("https://%s.s3.amazonaws.com/%s", *input.Bucket, *input.Key),
	}, nil
}
