package filemonitor

const (
	createFlag = 1 << iota
	deleteFlag
	renameFlag
	writeFlag
	permissionFlag
	createSentence     = "create"
	deleteSentence     = "delete"
	renameSentence     = "rename"
	writeSentence      = "write"
	permissionSentence = "permission"
)
