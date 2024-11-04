package disk

var (
	PayloadEntityMap = payloadEntityMap
	EntityPath       = entityPath
)

func FindDirectoryEntityPath(pt *PartitionTable, path string) []Entity {
	return pt.findDirectoryEntityPath(path)
}
