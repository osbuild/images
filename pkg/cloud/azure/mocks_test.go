package azure_test

import (
	"context"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"

	"github.com/osbuild/images/internal/common"
)

type mockPollerHandler[T any] struct {
	result *T
}

func (mp *mockPollerHandler[T]) Done() bool {
	return true
}

func (mp *mockPollerHandler[T]) Poll(ctx context.Context) (*http.Response, error) {
	return nil, nil
}

func (mp *mockPollerHandler[T]) Result(ctx context.Context, out *T) error {
	return nil
}

type resourcesMock struct {
	list []rmListArgs
}

type rmListArgs struct {
	rg      string
	options *armresources.ClientListByResourceGroupOptions
}

func (rm *resourcesMock) NewListByResourceGroupPager(
	rg string,
	options *armresources.ClientListByResourceGroupOptions) *runtime.Pager[armresources.ClientListByResourceGroupResponse] {
	rm.list = append(rm.list, rmListArgs{rg, options})

	return runtime.NewPager(
		runtime.PagingHandler[armresources.ClientListByResourceGroupResponse]{
			More: func(current armresources.ClientListByResourceGroupResponse) bool {
				return false
			},
			Fetcher: func(ctx context.Context, current *armresources.ClientListByResourceGroupResponse) (armresources.ClientListByResourceGroupResponse, error) {
				return armresources.ClientListByResourceGroupResponse{
					ResourceListResult: armresources.ResourceListResult{
						Value: []*armresources.GenericResourceExpanded{
							&armresources.GenericResourceExpanded{
								Name: common.ToPtr("storage-account"),
							},
						},
					},
				}, nil
			},
		},
	)
}

type resourceGroupsMock struct {
	get []rgmGetArgs
}

type rgmGetArgs struct {
	rg      string
	options *armresources.ResourceGroupsClientGetOptions
}

func (rgm *resourceGroupsMock) Get(
	ctx context.Context,
	rg string,
	options *armresources.ResourceGroupsClientGetOptions) (armresources.ResourceGroupsClientGetResponse, error) {
	rgm.get = append(rgm.get, rgmGetArgs{rg, options})

	return armresources.ResourceGroupsClientGetResponse{
		ResourceGroup: armresources.ResourceGroup{
			Location: common.ToPtr("test-universe"),
		},
	}, nil
}

type accountsMock struct {
	beginCreate []acmBeginCreateArgs
	listKeys    []acmListKeysArgs
}

type acmBeginCreateArgs struct {
	rg      string
	account string
	params  armstorage.AccountCreateParameters
	options *armstorage.AccountsClientBeginCreateOptions
}

func (acm *accountsMock) BeginCreate(
	ctx context.Context,
	rg string,
	account string,
	params armstorage.AccountCreateParameters,
	options *armstorage.AccountsClientBeginCreateOptions) (*runtime.Poller[armstorage.AccountsClientCreateResponse], error) {
	acm.beginCreate = append(acm.beginCreate, acmBeginCreateArgs{rg, account, params, options})

	p, err := runtime.NewPoller(
		&http.Response{},
		runtime.NewPipeline("", "", runtime.PipelineOptions{}, nil),
		&runtime.NewPollerOptions[armstorage.AccountsClientCreateResponse]{
			Handler: &mockPollerHandler[armstorage.AccountsClientCreateResponse]{
				result: &armstorage.AccountsClientCreateResponse{
					Account: armstorage.Account{
						Name: &account,
					},
				},
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return p, nil
}

type acmListKeysArgs struct {
	rg      string
	account string
	options *armstorage.AccountsClientListKeysOptions
}

func (acm *accountsMock) ListKeys(
	ctx context.Context,
	rg string,
	account string,
	options *armstorage.AccountsClientListKeysOptions) (armstorage.AccountsClientListKeysResponse, error) {
	acm.listKeys = append(acm.listKeys, acmListKeysArgs{rg, account, options})

	return armstorage.AccountsClientListKeysResponse{
		AccountListKeysResult: armstorage.AccountListKeysResult{
			Keys: []*armstorage.AccountKey{
				&armstorage.AccountKey{
					Value: common.ToPtr("real key"),
				},
			},
		},
	}, nil
}

type imagesMock struct {
	createOrUpdate []imBeginCreateOrUpdateArgs
}

type imBeginCreateOrUpdateArgs struct {
	rg      string
	name    string
	img     armcompute.Image
	options *armcompute.ImagesClientBeginCreateOrUpdateOptions
}

func (im *imagesMock) BeginCreateOrUpdate(ctx context.Context, rg string, name string, img armcompute.Image, options *armcompute.ImagesClientBeginCreateOrUpdateOptions) (*runtime.Poller[armcompute.ImagesClientCreateOrUpdateResponse], error) {
	im.createOrUpdate = append(im.createOrUpdate, imBeginCreateOrUpdateArgs{rg, name, img, options})

	p, err := runtime.NewPoller(
		&http.Response{},
		runtime.NewPipeline("", "", runtime.PipelineOptions{}, nil),
		&runtime.NewPollerOptions[armcompute.ImagesClientCreateOrUpdateResponse]{
			Handler: &mockPollerHandler[armcompute.ImagesClientCreateOrUpdateResponse]{
				result: &armcompute.ImagesClientCreateOrUpdateResponse{
					Image: img,
				},
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return p, nil
}
