package container

func NewResolverWithTestClient(arch string, f func(string) (*Client, error)) *Resolver {
	resolver := NewResolver(arch)
	resolver.newClient = f
	return resolver
}

func NewClientWithTestStorage(target, storage string) (*Client, error) {
	client, err := NewClient(target)
	client.store = storage
	return client, err
}
