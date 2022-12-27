package domain

type Operation string

const (
	Download Operation = "download"
	Backup   Operation = "backup"
	Restore  Operation = "restore"
)
