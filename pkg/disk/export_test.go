package disk

var (
	PayloadEntityMap         = payloadEntityMap
	EntityPath               = entityPath
	AddBootPartition         = addBootPartition
	AddPartitionsForBootMode = addPartitionsForBootMode
)

func FindDirectoryEntityPath(pt *PartitionTable, path string) []Entity {
	return pt.findDirectoryEntityPath(path)
}
