package henchman

var (
	DebugFlag bool
	OsNames   = []string{"darwin", "linux"}
)

const (
	HENCHMAN_PREFIX = "henchman_"
	MODULES_TARGET  = "modules.tar"
	IGNORED_EXTS    = "zip,tar,tar.gz"
	REMOTE_DIR      = "${HOME}/.henchman/"
)
