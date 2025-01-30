package awscloud

type AwsClient = awsClient

func MockNewAwsClient(f func(string) (awsClient, error)) (restore func()) {
	saved := newAwsClient
	newAwsClient = f
	return func() {
		newAwsClient = saved
	}
}
