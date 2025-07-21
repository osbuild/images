package awscloud

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

type fakeNewSnapshotImportedWaiterEC2 struct {
	returnDescribeImportSnapshotTasksOutput *ec2.DescribeImportSnapshotTasksOutput
	returnDescribeImportSnapshotTasksErr    error
}

func (f *fakeNewSnapshotImportedWaiterEC2) WaitForOutput(ctx context.Context, params *ec2.DescribeImportSnapshotTasksInput, maxWaitDur time.Duration, optFns ...func(*ec2.SnapshotImportedWaiterOptions)) (*ec2.DescribeImportSnapshotTasksOutput, error) {
	if f.returnDescribeImportSnapshotTasksErr != nil {
		return nil, f.returnDescribeImportSnapshotTasksErr
	}
	return f.returnDescribeImportSnapshotTasksOutput, nil
}

func MockNewSnapshotImportedWaiterEC2(out *ec2.DescribeImportSnapshotTasksOutput, err error) (restore func()) {
	original := newSnapshotImportedWaiterEC2
	newSnapshotImportedWaiterEC2 = func(client ec2.DescribeImportSnapshotTasksAPIClient, optFns ...func(*ec2.SnapshotImportedWaiterOptions)) snapshotImportedWaiterEC2 {
		return &fakeNewSnapshotImportedWaiterEC2{
			returnDescribeImportSnapshotTasksOutput: out,
			returnDescribeImportSnapshotTasksErr:    err,
		}
	}

	return func() {
		newSnapshotImportedWaiterEC2 = original
	}
}

type fakeNewInstanceRunningWaiterEC2 struct {
	returnDescribeInstancesErr error
}

func (f *fakeNewInstanceRunningWaiterEC2) Wait(ctx context.Context, params *ec2.DescribeInstancesInput, maxWaitDur time.Duration, optFns ...func(*ec2.InstanceRunningWaiterOptions)) error {
	if f.returnDescribeInstancesErr != nil {
		return f.returnDescribeInstancesErr
	}
	return nil
}

func MockNewInstanceRunningWaiterEC2(err error) (restore func()) {
	original := newInstanceRunningWaiterEC2
	newInstanceRunningWaiterEC2 = func(client ec2.DescribeInstancesAPIClient, optFns ...func(*ec2.InstanceRunningWaiterOptions)) instanceRunningWaiterEC2 {
		return &fakeNewInstanceRunningWaiterEC2{

			returnDescribeInstancesErr: err,
		}
	}

	return func() {
		newInstanceRunningWaiterEC2 = original
	}
}

type fakeNewTerminateInstancesWaiterEC2 struct {
	returnDescribeInstancesErr error
}

func (f *fakeNewTerminateInstancesWaiterEC2) Wait(ctx context.Context, params *ec2.DescribeInstancesInput, maxWaitDur time.Duration, optFns ...func(*ec2.InstanceTerminatedWaiterOptions)) error {
	if f.returnDescribeInstancesErr != nil {
		return f.returnDescribeInstancesErr
	}
	return nil
}

func MockNewTerminateInstancesWaiterEC2(err error) (restore func()) {
	original := newTerminateInstancesWaiterEC2
	newTerminateInstancesWaiterEC2 = func(client ec2.DescribeInstancesAPIClient, optFns ...func(*ec2.InstanceTerminatedWaiterOptions)) instanceTerminatedWaiterEC2 {
		return &fakeNewTerminateInstancesWaiterEC2{
			returnDescribeInstancesErr: err,
		}
	}

	return func() {
		newTerminateInstancesWaiterEC2 = original
	}
}
