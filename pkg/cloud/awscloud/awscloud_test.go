package awscloud_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/stretchr/testify/require"

	"github.com/osbuild/images/pkg/cloud/awscloud"
)

func TestS3MarkObjectAsPublic(t *testing.T) {
	fc := &fakeS3Client{}
	aws := awscloud.NewAWSForTest(fc, nil, nil)
	require.NotNil(t, aws)

	require.NoError(t, aws.MarkS3ObjectAsPublic("bucket", "object-key"))
	require.Len(t, fc.putObjectAclCalls, 1)
	require.Equal(t, "bucket", *fc.putObjectAclCalls[0].Bucket)
	require.Equal(t, "object-key", *fc.putObjectAclCalls[0].Key)
	require.Equal(t, s3types.ObjectCannedACLPublicRead, fc.putObjectAclCalls[0].ACL)
}

func TestS3MarkObjectAsPublicError(t *testing.T) {
	fc := &fakeS3Client{
		putObjectAclErr: fmt.Errorf("error marking object as public"),
	}
	aws := awscloud.NewAWSForTest(fc, nil, nil)
	require.NotNil(t, aws)

	err := aws.MarkS3ObjectAsPublic("bucket", "object-key")
	require.Error(t, err)
	require.Len(t, fc.putObjectAclCalls, 1)
	require.Equal(t, "bucket", *fc.putObjectAclCalls[0].Bucket)
	require.Equal(t, "object-key", *fc.putObjectAclCalls[0].Key)
}

func TestS3Upload(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFilePath := filepath.Join(tmpDir, "file")
	require.NoError(t, os.WriteFile(tmpFilePath, []byte("test content"), 0600))
	fm := &fakeS3Uploader{}
	aws := awscloud.NewAWSForTest(nil, fm, nil)
	require.NotNil(t, aws)

	uo, err := aws.Upload(tmpFilePath, "bucket", "object-key")
	require.NoError(t, err)
	require.Len(t, fm.uploadCalls, 1)
	require.Equal(t, "bucket", *fm.uploadCalls[0].Bucket)
	require.Equal(t, "object-key", *fm.uploadCalls[0].Key)
	require.Equal(t, "object-key", *uo.Key)
}

func TestS3UploadError(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFilePath := filepath.Join(tmpDir, "file")
	require.NoError(t, os.WriteFile(tmpFilePath, []byte("test content"), 0600))
	fm := &fakeS3Uploader{
		uploadErr: fmt.Errorf("upload error"),
	}
	aws := awscloud.NewAWSForTest(nil, fm, nil)
	require.NotNil(t, aws)

	_, err := aws.Upload(tmpFilePath, "bucket", "object-key")
	require.Error(t, err)
	require.Len(t, fm.uploadCalls, 1)
	require.Equal(t, "bucket", *fm.uploadCalls[0].Bucket)
	require.Equal(t, "object-key", *fm.uploadCalls[0].Key)
}

func TestS3ObjectPresignedURL(t *testing.T) {
	fs := &fakeS3Presign{}
	aws := awscloud.NewAWSForTest(nil, nil, fs)
	require.NotNil(t, aws)

	url, err := aws.S3ObjectPresignedURL("bucket", "object-key")
	require.NoError(t, err)
	require.Len(t, fs.presignGetObjectCalls, 1)
	require.Equal(t, "bucket", *fs.presignGetObjectCalls[0].Bucket)
	require.Equal(t, "object-key", *fs.presignGetObjectCalls[0].Key)
	require.Equal(t, "https://bucket.s3.amazonaws.com/object-key", url)
}

func TestS3ObjectPresignedURLError(t *testing.T) {
	fs := &fakeS3Presign{
		presignGetObjectErr: fmt.Errorf("presign error"),
	}
	aws := awscloud.NewAWSForTest(nil, nil, fs)
	require.NotNil(t, aws)

	_, err := aws.S3ObjectPresignedURL("bucket", "object-key")
	require.Error(t, err)
	require.Len(t, fs.presignGetObjectCalls, 1)
	require.Equal(t, "bucket", *fs.presignGetObjectCalls[0].Bucket)
	require.Equal(t, "object-key", *fs.presignGetObjectCalls[0].Key)
}

func TestDeleteObject(t *testing.T) {
	fc := &fakeS3Client{}
	aws := awscloud.NewAWSForTest(fc, nil, nil)
	require.NotNil(t, aws)

	require.NoError(t, aws.DeleteObject("bucket", "object-key"))
	require.Len(t, fc.deleteObjectCalls, 1)
	require.Equal(t, "bucket", *fc.deleteObjectCalls[0].Bucket)
	require.Equal(t, "object-key", *fc.deleteObjectCalls[0].Key)
}

func TestDeleteObjectError(t *testing.T) {
	fc := &fakeS3Client{
		deleteObjectErr: fmt.Errorf("error deleting object"),
	}
	aws := awscloud.NewAWSForTest(fc, nil, nil)
	require.NotNil(t, aws)

	err := aws.DeleteObject("bucket", "object-key")
	require.Error(t, err)
	require.Len(t, fc.deleteObjectCalls, 1)
	require.Equal(t, "bucket", *fc.deleteObjectCalls[0].Bucket)
	require.Equal(t, "object-key", *fc.deleteObjectCalls[0].Key)
}

func TestBuckets(t *testing.T) {
	fc := &fakeS3Client{
		buckets: []string{"bucket1", "bucket2"},
	}
	aws := awscloud.NewAWSForTest(fc, nil, nil)
	require.NotNil(t, aws)

	buckets, err := aws.Buckets()
	require.NoError(t, err)
	require.Len(t, buckets, 2)
	require.Equal(t, "bucket1", buckets[0])
	require.Equal(t, "bucket2", buckets[1])
	require.Equal(t, 1, fc.listBucketsCalls)
}

func TestBucketsError(t *testing.T) {
	fc := &fakeS3Client{
		listBucketsErr: fmt.Errorf("error listing buckets"),
	}
	aws := awscloud.NewAWSForTest(fc, nil, nil)
	require.NotNil(t, aws)

	_, err := aws.Buckets()
	require.Error(t, err)
	require.Equal(t, 1, fc.listBucketsCalls)
}

func TestCheckBucketPermission(t *testing.T) {
	type testCase struct {
		name       string
		fc         *fakeS3Client
		permission s3types.Permission
		expected   bool
		expectErr  bool
	}
	testCases := []testCase{
		{
			name: "happy path",
			fc: &fakeS3Client{
				bucketAcl: &s3.GetBucketAclOutput{
					Grants: []s3types.Grant{
						{
							Permission: s3types.PermissionRead,
						},
					},
				},
			},
			permission: s3types.PermissionRead,
			expected:   true,
		},
		{
			name: "permission not granted",
			fc: &fakeS3Client{
				bucketAcl: &s3.GetBucketAclOutput{
					Grants: []s3types.Grant{
						{
							Permission: s3types.PermissionRead,
						},
					},
				},
			},
			permission: s3types.PermissionWrite,
			expected:   false,
		},
		{
			name: "permissions covered by higher level permissions",
			fc: &fakeS3Client{
				bucketAcl: &s3.GetBucketAclOutput{
					Grants: []s3types.Grant{
						{
							Permission: s3types.PermissionFullControl,
						},
					},
				},
			},
			permission: s3types.PermissionWrite,
			expected:   true,
		},
		{
			name: "invalid permission",
			fc: &fakeS3Client{
				bucketAcl: &s3.GetBucketAclOutput{
					Grants: []s3types.Grant{
						{
							Permission: s3types.PermissionRead,
						},
					},
				},
			},
			permission: s3types.Permission("invalid-permission"),
			expected:   false,
		},
		{
			name: "error retrieving bucket ACL",
			fc: &fakeS3Client{
				getBucketAclErr: fmt.Errorf("error retrieving bucket ACL"),
			},
			permission: s3types.PermissionRead,
			expectErr:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			aws := awscloud.NewAWSForTest(tc.fc, nil, nil)
			require.NotNil(t, aws)

			result, err := aws.CheckBucketPermission("bucket", tc.permission)

			require.Len(t, tc.fc.getBucketAclCalls, 1)
			require.Equal(t, "bucket", *tc.fc.getBucketAclCalls[0].Bucket)

			if tc.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.expected, result)
		})
	}
}
