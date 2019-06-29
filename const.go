package filemonitor

const (
	createFlag = 1 << iota
	removeFlag
	renameFlag
	writeFlag
	permissionFlag
	createSentence     = "create"
	removeSentence     = "remove"
	renameSentence     = "rename"
	writeSentence      = "write"
	permissionSentence = "permission"
)
