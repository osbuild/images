package awscloud

type AwsClient = awsClient
type EC2Client = ec2Client
type S3Client = s3Client
type S3Uploader = s3Uploader
type S3Presign = s3Presign

func NewAWSForTest(ec2cli EC2Client, s3cli S3Client, upldr S3Uploader, sign S3Presign) *AWS {
	return &AWS{
		ec2:        ec2cli,
		s3:         s3cli,
		s3uploader: upldr,
		s3presign:  sign,
	}
}

func MockNewAwsClient(f func(string) (awsClient, error)) (restore func()) {
	saved := newAwsClient
	newAwsClient = f
	return func() {
		newAwsClient = saved
	}
}
