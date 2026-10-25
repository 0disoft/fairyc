package pecoff

import "fmt"

// Zero is unclaimed, negative owners are MSF metadata, and positive owners are
// one-based stream indexes. The directory validates at most 65,536 streams.
type pdbBlockOwner int32

const (
	pdbSuperblockOwner      pdbBlockOwner = -1
	pdbDirectoryMapOwner    pdbBlockOwner = -2
	pdbStreamDirectoryOwner pdbBlockOwner = -3
)

func (owner pdbBlockOwner) label() string {
	switch owner {
	case pdbSuperblockOwner:
		return "superblock"
	case pdbDirectoryMapOwner:
		return "directory block map"
	case pdbStreamDirectoryOwner:
		return "stream directory"
	default:
		return fmt.Sprintf("stream %d", owner-1)
	}
}
