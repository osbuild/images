package disk

var (
	PayloadEntityMap         = payloadEntityMap
	EntityPath               = entityPath
	AddBootPartition         = addBootPartition
	AddPartitionsForBootMode = addPartitionsForBootMode
)

type PartitionTableFeatures = partitionTableFeatures

func FindDirectoryEntityPath(pt *PartitionTable, path string) []Entity {
	return pt.findDirectoryEntityPath(path)
}

func GetPartitionTableFeatures(pt PartitionTable) PartitionTableFeatures {
	return pt.features()
}
